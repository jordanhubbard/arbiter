package analytics

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

// SMTPConfig defines SMTP server configuration for email notifications
type SMTPConfig struct {
	Host     string // SMTP server hostname (e.g., smtp.gmail.com)
	Port     int    // SMTP server port (e.g., 587 for TLS)
	Username string // SMTP username
	Password string // SMTP password
	From     string // From email address
	UseTLS   bool   // Whether to use TLS (default: true)
}

// AlertConfig defines alerting thresholds and settings
type AlertConfig struct {
	UserID              string  `json:"user_id"`
	DailyBudgetUSD      float64 `json:"daily_budget_usd"`   // Alert if daily spend exceeds
	MonthlyBudgetUSD    float64 `json:"monthly_budget_usd"` // Alert if monthly spend exceeds
	AnomalyThreshold    float64 `json:"anomaly_threshold"`  // Alert if spend is X times normal (e.g., 2.0 = 2x)
	EnableEmailAlerts   bool    `json:"enable_email_alerts"`
	EnableWebhookAlerts bool    `json:"enable_webhook_alerts"`
	WebhookURL          string  `json:"webhook_url"`
	EmailAddress        string  `json:"email_address"`
}

// Alert represents a triggered alert
type Alert struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Type         string    `json:"type"`     // "budget_exceeded", "anomaly_detected"
	Severity     string    `json:"severity"` // "info", "warning", "critical"
	Message      string    `json:"message"`
	CurrentCost  float64   `json:"current_cost"`
	Threshold    float64   `json:"threshold"`
	TriggeredAt  time.Time `json:"triggered_at"`
	Acknowledged bool      `json:"acknowledged"`
}

// AlertChecker monitors spending and triggers alerts
type AlertChecker struct {
	storage    Storage
	config     *AlertConfig
	smtpConfig *SMTPConfig
}

// NewAlertChecker creates a new alert checker
func NewAlertChecker(storage Storage, config *AlertConfig) *AlertChecker {
	return &AlertChecker{
		storage:    storage,
		config:     config,
		smtpConfig: loadSMTPConfigFromEnv(),
	}
}

// loadSMTPConfigFromEnv loads SMTP configuration from environment variables
func loadSMTPConfigFromEnv() *SMTPConfig {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return nil // SMTP not configured
	}

	portStr := os.Getenv("SMTP_PORT")
	port := 587 // Default TLS port
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	useTLS := true
	if tlsStr := os.Getenv("SMTP_USE_TLS"); tlsStr == "false" || tlsStr == "0" {
		useTLS = false
	}

	return &SMTPConfig{
		Host:     host,
		Port:     port,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
		UseTLS:   useTLS,
	}
}

// CheckAlerts checks for spending anomalies and budget overruns
func (ac *AlertChecker) CheckAlerts(ctx context.Context) ([]*Alert, error) {
	alerts := make([]*Alert, 0)

	// Check daily budget
	if ac.config.DailyBudgetUSD > 0 {
		if alert := ac.checkDailyBudget(ctx); alert != nil {
			alerts = append(alerts, alert)
		}
	}

	// Check monthly budget
	if ac.config.MonthlyBudgetUSD > 0 {
		if alert := ac.checkMonthlyBudget(ctx); alert != nil {
			alerts = append(alerts, alert)
		}
	}

	// Check for anomalies
	if ac.config.AnomalyThreshold > 1.0 {
		if alert := ac.checkAnomalies(ctx); alert != nil {
			alerts = append(alerts, alert)
		}
	}

	// Notify for each alert
	for _, alert := range alerts {
		ac.notify(alert)
	}

	return alerts, nil
}

// checkDailyBudget checks if daily spending exceeds budget
func (ac *AlertChecker) checkDailyBudget(ctx context.Context) *Alert {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	stats, err := ac.storage.GetLogStats(ctx, &LogFilter{
		UserID:    ac.config.UserID,
		StartTime: startOfDay,
		EndTime:   now,
	})
	if err != nil {
		return nil
	}

	if stats.TotalCostUSD > ac.config.DailyBudgetUSD {
		return &Alert{
			ID:          fmt.Sprintf("alert-daily-%d", time.Now().Unix()),
			UserID:      ac.config.UserID,
			Type:        "budget_exceeded",
			Severity:    "warning",
			Message:     fmt.Sprintf("Daily budget exceeded: $%.2f / $%.2f (%.0f%%)", stats.TotalCostUSD, ac.config.DailyBudgetUSD, (stats.TotalCostUSD/ac.config.DailyBudgetUSD)*100),
			CurrentCost: stats.TotalCostUSD,
			Threshold:   ac.config.DailyBudgetUSD,
			TriggeredAt: now,
		}
	}

	return nil
}

// checkMonthlyBudget checks if monthly spending exceeds budget
func (ac *AlertChecker) checkMonthlyBudget(ctx context.Context) *Alert {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	stats, err := ac.storage.GetLogStats(ctx, &LogFilter{
		UserID:    ac.config.UserID,
		StartTime: startOfMonth,
		EndTime:   now,
	})
	if err != nil {
		return nil
	}

	if stats.TotalCostUSD > ac.config.MonthlyBudgetUSD {
		return &Alert{
			ID:          fmt.Sprintf("alert-monthly-%d", time.Now().Unix()),
			UserID:      ac.config.UserID,
			Type:        "budget_exceeded",
			Severity:    "critical",
			Message:     fmt.Sprintf("Monthly budget exceeded: $%.2f / $%.2f (%.0f%%)", stats.TotalCostUSD, ac.config.MonthlyBudgetUSD, (stats.TotalCostUSD/ac.config.MonthlyBudgetUSD)*100),
			CurrentCost: stats.TotalCostUSD,
			Threshold:   ac.config.MonthlyBudgetUSD,
			TriggeredAt: now,
		}
	}

	return nil
}

// checkAnomalies detects unusual spending patterns
func (ac *AlertChecker) checkAnomalies(ctx context.Context) *Alert {
	now := time.Now()

	// Get today's spending
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayStats, err := ac.storage.GetLogStats(ctx, &LogFilter{
		UserID:    ac.config.UserID,
		StartTime: startOfToday,
		EndTime:   now,
	})
	if err != nil {
		return nil
	}

	// Get average spending from last 7 days (excluding today)
	sevenDaysAgo := startOfToday.Add(-7 * 24 * time.Hour)
	historicalStats, err := ac.storage.GetLogStats(ctx, &LogFilter{
		UserID:    ac.config.UserID,
		StartTime: sevenDaysAgo,
		EndTime:   startOfToday,
	})
	if err != nil {
		return nil
	}

	// Calculate average daily spending
	avgDailySpend := historicalStats.TotalCostUSD / 7.0

	// Check if today's spending is anomalous
	if avgDailySpend > 0 && todayStats.TotalCostUSD > (avgDailySpend*ac.config.AnomalyThreshold) {
		return &Alert{
			ID:          fmt.Sprintf("alert-anomaly-%d", time.Now().Unix()),
			UserID:      ac.config.UserID,
			Type:        "anomaly_detected",
			Severity:    "warning",
			Message:     fmt.Sprintf("Unusual spending detected: $%.2f today vs $%.2f average (%.0fx increase)", todayStats.TotalCostUSD, avgDailySpend, todayStats.TotalCostUSD/avgDailySpend),
			CurrentCost: todayStats.TotalCostUSD,
			Threshold:   avgDailySpend * ac.config.AnomalyThreshold,
			TriggeredAt: now,
		}
	}

	return nil
}

// notify sends notifications for an alert
func (ac *AlertChecker) notify(alert *Alert) {
	// Log the alert
	log.Printf("[ALERT] %s: %s", alert.Severity, alert.Message)

	// Send email notifications if enabled
	if ac.config.EnableEmailAlerts && ac.config.EmailAddress != "" {
		if ac.smtpConfig == nil {
			log.Printf("[ALERT] Email notifications enabled but SMTP not configured (set SMTP_HOST env var)")
		} else {
			if err := ac.sendEmail(alert); err != nil {
				log.Printf("[ALERT] Failed to send email to %s: %v", ac.config.EmailAddress, err)
			} else {
				log.Printf("[ALERT] Email notification sent to %s: %s", ac.config.EmailAddress, alert.Message)
			}
		}
	}

	// Send webhook notifications if enabled
	if ac.config.EnableWebhookAlerts && ac.config.WebhookURL != "" {
		if err := ac.sendWebhook(alert); err != nil {
			log.Printf("[ALERT] Failed to send webhook to %s: %v", ac.config.WebhookURL, err)
		} else {
			log.Printf("[ALERT] Webhook notification sent to %s: %s", ac.config.WebhookURL, alert.Message)
		}
	}
}

// sendWebhook sends an alert via HTTP webhook
func (ac *AlertChecker) sendWebhook(alert *Alert) error {
	// Prepare webhook payload
	payload := map[string]interface{}{
		"id":           alert.ID,
		"user_id":      alert.UserID,
		"type":         alert.Type,
		"severity":     alert.Severity,
		"message":      alert.Message,
		"current_cost": alert.CurrentCost,
		"threshold":    alert.Threshold,
		"triggered_at": alert.TriggeredAt.Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", ac.config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Loom-Alerts/1.0")
	req.Header.Set("X-Alert-Type", alert.Type)
	req.Header.Set("X-Alert-Severity", alert.Severity)

	// Send request with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
	}

	return nil
}

// sendEmail sends an alert via email using SMTP
func (ac *AlertChecker) sendEmail(alert *Alert) error {
	if ac.smtpConfig == nil {
		return fmt.Errorf("SMTP not configured")
	}

	// Determine sender email
	from := ac.smtpConfig.From
	if from == "" {
		from = ac.smtpConfig.Username // Fallback to username if From not set
	}

	// Build email message
	subject := fmt.Sprintf("[Loom Alert] %s: %s", alert.Severity, alert.Type)
	body := buildEmailBody(alert)

	// Construct email headers and body
	message := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		from,
		ac.config.EmailAddress,
		subject,
		body,
	))

	// Set up authentication
	auth := smtp.PlainAuth("", ac.smtpConfig.Username, ac.smtpConfig.Password, ac.smtpConfig.Host)

	// Send email
	addr := fmt.Sprintf("%s:%d", ac.smtpConfig.Host, ac.smtpConfig.Port)

	if ac.smtpConfig.UseTLS {
		// Use TLS (recommended for most SMTP servers)
		return sendEmailTLS(addr, auth, from, []string{ac.config.EmailAddress}, message, ac.smtpConfig.Host)
	}

	// Send without TLS (not recommended for production)
	return smtp.SendMail(addr, auth, from, []string{ac.config.EmailAddress}, message)
}

// sendEmailTLS sends email using explicit TLS
func sendEmailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte, host string) error {
	// Create TLS connection
	tlsConfig := &tls.Config{
		ServerName: host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to create TLS connection: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer func() { _ = client.Quit() }()

	// Authenticate
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender and recipients
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	// Send email data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err = writer.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

// buildEmailBody creates an HTML email body for the alert
func buildEmailBody(alert *Alert) string {
	severityColor := "#FFA500" // Orange for warning
	if alert.Severity == "critical" {
		severityColor = "#DC3545" // Red
	} else if alert.Severity == "info" {
		severityColor = "#17A2B8" // Blue
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: %s; color: white; padding: 20px; border-radius: 5px 5px 0 0; }
        .content { background-color: #f9f9f9; padding: 20px; border: 1px solid #ddd; border-radius: 0 0 5px 5px; }
        .detail { margin: 10px 0; }
        .label { font-weight: bold; color: #555; }
        .value { color: #333; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #777; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1 style="margin: 0;">Loom Alert</h1>
            <p style="margin: 5px 0 0 0;">%s Alert</p>
        </div>
        <div class="content">
            <div class="detail">
                <span class="label">Type:</span>
                <span class="value">%s</span>
            </div>
            <div class="detail">
                <span class="label">Message:</span>
                <span class="value">%s</span>
            </div>
            <div class="detail">
                <span class="label">Current Cost:</span>
                <span class="value">$%.2f USD</span>
            </div>
            <div class="detail">
                <span class="label">Threshold:</span>
                <span class="value">$%.2f USD</span>
            </div>
            <div class="detail">
                <span class="label">Triggered At:</span>
                <span class="value">%s</span>
            </div>
            <div class="detail">
                <span class="label">Alert ID:</span>
                <span class="value">%s</span>
            </div>
        </div>
        <div class="footer">
            <p>This is an automated alert from Loom. Please do not reply to this email.</p>
            <p>To manage your alert settings, please log in to your Loom dashboard.</p>
        </div>
    </div>
</body>
</html>
`,
		severityColor,
		alert.Severity,
		alert.Type,
		alert.Message,
		alert.CurrentCost,
		alert.Threshold,
		alert.TriggeredAt.Format("2006-01-02 15:04:05 MST"),
		alert.ID,
	)
}

// DefaultAlertConfig provides sensible defaults
func DefaultAlertConfig(userID string) *AlertConfig {
	return &AlertConfig{
		UserID:              userID,
		DailyBudgetUSD:      100.0,  // $100/day default
		MonthlyBudgetUSD:    2000.0, // $2000/month default
		AnomalyThreshold:    2.0,    // Alert if 2x normal spending
		EnableEmailAlerts:   false,
		EnableWebhookAlerts: false,
	}
}
