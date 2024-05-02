package sdbc

import (
	"context"
	"fmt"
	"math/rand"
	"nhooyr.io/websocket"
	"strings"
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
)

const (
	nilValue = "null"
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
	res, err := c.send(ctx,
		request{
			Method: methodUse,
			Params: []any{
				namespace,
				database,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("could not send request: %w", err)
	}

	if string(res) != nilValue {
		return fmt.Errorf("could not select database due to %s", string(res))
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
		return nil, fmt.Errorf("could not send request: %w", err)
	}

	return res, nil
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func (c *Client) Live(ctx context.Context, query string, vars map[string]any) (<-chan []byte, error) {
	varPrefix := RandStringBytes(32)

	var paramStatements []string

	for key := range vars {
		newKey := fmt.Sprintf("%s_%s", varPrefix, key)

		paramStatements = append(paramStatements, fmt.Sprintf("DEFINE PARAM $%s VALUE $%s", newKey, key))

		query = strings.ReplaceAll(query, fmt.Sprintf("$%s", key), fmt.Sprintf("$%s", newKey))
	}

	query = livePrefix + " " + query

	if len(paramStatements) > 0 {
		query = strings.Join(paramStatements, "; ") + "; " + query
	}

	raw, err := c.send(ctx,
		request{
			Method: methodQuery, // method "live" does not support filtering
			Params: []any{
				query,
				vars,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}

	var res []basicResponse[string]

	if err := c.jsonUnmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w", err)
	}

	resElem := len(paramStatements)

	if len(res) < 1 || res[resElem].Result == "" {
		return nil, fmt.Errorf("empty response")
	}

	liveKey := res[resElem].Result

	ch, ok := c.liveQueries.get(liveKey, true)
	if !ok {
		return nil, fmt.Errorf("could not get live query channel")
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
		var killCtx context.Context

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

		for key := range vars {
			newKey := fmt.Sprintf("%s_%s", varPrefix, key)

			if _, err := c.Query(killCtx, fmt.Sprintf("REMOVE PARAM $%s", newKey), nil); err != nil {
				c.logger.ErrorContext(killCtx, "Could not remove param.", "key", newKey, "error", err)
			}
		}
	}(liveKey)

	return ch, nil
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
		return res, fmt.Errorf("could not send request: %w", err)
	}

	return res, nil
}

// Select a table or record from the database.
func (c *Client) Select(ctx context.Context, thing string) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodSelect,
			Params: []any{
				thing,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}

	return res, nil
}

func (c *Client) Create(ctx context.Context, thing string, data any) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodCreate,
			Params: []any{
				thing,
				data,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}

	return res, nil
}

// Update a table or record in the database like a PUT request.
func (c *Client) Update(ctx context.Context, thing string, data any) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodUpdate,
			Params: []any{
				thing,
				data,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}

	return res, nil
}

// Delete a table or a row from the database like a DELETE request.
func (c *Client) Delete(ctx context.Context, thing string) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodDelete,
			Params: []any{
				thing,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
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

func (c *Client) send(ctx context.Context, req request) (_ []byte, err error) {
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
			return nil, fmt.Errorf("channel closed")
		}

		return res.data, res.err
	}
}

// write writes the JSON message v to c.
// It will reuse buffers in between calls to avoid allocations.
func (c *Client) write(ctx context.Context, req request) (err error) {
	defer c.checkWebsocketConn(err)

	data, err := c.jsonMarshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = c.conn.Write(ctx, websocket.MessageText, data)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	// TODO: use Writer instead of Write to stream the message?
	return nil
}
