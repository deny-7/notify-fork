import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	webhook := "https://example.com/webhook"
	timeout := 10

	w := New(webhook, timeout)

	require.NotNil(t, w)
	require.Equal(t, webhook, w.webhook)
	require.Equal(t, timeout, w.timeout)
}

func TestWebhook_Send(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		webhook       string
		timeout       int
		message       string
		subject       string
		serverHandler http.HandlerFunc
		expectedError string
	}{
		{
			name:    "Successful webhook send",
			webhook: "https://example.com/webhook",
			timeout: 5,
			subject: "Test Subject",
			message: `{"text":"Test Message"}`,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.Equal(t, `{"text":"Test Message"}`, string(body))
				w.WriteHeader(http.StatusOK)
			},
			expectedError: "",
		},
		{
			name:    "Webhook returns 201 Created",
			webhook: "https://example.com/webhook",
			timeout: 5,
			subject: "Test",
			message: `{"data":"test"}`,
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
			},
			expectedError: "",
		},
		{
			name:    "Webhook returns 299 status",
			webhook: "https://example.com/webhook",
			timeout: 5,
			subject: "Test",
			message: `{"data":"test"}`,
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusIMUsed)
			},
			expectedError: "",
		},
		{
			name:    "Webhook returns 300 status",
			webhook: "https://example.com/webhook",
			timeout: 5,
			subject: "Test",
			message: `{"data":"test"}`,
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusMultipleChoices)
			},
			expectedError: "webhook returned status 300 Multiple Choices",
		},
		{
			name:    "Webhook returns 400 Bad Request",
			webhook: "https://example.com/webhook",
			timeout: 5,
			subject: "Test",
			message: `{"data":"test"}`,
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			expectedError: "webhook returned status 400 Bad Request",
		},
		{
			name:    "Webhook returns 500 Internal Server Error",
			webhook: "https://example.com/webhook",
			timeout: 5,
			subject: "Test",
			message: `{"data":"test"}`,
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError: "webhook returned status 500 Internal Server Error",
		},
		{
			name:    "Subject is ignored",
			webhook: "https://example.com/webhook",
			timeout: 5,
			subject: "This Subject Should Be Ignored",
			message: `{"data":"test"}`,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				// Subject should not be in the request body
				require.False(t, strings.Contains(string(body), "This Subject Should Be Ignored"))
				w.WriteHeader(http.StatusOK)
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create test server
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			// Create webhook with server URL
			w := New(server.URL, tt.timeout)

			err := w.Send(context.Background(), tt.subject, tt.message)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWebhook_SendWithContextCancellation(t *testing.T) {
	t.Parallel()

	// Create a server that blocks indefinitely
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	w := New(server.URL, 5)

	// Create a context that is already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := w.Send(ctx, "Test", `{"data":"test"}`)

	require.Error(t, err)
	// Context cancellation results in a URL error wrapping context.Canceled
	require.True(t, strings.Contains(err.Error(), "context canceled") || strings.Contains(err.Error(), "Post"))
}

func TestWebhook_SendWithTimeout(t *testing.T) {
	t.Parallel()

	// Create a server that delays longer than the webhook timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * time.Second)
	}))
	defer server.Close()

	w := New(server.URL, 1) // 1 second timeout

	err := w.Send(context.Background(), "Test", `{"data":"test"}`)

	require.Error(t, err)
	// Check that it's a timeout error
	require.True(t, strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "timeout"), fmt.Sprintf("Unexpected error: %v", err))
}

func TestWebhook_SendWithInvalidURL(t *testing.T) {
	t.Parallel()

	w := New("invalid://[::1]invalid", 5)

	err := w.Send(context.Background(), "", `{"data":"test"}`)

	require.Error(t, err)
}

func TestWebhook_SendWithUnreachableHost(t *testing.T) {
	t.Parallel()

	w := New("http://192.0.2.1:8888", 1)

	err := w.Send(context.Background(), "", `{"data":"test"}`)

	require.Error(t, err)
}
