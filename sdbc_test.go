package sdbc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// errAlreadyInProgress is a regular expression that matches the error for a container
// removal that is already in progress.
var errAlreadyInProgress = regexp.MustCompile(`removal of container .* is already in progress`)

func prepare(tb testing.TB) {
	tb.Helper()

	slog.SetDefault(slog.New(newLogger(tb, nil)))
}

func prepareSurreal(ctx context.Context, tb testing.TB, opts ...Option) (*Client, func()) {
	tb.Helper()

	username := gofakeit.Username()
	password := gofakeit.Password(true, true, true, true, true, 32)
	namespace := gofakeit.FirstName()
	database := gofakeit.LastName()

	tb.Logf("Creating database with: username=%s, password=%s, namespace=%s, database=%s",
		username, password, namespace, database,
	)

	dbHost, dbCleanup := prepareDatabase(ctx, tb, username, password)

	client, clientCleanup := prepareClient(ctx, tb, dbHost, username, password, namespace, database, opts...)

	cleanup := func() {
		clientCleanup()
		dbCleanup()
	}

	return client, cleanup
}

func prepareClient(
	ctx context.Context, tb testing.TB, host, username, password, namespace, database string, opts ...Option,
) (
	*Client, func(),
) {
	tb.Helper()

	opts = append(
		[]Option{
			WithLogger(slog.New(newLogger(tb, nil))),
			WithHttpClient(http.DefaultClient),
			WithTimeout(defaultTimeout),
			WithReadLimit(defaultReadLimit),
		},
		opts...,
	)

	client, err := NewClient(ctx,
		Config{
			Host:      host,
			Username:  username,
			Password:  password,
			Namespace: namespace,
			Database:  database,
		},
		opts...,
	)
	if err != nil {
		tb.Fatal(err)
	}

	cleanup := func() {
		if err := client.Close(); err != nil {
			tb.Fatalf("failed to close client: %s", err.Error())
		}
	}

	return client, cleanup
}

func prepareDatabase(
	ctx context.Context, tb testing.TB, username, password string,
) (
	string, func(),
) {
	tb.Helper()

	req := testcontainers.ContainerRequest{
		Name:  "sdbc_" + toSlug(tb.Name()),
		Image: "surrealdb/surrealdb:v" + surrealDBVersion,
		Env: map[string]string{
			"SURREAL_PATH":   "memory",
			"SURREAL_STRICT": "true",
			"SURREAL_AUTH":   "true",
			"SURREAL_USER":   username,
			"SURREAL_PASS":   password,
		},
		Cmd: []string{
			"start", "--allow-funcs", "--log", "trace",
		},
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor:   wait.ForLog(containerStartedMsg),
		HostConfigModifier: func(conf *container.HostConfig) {
			conf.AutoRemove = true
		},
	}

	surreal, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
			Reuse:            true,
			Logger:           &logger{},
		},
	)
	if err != nil {
		tb.Fatal(err)
	}

	host, err := surreal.Endpoint(ctx, "")
	if err != nil {
		tb.Fatal(err)
	}

	cleanup := func() {
		if err := surreal.Terminate(ctx); err != nil {
			if errAlreadyInProgress.MatchString(err.Error()) {
				return // this "error" is not caught by the Terminate method even though it is safe to ignore
			}

			tb.Fatalf("failed to terminate container: %s", err.Error())
		}
	}

	return host, cleanup
}

func toSlug(input string) string {
	// Remove special characters
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		panic(err)
	}
	processedString := reg.ReplaceAllString(input, " ")

	// Remove leading and trailing spaces
	processedString = strings.TrimSpace(processedString)

	// Replace spaces with dashes
	slug := strings.ReplaceAll(processedString, " ", "-")

	// Convert to lowercase
	slug = strings.ToLower(slug)

	return slug
}

type logger struct{}

func (l *logger) Printf(format string, v ...any) {
	slog.Info(fmt.Sprintf(format, v...))
}

//
// -- LOGGER
//

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

//
// -- CONTEXT
//

type testContext struct {
	mu  sync.Mutex
	err error
}

func (t *testContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (t *testContext) Done() <-chan struct{} {
	return make(chan struct{})
}

func (t *testContext) Err() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.err
}

func (t *testContext) Value(_ any) any {
	return nil
}

func (t *testContext) setErr(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.err = err
}

//
// -- HTTP CLIENT
//

type mockHttpClientWithError struct{}

func (m *mockHttpClientWithError) Do(_ *http.Request) (*http.Response, error) {
	return nil, errors.New("mock http client error")
}
