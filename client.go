package sdbc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"nhooyr.io/websocket"
	"sync"
	"time"
)

type Client struct {
	*options

	conf  Config
	token string

	conn       *websocket.Conn
	connCtx    context.Context
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

	// Address is the address of the database.
	Address string

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
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	return client, nil
}

func (c *Client) connect() error {
	if err := c.openWebsocket(); err != nil {
		return fmt.Errorf("could not open websocket: %w", err)
	}

	if err := c.init(); err != nil {
		return fmt.Errorf("could not initialize client: %v", err)
	}

	return nil
}

func (c *Client) openWebsocket() error {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	// make sure the previous connection is closed
	// if c.conn != nil {
	// 	if err := c.conn.Close(websocket.StatusServiceRestart, "reconnect"); err != nil {
	// 		return fmt.Errorf("could not close websocket connection: %w", err)
	// 	}
	// }

	conn, _, err := websocket.Dial(c.connCtx, c.conf.Address, &websocket.DialOptions{
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
		return fmt.Errorf("could not sign in: %v", err)
	}

	if err := c.use(c.connCtx, c.conf.Namespace, c.conf.Database); err != nil {
		return fmt.Errorf("could not select namespace and database: %v", err)
	}

	response, err := c.Query(c.connCtx, "define namespace "+c.conf.Namespace, nil)
	if err != nil {
		return err
	}

	if err := c.checkBasicResponse(response); err != nil {
		return fmt.Errorf("could not define namespace: %w", err)
	}

	response, err = c.Query(c.connCtx, "define database "+c.conf.Database, nil)
	if err != nil {
		return err
	}

	if err := c.checkBasicResponse(response); err != nil {
		return fmt.Errorf("could not define database: %w", err)
	}

	return nil
}

func (c *Client) checkBasicResponse(resp []byte) error {
	var res []basicResponse[string]

	if err := c.jsonUnmarshal(resp, &res); err != nil {
		return fmt.Errorf("could not unmarshal response: %w", err)
	}

	if len(res) < 1 {
		return fmt.Errorf("empty response")
	}

	if res[0].Status != "OK" {
		return fmt.Errorf("response status is not OK")
	}

	return nil
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
		return fmt.Errorf("could not close websocket connection: %v", err)
	}

	defer c.requests.reset()
	defer c.liveQueries.reset()

	// cancel the connection context
	c.connCancel()

	c.logger.Debug("Waiting for goroutines to finish.")

	ch := make(chan struct{})

	go func() {
		defer close(ch)
		c.waitGroup.Wait()
	}()

	select {

	case <-ch:
		return nil

	case <-time.After(10 * time.Second):
		return errors.New("internal goroutines did not finish in time")
	}
}
