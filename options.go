package sdbc

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const (
	defaultTimeout   = 1 * time.Minute
	defaultReadLimit = 1 << (10 * 2) // 1 MB
)

type options struct {
	timeout    time.Duration
	logger     *slog.Logger
	readLimit  int64
	httpClient HTTPClient
}

type Option func(*options)

// WithTimeout sets a custom timeout for requests.
// If not set, the default timeout is 1 minute.
func WithTimeout(timeout time.Duration) Option {
	return func(c *options) {
		c.timeout = timeout
	}
}

// WithLogger sets the logger.
// If not set, no log output is created.
func WithLogger(logger *slog.Logger) Option {
	return func(c *options) {
		c.logger = logger
	}
}

// WithReadLimit sets a custom read limit (in bytes) for the websocket connection.
// If not set, the default read limit is 1 MB.
func WithReadLimit(limit int64) Option {
	return func(c *options) {
		c.readLimit = limit
	}
}

// WithHTTPClient sets a custom http client.
// If not set, the default http client is used.
func WithHTTPClient(client HTTPClient) Option {
	return func(c *options) {
		c.httpClient = client
	}
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type (
	Marshal   func(val any) ([]byte, error)
	Unmarshal func(buf []byte, val any) error
)

func applyOptions(opts []Option) *options {
	out := &options{
		timeout:    defaultTimeout,
		logger:     slog.New(&emptyLogHandler{}),
		readLimit:  defaultReadLimit,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(out)
	}

	return out
}

type emptyLogHandler struct{}

func (h emptyLogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

func (h emptyLogHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (h emptyLogHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h emptyLogHandler) WithGroup(_ string) slog.Handler {
	return h
}
