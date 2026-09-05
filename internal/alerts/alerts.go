package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maximumTitleBytes   = 200
	maximumMessageBytes = 2000
	queueCapacity       = 128
	deliveryAttempts    = 3
)

var topicPattern = regexp.MustCompile(`^[-_A-Za-z0-9]{1,64}$`)

type Notification struct {
	ID       string
	Title    string
	Message  string
	Priority int
	Tags     []string
}

type Publisher interface {
	Configured() bool
	Publish(context.Context, Notification) error
}

type NtfyPublisher struct {
	baseURL *url.URL
	topic   string
	token   string
	client  *http.Client
}

type publishRequest struct {
	Topic      string   `json:"topic"`
	Message    string   `json:"message"`
	Title      string   `json:"title"`
	Priority   int      `json:"priority"`
	Tags       []string `json:"tags,omitempty"`
	SequenceID string   `json:"sequence_id,omitempty"`
}

func NewNtfyPublisher(baseURL, topic, token string) (*NtfyPublisher, error) {
	baseURL, topic, token = strings.TrimSpace(baseURL), strings.TrimSpace(topic), strings.TrimSpace(token)
	if baseURL == "" && topic == "" && token == "" {
		return &NtfyPublisher{}, nil
	}
	if baseURL == "" || topic == "" {
		return nil, errors.New("NTFY_BASE_URL and NTFY_TOPIC must be set together")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("NTFY_BASE_URL must be an absolute HTTPS origin without credentials, query, or fragment")
	}
	if !topicPattern.MatchString(topic) {
		return nil, errors.New("NTFY_TOPIC must be 1-64 letters, numbers, underscores, or dashes")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/"
	return &NtfyPublisher{
		baseURL: parsed, topic: topic, token: token,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(request *http.Request, via []*http.Request) error {
				if len(via) >= 3 || request.URL.Scheme != parsed.Scheme || !strings.EqualFold(request.URL.Host, parsed.Host) {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}, nil
}

func (publisher *NtfyPublisher) Configured() bool {
	return publisher != nil && publisher.baseURL != nil && publisher.client != nil && publisher.topic != ""
}

func (publisher *NtfyPublisher) Publish(ctx context.Context, notification Notification) error {
	if !publisher.Configured() {
		return nil
	}
	payload := publishRequest{
		Topic: publisher.topic, Title: truncate(notification.Title, maximumTitleBytes),
		Message: truncate(notification.Message, maximumMessageBytes), Priority: min(max(notification.Priority, 1), 5),
		Tags: safeTags(notification.Tags), SequenceID: truncate(notification.ID, 80),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return errors.New("encode ntfy notification")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, publisher.baseURL.String(), bytes.NewReader(body))
	if err != nil {
		return errors.New("create ntfy request")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if publisher.token != "" {
		request.Header.Set("Authorization", "Bearer "+publisher.token)
	}
	response, err := publisher.client.Do(request)
	if err != nil {
		return errors.New("publish ntfy notification")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned HTTP %d", response.StatusCode)
	}
	return nil
}

type Dispatcher struct {
	publisher  Publisher
	queue      chan Notification
	logger     *slog.Logger
	retryDelay time.Duration
}

func NewDispatcher(publisher Publisher, logger *slog.Logger) *Dispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Dispatcher{
		publisher: publisher, queue: make(chan Notification, queueCapacity),
		logger: logger, retryDelay: time.Second,
	}
}

func (dispatcher *Dispatcher) Configured() bool {
	return dispatcher != nil && dispatcher.publisher != nil && dispatcher.publisher.Configured()
}

func (dispatcher *Dispatcher) Notify(ctx context.Context, notification Notification) error {
	if !dispatcher.Configured() {
		return nil
	}
	select {
	case dispatcher.queue <- notification:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (dispatcher *Dispatcher) Run(ctx context.Context) {
	if !dispatcher.Configured() {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case notification := <-dispatcher.queue:
			dispatcher.deliver(ctx, notification)
		}
	}
}

func (dispatcher *Dispatcher) deliver(ctx context.Context, notification Notification) {
	for attempt := 1; attempt <= deliveryAttempts; attempt++ {
		if err := dispatcher.publisher.Publish(ctx, notification); err == nil {
			return
		} else if ctx.Err() != nil {
			return
		}
		if attempt < deliveryAttempts && !wait(ctx, dispatcher.retryDelay*time.Duration(attempt)) {
			return
		}
	}
	dispatcher.logger.Error("Alert notification delivery failed", "notification_id", notification.ID)
}

func safeTags(tags []string) []string {
	result := make([]string, 0, min(len(tags), 5))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" && len(tag) <= 40 {
			result = append(result, tag)
		}
		if len(result) == 5 {
			break
		}
	}
	return result
}

func truncate(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maximum {
		return value
	}
	end := maximum
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end]
}

func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
