package email

import "casadrop/internal/models"

// SMTPConfigForTest builds a minimal SMTP config with a From identity.
func SMTPConfigForTest(fromEmail, fromName string) models.SMTPConfig {
	return models.SMTPConfig{FromEmail: fromEmail, FromName: fromName}
}

type transferForTest struct {
	email string
	name  string
	msg   *models.EmailTransfer
}

func (t *transferForTest) e() *models.EmailTransfer {
	t.msg = &models.EmailTransfer{SenderEmail: t.email, SenderName: t.name}
	return t.msg
}
