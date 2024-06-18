package sdbc

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
)

func prepare(tb testing.TB) {
	tb.Helper()

	slog.SetDefault(slog.New(newLogger(tb, nil)))
}

func newLogger(tb testing.TB, opts *slog.HandlerOptions) *testLogger {
	tb.Helper()

	buf := &bytes.Buffer{}

	handler := slog.NewTextHandler(buf, opts)

	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	return &testLogger{
		tb:      tb,
		opts:    opts,
		handler: handler,
		buf:     buf,
		mu:      &sync.Mutex{},
	}
}

var _ slog.Handler = (*testLogger)(nil)

type testLogger struct {
	tb      testing.TB
	opts    *slog.HandlerOptions
	handler slog.Handler
	buf     *bytes.Buffer
	records []slog.Record
	mu      *sync.Mutex
}

func (l *testLogger) Enabled(_ context.Context, level slog.Level) bool {
	if l.opts == nil || l.opts.Level == nil {
		return true
	}

	return level >= l.opts.Level.Level()
}

func (l *testLogger) Handle(ctx context.Context, record slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.records = append(l.records, record)

	if err := l.handler.Handle(ctx, record); err != nil {
		return err
	}

	output, err := io.ReadAll(l.buf)
	if err != nil {
		return err
	}

	// The output comes back with a newline, which we need to
	// trim before feeding to t.Log.
	output = bytes.TrimSuffix(output, []byte("\n"))

	// Add calldepth. But it won't be enough, and the internal slog
	// callsite will be printed. See discussion in README.md.
	l.tb.Helper()

	l.tb.Log(string(output))

	return nil
}

func (l *testLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &testLogger{
		tb:      l.tb,
		opts:    l.opts,
		handler: l.handler.WithAttrs(attrs),
		buf:     l.buf,
		mu:      l.mu,
	}
}

func (l *testLogger) WithGroup(group string) slog.Handler {
	return &testLogger{
		tb:      l.tb,
		opts:    l.opts,
		handler: l.handler.WithGroup(group),
		buf:     l.buf,
		mu:      l.mu,
	}
}

func (l *testLogger) hasRecordMsg(msg string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, r := range l.records {
		if r.Message == msg {
			return true
		}
	}

	return false
}
