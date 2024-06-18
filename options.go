package sdbc

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const (
	defaultReadLimit = 1 << (10 * 2) // 1 MB
)

type options struct {
	timeout       time.Duration
	logger        *slog.Logger
	jsonMarshal   JsonMarshal
	jsonUnmarshal JsonUnmarshal
	readLimit     int64
	httpClient    HttpClient
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

// WithJsonHandlers sets custom json marshal and unmarshal functions.
// If not set, the default json marshal and unmarshal functions are used.
func WithJsonHandlers(marshal JsonMarshal, unmarshal JsonUnmarshal) Option {
	return func(c *options) {
		c.jsonMarshal = marshal
		c.jsonUnmarshal = unmarshal
	}
}

// WithReadLimit sets a custom read limit (in bytes) for the websocket connection.
// If not set, the default read limit is 1 MB.
func WithReadLimit(limit int64) Option {
	return func(c *options) {
		c.readLimit = limit
	}
}

// WithHttpClient sets a custom http client.
// If not set, the default http client is used.
func WithHttpClient(client HttpClient) Option {
	return func(c *options) {
		c.httpClient = client
	}
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type (
	JsonMarshal   func(val any) ([]byte, error)
	JsonUnmarshal func(buf []byte, val any) error
)

func applyOptions(opts []Option) *options {
	out := &options{
		timeout:       time.Minute,
		logger:        slog.New(&emptyLogHandler{}),
		jsonMarshal:   json.Marshal,
		jsonUnmarshal: json.Unmarshal,
		readLimit:     defaultReadLimit,
		httpClient:    http.DefaultClient,
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
