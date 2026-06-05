package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"casadrop/internal/models"
	"casadrop/internal/utils"
)

// maxConcurrentWebhooks bounds in-flight webhook deliveries so a slow or
// hostile receiver can't exhaust goroutines/sockets under a burst of events.
const maxConcurrentWebhooks = 32

// Service handles webhook notifications
type Service struct {
	config     models.WebhookConfig
	configPath string
	client     *http.Client
	mu         sync.RWMutex
	sem        chan struct{}
	wg         sync.WaitGroup
	stopped    atomic.Bool
}

// strictSSRFTransport resolves each target host and refuses to dial any
// private/loopback/link-local IP, then pins the connection to a validated IP so
// DNS rebinding between validation and dial can't redirect the request to an
// internal service. Enabled via WEBHOOK_STRICT_SSRF=true (off by default to keep
// LAN webhook receivers working in homelab setups).
func strictSSRFTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			var lastErr error = fmt.Errorf("webhook: no dialable address for %s", host)
			for _, ipa := range ips {
				if utils.IsBlockedIP(ipa.IP) {
					return nil, fmt.Errorf("webhook: refusing to connect to blocked address %s", ipa.IP)
				}
			}
			for _, ipa := range ips {
				conn, derr := dialer.DialContext(ctx, network, net.JoinHostPort(ipa.IP.String(), port))
				if derr == nil {
					return conn, nil
				}
				lastErr = derr
			}
			return nil, lastErr
		},
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	}
}

// New creates a new webhook service
func New(dataDir string) *Service {
	s := &Service{
		configPath: filepath.Join(dataDir, "webhook_config.json"),
		client: &http.Client{
			Timeout: 10 * time.Second,
			// Refuse redirects: a validated public URL could otherwise 302 to an
			// internal address (169.254.169.254, 127.0.0.1, …), bypassing the
			// SSRF guard that only inspects the originally configured URL.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		sem: make(chan struct{}, maxConcurrentWebhooks),
	}
	if os.Getenv("WEBHOOK_STRICT_SSRF") == "true" {
		s.client.Transport = strictSSRFTransport()
	}
	s.loadConfig()

	// Override from environment variables
	if url := os.Getenv("WEBHOOK_URL"); url != "" {
		// Apply the same SSRF guard to operator-supplied env URLs.
		if err := utils.ValidateExternalWebhookURL(url); err != nil {
			log.Printf("Webhook: ignoring WEBHOOK_URL from env: %v", err)
		} else {
			s.config.Enabled = true
			s.config.URL = url
			s.config.OnDownload = true
		}
	}
	if secret := os.Getenv("WEBHOOK_SECRET"); secret != "" {
		s.config.Secret = secret
	}

	return s
}

// loadConfig loads webhook configuration from file
func (s *Service) loadConfig() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		// Default config
		s.config = models.WebhookConfig{
			Enabled:    false,
			OnDownload: true,
		}
		return
	}

	if err := json.Unmarshal(data, &s.config); err != nil {
		log.Printf("Failed to parse webhook config: %v", err)
	}
}

// SaveConfig saves webhook configuration to file
func (s *Service) SaveConfig(config models.WebhookConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		return err
	}

	s.config = config
	return nil
}

// GetConfig returns the current webhook configuration
func (s *Service) GetConfig() models.WebhookConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// NotifyDownload sends a webhook notification for a download event
func (s *Service) NotifyDownload(share *models.Share, clientIP, userAgent string) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	if !config.Enabled || !config.OnDownload || config.URL == "" {
		return
	}

	payload := models.WebhookPayload{
		Event:     "download",
		ShareID:   share.ID,
		FileName:  share.OriginalName,
		Downloads: share.Downloads,
		Timestamp: time.Now(),
		ClientIP:  clientIP,
		UserAgent: userAgent,
	}

	s.dispatch(payload)
}

// NotifyLimitReached sends a webhook notification when download limit is reached
func (s *Service) NotifyLimitReached(share *models.Share) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	if !config.Enabled || !config.OnLimitReached || config.URL == "" {
		return
	}

	payload := models.WebhookPayload{
		Event:     "limit_reached",
		ShareID:   share.ID,
		FileName:  share.OriginalName,
		Downloads: share.Downloads,
		Timestamp: time.Now(),
	}

	s.dispatch(payload)
}

// NotifyExpire sends a webhook notification when a share expires
func (s *Service) NotifyExpire(share *models.Share) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	if !config.Enabled || !config.OnExpire || config.URL == "" {
		return
	}

	payload := models.WebhookPayload{
		Event:     "expire",
		ShareID:   share.ID,
		FileName:  share.OriginalName,
		Downloads: share.Downloads,
		Timestamp: time.Now(),
	}

	s.dispatch(payload)
}

// dispatch sends a webhook in the background while capping concurrency.
// If the in-flight budget is exhausted the event is dropped (and logged)
// rather than piling up unbounded goroutines.
func (s *Service) dispatch(payload models.WebhookPayload) {
	if s.stopped.Load() {
		return
	}
	select {
	case s.sem <- struct{}{}:
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() { <-s.sem }()
			s.sendWebhook(payload)
		}()
	default:
		log.Printf("Webhook: delivery budget exhausted, dropping %s notification for share %s", payload.Event, payload.ShareID)
	}
}

// Stop refuses new webhook deliveries and waits (up to 5s) for in-flight ones
// to finish, so notifications aren't abandoned mid-flight on graceful shutdown.
func (s *Service) Stop() {
	s.stopped.Store(true)
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Printf("Webhook: timed out draining in-flight deliveries on shutdown")
	}
}

// sendWebhook sends the webhook HTTP request
func (s *Service) sendWebhook(payload models.WebhookPayload) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Webhook: failed to marshal payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", config.URL, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Webhook: failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Zima-Share/1.1")
	req.Header.Set("X-Webhook-Event", payload.Event)

	// Add HMAC signature if secret is configured
	if config.Secret != "" {
		signature := s.computeSignature(data, config.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Webhook: failed to send request: %v", err)
		return
	}
	defer resp.Body.Close()
	// Drain body to enable HTTP connection reuse
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		log.Printf("Webhook: received error status %d", resp.StatusCode)
	} else {
		log.Printf("Webhook: successfully sent %s notification for share %s", payload.Event, payload.ShareID)
	}
}

// computeSignature computes HMAC-SHA256 signature
func (s *Service) computeSignature(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
