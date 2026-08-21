package ntfy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/oschrenk/arto/internal/config"
)

type NotificationParams struct {
	Server    string
	Topic     string
	Title     string
	Message   string
	Actions   []Action
	RequestID string
	Auth      *config.Auth
}

type Action struct {
	ActionType string            `json:"action"`
	Label      string            `json:"label"`
	URL        string            `json:"url"`
	Body       string            `json:"body"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type notificationBody struct {
	Topic   string   `json:"topic"`
	Title   string   `json:"title"`
	Message string   `json:"message"`
	Actions []Action `json:"actions"`
}

type Response struct {
	Approved   bool
	AlwaysAllow bool
	Answer     string
	Timeout    bool
	Err        error
}

func BuildAuthHeader(auth *config.Auth) map[string]string {
	if auth == nil {
		return nil
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(auth.Username + ":" + auth.Password))
	return map[string]string{"Authorization": "Basic " + encoded}
}

func SendNotification(params NotificationParams) error {
	server := strings.TrimRight(params.Server, "/")

	body := notificationBody{
		Topic:   params.Topic,
		Title:   params.Title,
		Message: params.Message,
		Actions: params.Actions,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling notification: %w", err)
	}

	req, err := http.NewRequest("POST", server, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if headers := BuildAuthHeader(params.Auth); headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy notification failed: HTTP %d", resp.StatusCode)
	}

	return nil
}

func WaitForResponse(ctx context.Context, params NotificationParams, requestID string, timeout time.Duration) Response {
	server := strings.TrimRight(params.Server, "/")
	url := fmt.Sprintf("%s/%s-response/json", server, params.Topic)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Response{Err: fmt.Errorf("creating request: %w", err)}
	}

	if headers := BuildAuthHeader(params.Auth); headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Response{Err: fmt.Errorf("connecting to response stream: %w", err)}
	}
	defer resp.Body.Close()

	// Start timeout after connection is established
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		resp Response
	}
	ch := make(chan result, 1)

	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var event struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			var parsed struct {
				RequestID   string `json:"requestId"`
				Approved    bool   `json:"approved"`
				AlwaysAllow bool   `json:"alwaysAllow"`
				Answer      string `json:"answer"`
			}
			if err := json.Unmarshal([]byte(event.Message), &parsed); err != nil {
				continue
			}

			if parsed.RequestID == requestID {
				if parsed.Answer != "" {
					ch <- result{Response{Answer: parsed.Answer}}
				} else {
					ch <- result{Response{Approved: parsed.Approved, AlwaysAllow: parsed.AlwaysAllow}}
				}
				return
			}
		}
		// Stream ended without match
		ch <- result{Response{Timeout: true}}
	}()

	select {
	case r := <-ch:
		return r.resp
	case <-ctx.Done():
		resp.Body.Close() // Force close to unblock scanner
		return Response{Timeout: true}
	}
}

func SendWithRetry(params NotificationParams) error {
	const maxRetries = 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if err := SendNotification(params); err != nil {
			lastErr = err
			if i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * time.Second)
			}
			continue
		}
		return nil
	}

	fmt.Fprintf(os.Stderr, "[arto] Notification failed after %d attempts: %v — Falling back to CLI.\n", maxRetries, lastErr)
	return lastErr
}
