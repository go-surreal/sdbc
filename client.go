package sdbc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"

	schemeWS  = "ws"
	schemeWSS = "wss"

	pathVersion   = "/version"
	pathWebsocket = "/rpc"

	versionPrefix = "surrealdb-"
)

type Client struct {
	*options

	conf    Config
	version string
	token   string

	conn       *websocket.Conn
	connCtx    context.Context //nolint:containedctx // runtime context is used for websocket connection
	connCancel context.CancelFunc
	connMutex  sync.Mutex
	connClosed bool

	waitGroup sync.WaitGroup

	buffers     bufPool
	requests    requests
	liveQueries liveQueries
}

// Config is the configuration for the client.
type Config struct {
	// Host is the host address of the database.
	// It must not contain a protocol or sub path like /rpc.
	Host string

	// Secure indicates whether to use a secure connection (https, wss) or not.
	Secure bool

	// Username is the username to use for authentication.
	Username string

	// Password is the password to use for authentication.
	Password string

	// Namespace is the namespace to use.
	// It will automatically be created if it does not exist.
	Namespace string

	// Database is the database to use.
	// It will automatically be created if it does not exist.
	Database string
}

// NewClient creates a new client and connects to
// the database using a websocket connection.
func NewClient(ctx context.Context, conf Config, opts ...Option) (*Client, error) {
	client := &Client{
		options: applyOptions(opts),
		conf:    conf,
	}

	client.connCtx, client.connCancel = context.WithCancel(ctx)

	if err := client.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return client, nil
}

func (c *Client) connect() error {
	if err := c.readVersion(c.connCtx); err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}

	if err := c.openWebsocket(); err != nil {
		return fmt.Errorf("failed to open websocket: %w", err)
	}

	if err := c.init(); err != nil {
		return fmt.Errorf("failed to initialize client: %w", err)
	}

	return nil
}

func (c *Client) readVersion(ctx context.Context) error {
	requestURL := url.URL{
		Scheme: schemeHTTP,
		Host:   c.conf.Host,
		Path:   pathVersion,
	}

	if c.conf.Secure {
		requestURL.Scheme = schemeHTTPS
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	defer res.Body.Close()

	out, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	c.version = strings.TrimPrefix(string(out), versionPrefix)

	return nil
}

func (c *Client) openWebsocket() error {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	// make sure the previous connection is closed
	//if c.conn != nil {
	//	if err := c.conn.Close(websocket.StatusServiceRestart, "reconnect"); err != nil {
	//		return fmt.Errorf("failed to close websocket connection: %w", err)
	//	}
	//}

	requestURL := url.URL{
		Scheme: schemeWS,
		Host:   c.conf.Host,
		Path:   pathWebsocket,
	}

	if c.conf.Secure {
		requestURL.Scheme = schemeWSS
	}

	//nolint:bodyclose // connection is closed by the client Close() method
	conn, _, err := websocket.Dial(c.connCtx, requestURL.String(), &websocket.DialOptions{
		CompressionMode: websocket.CompressionContextTakeover,
	})
	if err != nil {
		return fmt.Errorf("failed to dial websocket address: %w", err)
	}

	conn.SetReadLimit(c.options.readLimit)

	c.conn = conn

	c.waitGroup.Add(1)
	go func() {
		defer c.waitGroup.Done()
		c.subscribe()
	}()

	return nil
}

func (c *Client) withReconnect(fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}

	if c.connClosed {
		return err
	}

	status := websocket.CloseStatus(err)

	if !errors.Is(err, io.EOF) && (status == -1 || status == websocket.StatusNormalClosure) {
		return err
	}

	select {

	case <-c.connCtx.Done():
		return err

	default:
		{
			c.logger.Error("Websocket connection closed unexpectedly. Trying to reconnect.", "error", err)

			if err := c.connect(); err != nil {
				c.logger.Error("Could not reconnect to websocket.", "error", err)

				return err
			}

			return fn()
		}
	}
}

func (c *Client) init() error {
	if err := c.signIn(c.connCtx, c.conf.Username, c.conf.Password); err != nil {
		return fmt.Errorf("failed to sign in: %w", err)
	}

	if err := c.use(c.connCtx, c.conf.Namespace, c.conf.Database); err != nil {
		return fmt.Errorf("failed to select namespace and database: %w", err)
	}

	resp, err := c.Query(c.connCtx, "DEFINE NAMESPACE "+c.conf.Namespace, nil)
	if err != nil {
		return err
	}

	if err := c.checkBasicResponse(resp); err != nil {
		return fmt.Errorf("failed to define namespace: %w", err)
	}

	resp, err = c.Query(c.connCtx, "DEFINE DATABASE "+c.conf.Database, nil)
	if err != nil {
		return err
	}

	if err := c.checkBasicResponse(resp); err != nil {
		return fmt.Errorf("failed to define database: %w", err)
	}

	return nil
}

func (c *Client) checkBasicResponse(resp []byte) error {
	var res []basicResponse[string]

	if err := c.jsonUnmarshal(resp, &res); err != nil {
		return fmt.Errorf("could not unmarshal response: %w", err)
	}

	if len(res) < 1 {
		return ErrEmptyResponse
	}

	if res[0].Status != "OK" {
		return ErrResponseNotOkay
	}

	return nil
}

func (c *Client) DatabaseVersion() string {
	return c.version
}

// Close closes the client and the websocket connection.
// Furthermore, it cleans up all idle goroutines.
func (c *Client) Close() error {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	if c.connClosed {
		return nil
	}
	c.connClosed = true

	c.logger.Info("Closing client.")

	err := c.conn.Close(websocket.StatusNormalClosure, "closing client")
	if err != nil {
		return fmt.Errorf("could not close websocket connection: %w", err)
	}

	defer c.requests.reset()
	defer c.liveQueries.reset()

	// cancel the connection context
	c.connCancel()

	c.logger.Debug("Waiting for goroutines to finish.")

	waitChan := make(chan struct{})

	go func() {
		defer close(waitChan)
		c.waitGroup.Wait()
	}()

	select {

	case <-waitChan:
		return nil

	case <-time.After(10 * time.Second):
		return ErrTimeoutWaitingForGoroutines
	}
}
