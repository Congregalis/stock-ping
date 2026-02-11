package notify

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Notifier sends push notifications via Bark
type Notifier struct {
	serverURL  string
	key        string
	httpClient *http.Client
}

// NewNotifier creates a new Bark notifier
func NewNotifier(serverURL, key string) *Notifier {
	if serverURL == "" {
		serverURL = "https://api.day.app"
	}
	return &Notifier{
		serverURL: serverURL,
		key:       key,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send sends a push notification with title and body
func (n *Notifier) Send(title, body string) error {
	if n.key == "" {
		return fmt.Errorf("bark key is not configured")
	}

	// URL encode the title and body
	encodedTitle := url.PathEscape(title)
	encodedBody := url.PathEscape(body)

	// Build the URL: https://api.day.app/{key}/{title}/{body}
	notifyURL := fmt.Sprintf("%s/%s/%s/%s", n.serverURL, n.key, encodedTitle, encodedBody)

	resp, err := n.httpClient.Get(notifyURL)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bark returned status %d", resp.StatusCode)
	}

	return nil
}

// SendWithGroup sends a notification with a group name
func (n *Notifier) SendWithGroup(title, body, group string) error {
	if n.key == "" {
		return fmt.Errorf("bark key is not configured")
	}

	encodedTitle := url.PathEscape(title)
	encodedBody := url.PathEscape(body)

	notifyURL := fmt.Sprintf("%s/%s/%s/%s?group=%s",
		n.serverURL, n.key, encodedTitle, encodedBody, url.QueryEscape(group))

	resp, err := n.httpClient.Get(notifyURL)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bark returned status %d", resp.StatusCode)
	}

	return nil
}

// IsConfigured returns true if the notifier is properly configured
func (n *Notifier) IsConfigured() bool {
	return n.key != ""
}
