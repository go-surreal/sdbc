package sdbc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/fxamacker/cbor/v2"
)

const (
	CborMinNestedLevels = 4
	CborMaxNestedLevels = 65535
	
	CborMinArrayElements = 16
	CborMaxArrayElements = 2147483647

	CborMinMapPairs = 16
	CborMaxMapPairs = 2147483647
)

const (
	schemeWS  = "ws"
	schemeWSS = "wss"

	pathWebsocket = "/rpc"
)

var (
	regexName = regexp.MustCompile("^[A-Za-z0-9_]+$")

	ErrInvalidNamespaceName = errors.New("invalid namespace name")
	ErrInvalidDatabaseName  = errors.New("invalid database name")

	ErrContextNil = errors.New("context is nil")
)

type Client struct {
	*options

	marshal   Marshal
	unmarshal Unmarshal

	conf  Config
	token string

	conn       *websocket.Conn
	connCtx    context.Context //nolint:containedctx // runtime context is used for websocket connection
	connCancel context.CancelFunc
	connMutex  sync.Mutex
	connClosed bool

	waitGroup sync.WaitGroup

	buffers     bufPool
	requests    *requests
	liveQueries *liveQueries
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

	// CborMaxNestedLevels specifies the max nested levels allowed for any combination of CBOR array, maps, and tags.
	// Default is 32 levels, minimum is 4, maximum is 65535. Note that higher maximum levels of nesting can
	// require larger amounts of stack to deserialize. Don't increase this higher than you require.
	CborMaxNestedLevels int

	// CborMaxArrayElements specifies the max number of elements for CBOR arrays.
	// Default is 128*1024=131072, minimum is 16, maximum is 2147483647.
	CborMaxArrayElements int

	// CborMaxMapPairs specifies the max number of key-value pairs for CBOR maps.
	// Default is 128*1024=131072, minimum is 16, maximum is 2147483647.
	CborMaxMapPairs int
}

// NewClient creates a new client and connects to
// the database using a websocket connection.
func NewClient(ctx context.Context, conf Config, opts ...Option) (*Client, error) {
	client := &Client{
		options: applyOptions(opts),
		conf:    conf,
	}

	encOpts := cbor.EncOptions{}

	decOpts := cbor.DecOptions{
		MaxNestedLevels:   conf.CborMaxNestedLevels,
		MaxArrayElements:  conf.CborMaxArrayElements,
		MaxMapPairs:       conf.CborMaxMapPairs,
		DupMapKey:         cbor.DupMapKeyQuiet,    // let the database handle that
		ExtraReturnErrors: cbor.ExtraDecErrorNone, // ignore missing fields (currently required)
		UTF8:              cbor.UTF8RejectInvalid, // reject invalid UTF-8
	}

	encTags := cbor.NewTagSet()
	decTags := cbor.NewTagSet()

	enc, err := encOpts.EncModeWithTags(encTags)
	if err != nil {
		return nil, fmt.Errorf("failed to create cbor encoder: %w", err)
	}

	dec, err := decOpts.DecModeWithTags(decTags)
	if err != nil {
		return nil, fmt.Errorf("failed to create cbor decoder: %w", err)
	}

	client.marshal = enc.Marshal
	client.unmarshal = dec.Unmarshal

	client.requests = newRequests()
	client.liveQueries = newLiveQueries()

	client.connCtx, client.connCancel = context.WithCancel(ctx)

	if err := client.openWebsocket(); err != nil {
		return nil, err
	}

	if err := client.init(ctx, conf); err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	return client, nil
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

	conn.SetReadLimit(c.readLimit)

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
	if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) {
		// TODO: is it really properly closed despite the io.EOF error?
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
