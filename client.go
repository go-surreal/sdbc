package sdbc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"
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

	marshal   Marshal
	unmarshal Unmarshal

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

	encTags := cbor.NewTagSet()
	decTags := cbor.NewTagSet()

	enc, err := cbor.EncOptions{}.EncModeWithTags(encTags)
	if err != nil {
		return nil, fmt.Errorf("failed to create cbor encoder: %w", err)
	}

	dec, err := cbor.DecOptions{}.DecModeWithTags(decTags)
	if err != nil {
		return nil, fmt.Errorf("failed to create cbor decoder: %w", err)
	}

	client.marshal = enc.Marshal
	client.unmarshal = dec.Unmarshal

	client.connCtx, client.connCancel = context.WithCancel(ctx)

	if err := client.readVersion(ctx); err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}

	if err := client.openWebsocket(); err != nil {
		return nil, err
	}

	if err := client.init(ctx, conf); err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	return client, nil
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

	res, err := c.httpClient.Do(req)
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
	if c.conn != nil {
		if err := c.conn.Close(websocket.StatusServiceRestart, "reconnect"); err != nil {
			return fmt.Errorf("failed to close websocket connection: %w", err)
		}
	}

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
		Subprotocols:    []string{"cbor"},
		CompressionMode: websocket.CompressionContextTakeover,
	})
	if err != nil {
		return fmt.Errorf("failed to open websocket connection: %w", err)
	}

	conn.SetReadLimit(c.options.readLimit)

	c.conn = conn

	c.waitGroup.Add(1)
	go func() {
		defer c.waitGroup.Done()
		c.subscribe(c.connCtx)
	}()

	return nil
}

func (c *Client) checkWebsocketConn(err error) {
	if err == nil {
		return
	}

	status := websocket.CloseStatus(err)

	if status == -1 || status == websocket.StatusNormalClosure {
		return
	}

	select {

	case <-c.connCtx.Done():
		return

	default:
		{
			c.logger.Error("Websocket connection closed unexpectedly. Trying to reconnect.", "error", err)

			if err := c.openWebsocket(); err != nil {
				c.logger.Error("Could not reconnect to websocket.", "error", err)
			}
		}
	}
}

var (
	regexName = regexp.MustCompile("^[A-Za-z0-9_]+$")

	ErrInvalidNamespaceName = fmt.Errorf("invalid namespace name")
	ErrInvalidDatabaseName  = fmt.Errorf("invalid database name")

	ErrContextNil = errors.New("context is nil")
)

func (c *Client) init(ctx context.Context, conf Config) error {
	if !regexName.MatchString(conf.Namespace) {
		return ErrInvalidNamespaceName
	}

	if !regexName.MatchString(conf.Database) {
		return ErrInvalidDatabaseName
	}

	if err := c.signIn(ctx, conf.Username, conf.Password); err != nil {
		return fmt.Errorf("failed to sign in: %w", err)
	}

	if err := c.use(ctx, conf.Namespace, conf.Database); err != nil {
		return fmt.Errorf("failed to select namespace and database: %w", err)
	}

	resp, err := c.Query(ctx, "DEFINE NAMESPACE IF NOT EXISTS "+conf.Namespace, nil)
	if err != nil {
		return err
	}

	if err := c.checkBasicResponse(resp); err != nil {
		return fmt.Errorf("could not define namespace: %w", err)
	}

	resp, err = c.Query(ctx, "DEFINE DATABASE IF NOT EXISTS "+conf.Database, nil)
	if err != nil {
		return err
	}

	if err := c.checkBasicResponse(resp); err != nil {
		return fmt.Errorf("could not define database: %w", err)
	}

	return nil
}

func (c *Client) checkBasicResponse(resp []byte) error {
	var res []basicResponse[string]

	if err := c.unmarshal(resp, &res); err != nil {
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

func (c *Client) Marshal(val any) ([]byte, error) {
	return c.marshal(val)
}

func (c *Client) Unmarshal(data []byte, val any) error {
	return c.unmarshal(data, val)
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
	if err != nil && !errors.Is(err, net.ErrClosed) {
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
