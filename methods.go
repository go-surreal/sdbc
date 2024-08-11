package sdbc

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/coder/websocket"

	"golang.org/x/exp/maps"
)

const (
	methodSignIn = "signin"
	methodUse    = "use"
	methodQuery  = "query"
	methodKill   = "kill"
	methodUpdate = "update"
	methodDelete = "delete"
	methodSelect = "select"
	methodCreate = "create"

	livePrefix = "live"

	randomVariablePrefixLength = 32
)

// signIn is a helper method for signing in a user.
func (c *Client) signIn(ctx context.Context, username, password string) error {
	res, err := c.send(ctx,
		request{
			Method: methodSignIn,
			Params: []any{
				signInParams{
					User: username,
					Pass: password,
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("could not sign in: %w", err)
	}

	c.token = string(res)

	return nil
}

// use is a method to select the namespace and table for the connection.
func (c *Client) use(ctx context.Context, namespace, database string) error {
	_, err := c.send(ctx,
		request{
			Method: methodUse,
			Params: []any{
				namespace,
				database,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return nil
}

// Query is a convenient method for sending a query to the database.
func (c *Client) Query(ctx context.Context, query string, vars map[string]any) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodQuery,
			Params: []any{
				query,
				vars,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return res, nil
}

// Live executes a live query request and returns a channel to receive the results.
//
// NOTE: SurrealDB does not yet support proper variable handling for live queries.
// To circumvent this limitation, params are registered in the database before issuing
// the actual live query. Those params are given the values of the variables passed to
// this method. This way, the live query can be filtered by said params.
// Please note that this is a workaround and may not work as expected in all cases.
//
// References:
// Bug: Using variables in filters does not emit live messages (https://github.com/surrealdb/surrealdb/issues/2623)
// Bug: LQ params should be evaluated before registering (https://github.com/surrealdb/surrealdb/issues/2641)
// Bug: parameters do not work with live queries (https://github.com/surrealdb/surrealdb/issues/3602)
//
// TODO: prevent query from being more than one statement.
func (c *Client) Live(ctx context.Context, query string, vars map[string]any) (<-chan []byte, error) {
	varPrefix, err := randString(randomVariablePrefixLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random string: %w", err)
	}

	params := make(map[string]string, len(vars))

	for key := range vars {
		newKey := varPrefix + "_" + key
		params[newKey] = "DEFINE PARAM $" + newKey + " VALUE $" + key
		query = strings.ReplaceAll(query, "$"+key, "$"+newKey)
	}

	query = livePrefix + " " + query

	if len(params) > 0 {
		query = strings.Join(maps.Values(params), "; ") + "; " + query
	}

	raw, err := c.send(ctx,
		request{
			Method: methodQuery,
			Params: []any{
				query,
				vars,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var res []basicResponse[[]byte]

	if err := c.unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w", err)
	}

	// The last response contains the live key.
	queryIndex := len(params)

	if len(res) < 1 || string(res[queryIndex].Result) == "" {
		return nil, ErrEmptyResponse
	}

	liveKey := res[queryIndex].Result

	liveChan, ok := c.liveQueries.get(string(liveKey), true)
	if !ok {
		return nil, ErrCouldNotGetLiveQueryChannel
	}

	c.waitGroup.Add(1)
	go func(key string) {
		defer c.waitGroup.Done()

		select {

		case <-c.connCtx.Done():
			// No kill needed, because the connection is already closed.
			return

		case <-ctx.Done():
			c.logger.DebugContext(ctx, "Context done, closing live query channel.", "key", key)
		}

		c.liveQueries.del(key)

		// Find the best context to kill the live query with.
		var killCtx context.Context //nolint:contextcheck // assigned in switch below

		switch {

		case ctx.Err() == nil:
			killCtx = ctx

		case c.connCtx.Err() == nil:
			killCtx = c.connCtx

		default:
			killCtx = context.Background()
		}

		if _, err := c.Kill(killCtx, key); err != nil {
			c.logger.ErrorContext(killCtx, "Could not kill live query.", "key", key, "error", err)
		}

		for newKey := range params {
			if _, err := c.Query(killCtx, fmt.Sprintf("REMOVE PARAM $%s;", newKey), nil); err != nil {
				c.logger.ErrorContext(killCtx, "Could not remove param.", "key", newKey, "error", err)
			}
		}
	}(string(liveKey))

	return liveChan, nil
}

func (c *Client) Kill(ctx context.Context, uuid string) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodKill,
			Params: []any{
				uuid,
			},
		},
	)
	if err != nil {
		return res, fmt.Errorf("failed to send request: %w", err)
	}

	return res, nil
}

// Select a table or record from the database.
func (c *Client) Select(ctx context.Context, id *ID) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodSelect,
			Params: []any{
				id,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return res, nil
}

func (c *Client) Create(ctx context.Context, id RecordID, data any) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodCreate,
			Params: []any{
				id,
				data,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return res, nil
}

// Update a table or record in the database like a PUT request.
func (c *Client) Update(ctx context.Context, id *ID, data any) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodUpdate,
			Params: []any{
				id,
				data,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return res, nil
}

// Delete a table or a row from the database like a DELETE request.
func (c *Client) Delete(ctx context.Context, id *ID) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodDelete,
			Params: []any{
				id,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return res, nil
}

//
// -- TYPES
//

type signInParams struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

//
// -- INTERNAL
//

func (c *Client) send(ctx context.Context, req request) ([]byte, error) {
	var err error
	defer c.checkWebsocketConn(err)

	reqID, resCh := c.requests.prepare()
	defer c.requests.cleanup(reqID)

	req.ID = reqID

	c.logger.DebugContext(ctx, "Sending request.",
		"id", req.ID,
		"method", req.Method,
		"params", req.Params,
	)

	if err := c.write(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to write to websocket: %w", err)
	}

	select {

	case <-ctx.Done():
		return nil, fmt.Errorf("context done: %w", ctx.Err())

	case res, more := <-resCh:
		if !more {
			return nil, ErrChannelClosed
		}

		return res.data, res.err
	}
}

// write writes the JSON message v to c.
// It will reuse buffers in between calls to avoid allocations.
func (c *Client) write(ctx context.Context, req request) error {
	var err error
	defer c.checkWebsocketConn(err)

	data, err := c.marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = c.conn.Write(ctx, websocket.MessageBinary, data)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	// TODO: use Writer instead of Write to stream the message?
	return nil
}

//
// -- HELPER
//

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func randString(n int) (string, error) {
	byteSlice := make([]byte, n)

	for index := range byteSlice {
		randInt, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random string: %w", err)
		}

		byteSlice[index] = letterBytes[randInt.Int64()]
	}

	return string(byteSlice), nil
}
