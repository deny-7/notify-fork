package webhook

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Webhook struct holds necessary data to communicate with the Webhook API.
type Webhook struct {
	webhook string
	timeout int
}

// New returns a new instance of a webhook service.
func New(webhook string, timeout int) *Webhook {
	w := &Webhook{
		webhook: webhook,
		timeout: timeout,
	}

	return w
}

// Send takes a JSON message and sends it to specified webook URL, subject is ignored.
func (w Webhook) Send(ctx context.Context, _ string, message string) error {
	jsonReader := strings.NewReader(message)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		w.webhook,
		jsonReader,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Duration(w.timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %s", resp.Status)
	}

	return nil
}
