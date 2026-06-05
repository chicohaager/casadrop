package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"casadrop/internal/models"
)

// Service handles email sending
type Service struct {
	config *models.SMTPConfig
}

// NewService creates a new email service
func NewService(config *models.SMTPConfig) *Service {
	return &Service{config: config}
}

// UpdateConfig updates the SMTP configuration
func (s *Service) UpdateConfig(config *models.SMTPConfig) {
	s.config = config
}

// IsEnabled returns true if email service is configured and enabled
func (s *Service) IsEnabled() bool {
	return s.config != nil && s.config.Enabled && s.config.Host != ""
}

// TestConnection tests the SMTP connection
func (s *Service) TestConnection() error {
	if !s.IsEnabled() {
		return fmt.Errorf("email service is not configured")
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	var client *smtp.Client
	var err error

	if s.config.UseTLS {
		// Direct TLS connection (port 465)
		tlsConfig := &tls.Config{
			ServerName: s.config.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS connection failed: %w", err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, s.config.Host)
		if err != nil {
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
	} else {
		// Plain connection (with optional STARTTLS)
		client, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}

		if s.config.UseStartTLS {
			tlsConfig := &tls.Config{
				ServerName: s.config.Host,
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				client.Close()
				return fmt.Errorf("STARTTLS failed: %w", err)
			}
		}
	}
	defer client.Close()

	// Authenticate if credentials provided
	if s.config.Username != "" && s.config.Password != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	return client.Quit()
}

// SendTransferEmail sends an email with the download link
func (s *Service) SendTransferEmail(transfer *models.EmailTransfer, downloadURL string, fileName string, fileSize string) error {
	if !s.IsEnabled() {
		return fmt.Errorf("email service is not configured")
	}

	subject := transfer.Title
	if subject == "" {
		if transfer.SenderName != "" {
			subject = fmt.Sprintf("%s shared a file with you", transfer.SenderName)
		} else {
			subject = fmt.Sprintf("%s shared a file with you", transfer.SenderEmail)
		}
	}

	body, err := s.buildTransferEmailHTML(transfer, downloadURL, fileName, fileSize)
	if err != nil {
		return fmt.Errorf("failed to build email: %w", err)
	}

	return s.sendEmail(transfer.RecipientEmail, subject, body)
}

// SendDownloadNotification sends notification to sender when file is downloaded
func (s *Service) SendDownloadNotification(senderEmail, senderName, recipientEmail, fileName string) error {
	if !s.IsEnabled() {
		return fmt.Errorf("email service is not configured")
	}

	subject := fmt.Sprintf("Your file has been downloaded")
	body := s.buildDownloadNotificationHTML(senderName, recipientEmail, fileName)

	return s.sendEmail(senderEmail, subject, body)
}

// SendExpiryWarning sends an expiry warning email to the recipient
func (s *Service) SendExpiryWarning(recipientEmail, recipientName, fileName string, expiresAt interface{}) error {
	if !s.IsEnabled() {
		return fmt.Errorf("email service is not configured")
	}

	subject := fmt.Sprintf("File expiring soon: %s", fileName)

	// Format expiry time
	expiryStr := fmt.Sprintf("%v", expiresAt)

	nameOrEmail := recipientName
	if nameOrEmail == "" {
		nameOrEmail = recipientEmail
	}

	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="background: #1a1a2e; color: #eee; padding: 30px; border-radius: 12px;">
    <h2 style="color: #ff9800; margin-top: 0;">File Expiring Soon</h2>
    <p>Hi %s,</p>
    <p>The shared file <strong>%s</strong> will expire on <strong>%s</strong>.</p>
    <p>Please download it before it expires if you still need it.</p>
    <hr style="border-color: #333; margin: 20px 0;">
    <p style="color: #888; font-size: 12px;">This is an automated notification from CasaDrop.</p>
  </div>
</body>
</html>`, template.HTMLEscapeString(nameOrEmail), template.HTMLEscapeString(fileName), template.HTMLEscapeString(expiryStr))

	return s.sendEmail(recipientEmail, subject, body)
}

func (s *Service) sendEmail(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Build email headers
	fromHeader := s.config.FromEmail
	if s.config.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)
	}

	headers := make(map[string]string)
	headers["From"] = fromHeader
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	var client *smtp.Client
	var err error

	if s.config.UseTLS {
		// Direct TLS (port 465)
		tlsConfig := &tls.Config{
			ServerName: s.config.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS connection failed: %w", err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, s.config.Host)
		if err != nil {
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
	} else {
		client, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}

		if s.config.UseStartTLS {
			tlsConfig := &tls.Config{
				ServerName: s.config.Host,
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				client.Close()
				return fmt.Errorf("STARTTLS failed: %w", err)
			}
		}
	}
	defer client.Close()

	// Authenticate
	if s.config.Username != "" && s.config.Password != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Send email
	if err := client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}

	_, err = w.Write(msg.Bytes())
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("close failed: %w", err)
	}

	return client.Quit()
}

func (s *Service) buildTransferEmailHTML(transfer *models.EmailTransfer, downloadURL, fileName, fileSize string) (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; background-color: #0c1222; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;">
    <table width="100%" cellpadding="0" cellspacing="0" style="background-color: #0c1222; padding: 40px 20px;">
        <tr>
            <td align="center">
                <table width="600" cellpadding="0" cellspacing="0" style="background-color: #162032; border-radius: 16px; overflow: hidden;">
                    <!-- Logo Header -->
                    <tr>
                        <td style="padding: 30px 40px 10px; text-align: center;">
                            <table cellpadding="0" cellspacing="0" style="margin: 0 auto;">
                                <tr>
                                    <td style="vertical-align: middle; padding-right: 12px;">
                                        <svg width="40" height="40" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
                                            <path d="M24 4L4 20V44H18V30H30V44H44V20L24 4Z" fill="#0f172a" stroke="#22c55e" stroke-width="2"/>
                                            <path d="M20 20H28V26H20V20Z" fill="#22d3ee"/>
                                        </svg>
                                    </td>
                                    <td style="vertical-align: middle;">
                                        <span style="font-size: 24px; font-weight: 600; color: #22c55e;">Casa</span><span style="font-size: 24px; font-weight: 600; color: #3b82f6;">Drop</span>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    <!-- Title -->
                    <tr>
                        <td style="padding: 20px 40px 20px; text-align: center;">
                            <h1 style="margin: 0; color: #f1f5f9; font-size: 22px; font-weight: 600;">
                                {{if .SenderName}}{{.SenderName}}{{else}}{{.SenderEmail}}{{end}} shared a file with you
                            </h1>
                        </td>
                    </tr>

                    {{if .Message}}
                    <!-- Message -->
                    <tr>
                        <td style="padding: 0 40px 20px;">
                            <div style="background-color: #1a2740; border-radius: 8px; padding: 16px; color: #94a3b8; font-size: 14px; line-height: 1.6;">
                                {{.Message}}
                            </div>
                        </td>
                    </tr>
                    {{end}}

                    <!-- File Card -->
                    <tr>
                        <td style="padding: 0 40px 30px;">
                            <div style="background-color: #1a2740; border-radius: 12px; padding: 20px;">
                                <table width="100%" cellpadding="0" cellspacing="0">
                                    <tr>
                                        <td width="60">
                                            <div style="width: 50px; height: 50px; background: rgba(16, 185, 129, 0.15); border-radius: 10px; display: flex; align-items: center; justify-content: center;">
                                                <span style="font-size: 24px;">📄</span>
                                            </div>
                                        </td>
                                        <td style="padding-left: 16px;">
                                            <div style="color: #f1f5f9; font-weight: 600; font-size: 16px; margin-bottom: 4px;">{{.FileName}}</div>
                                            <div style="color: #94a3b8; font-size: 14px;">{{.FileSize}}</div>
                                        </td>
                                    </tr>
                                </table>
                            </div>
                        </td>
                    </tr>

                    <!-- Download Button -->
                    <tr>
                        <td style="padding: 0 40px 40px; text-align: center;">
                            <a href="{{.DownloadURL}}" style="display: inline-block; background: linear-gradient(135deg, #3b82f6, #2563eb); color: white; text-decoration: none; padding: 16px 48px; border-radius: 10px; font-weight: 600; font-size: 16px;">
                                Download File
                            </a>
                        </td>
                    </tr>

                    <!-- Footer -->
                    <tr>
                        <td style="padding: 20px 40px; background-color: #1a2740; text-align: center;">
                            <p style="margin: 0; color: #64748b; font-size: 12px;">
                                Sent via CasaDrop - Secure File Sharing
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	data := struct {
		SenderName  string
		SenderEmail string
		Message     string
		FileName    string
		FileSize    string
		DownloadURL string
	}{
		SenderName:  transfer.SenderName,
		SenderEmail: transfer.SenderEmail,
		Message:     strings.ReplaceAll(transfer.Message, "\n", "<br>"),
		FileName:    fileName,
		FileSize:    fileSize,
		DownloadURL: downloadURL,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *Service) buildDownloadNotificationHTML(senderName, recipientEmail, fileName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="margin: 0; padding: 0; background-color: #0c1222; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;">
    <table width="100%%" cellpadding="0" cellspacing="0" style="background-color: #0c1222; padding: 40px 20px;">
        <tr>
            <td align="center">
                <table width="600" cellpadding="0" cellspacing="0" style="background-color: #162032; border-radius: 16px; overflow: hidden;">
                    <!-- Logo Header -->
                    <tr>
                        <td style="padding: 30px 40px 10px; text-align: center;">
                            <table cellpadding="0" cellspacing="0" style="margin: 0 auto;">
                                <tr>
                                    <td style="vertical-align: middle; padding-right: 12px;">
                                        <svg width="40" height="40" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
                                            <path d="M24 4L4 20V44H18V30H30V44H44V20L24 4Z" fill="#0f172a" stroke="#22c55e" stroke-width="2"/>
                                            <path d="M20 20H28V26H20V20Z" fill="#22d3ee"/>
                                        </svg>
                                    </td>
                                    <td style="vertical-align: middle;">
                                        <span style="font-size: 24px; font-weight: 600; color: #22c55e;">Casa</span><span style="font-size: 24px; font-weight: 600; color: #3b82f6;">Drop</span>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    <tr>
                        <td style="padding: 20px 40px 40px; text-align: center;">
                            <div style="width: 80px; height: 80px; background: rgba(34, 197, 94, 0.15); border-radius: 50%%; margin: 0 auto 24px; line-height: 80px;">
                                <span style="font-size: 40px;">✓</span>
                            </div>
                            <h1 style="margin: 0 0 16px; color: #f1f5f9; font-size: 24px;">Your file was downloaded!</h1>
                            <p style="margin: 0; color: #94a3b8; font-size: 16px; line-height: 1.6;">
                                <strong style="color: #f1f5f9;">%s</strong> has downloaded<br>
                                <strong style="color: #f1f5f9;">%s</strong>
                            </p>
                        </td>
                    </tr>
                    <tr>
                        <td style="padding: 20px 40px; background-color: #1a2740; text-align: center;">
                            <p style="margin: 0; color: #64748b; font-size: 12px;">
                                Sent via CasaDrop - Secure File Sharing
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`, template.HTMLEscapeString(recipientEmail), template.HTMLEscapeString(fileName))
}
