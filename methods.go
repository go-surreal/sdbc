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
	methodUse          = "use"
	methodSignUp       = "signup"
	methodSignIn       = "signin"
	methodInfo         = "info"
	methodAuthenticate = "authenticate"
	methodInvalidate   = "invalidate"

	methodCreate = "create"
	methodInsert = "insert"
	methodUpdate = "update"
	methodUpsert = "upsert"
	methodMerge  = "merge"
	methodPatch  = "patch"
	methodDelete = "delete"
	methodSelect = "select"

	methodRelate = "relate"

	methodQuery = "query"

	livePrefix = "live"
	methodKill = "kill"

	methodLet   = "let"
	methodUnset = "unset"

	randomVariablePrefixLength = 32
)

// use specifies or unsets the namespace and/or database for the current connection.
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
		return fmt.Errorf("failed to use ns/db: %w", err)
	}

	return nil
}

//
// -- AUTH
//

// SignUp a user using the SIGNUP query defined in a record access method.
func (c *Client) SignUp(ctx context.Context, ns, db, ac string, vars map[string]any) ([]byte, error) {
	params := make(map[string]any, len(vars)+3)

	params["NS"] = ns
	params["DB"] = db
	params["AC"] = ac

	for key, val := range vars {
		params[key] = val
	}

	res, err := c.send(ctx,
		request{
			Method: methodSignUp,
			Params: []any{
				params,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign up: %w", err)
	}

	return res, nil
}

// signIn a root, NS, DB or record user against SurrealDB.
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
		return fmt.Errorf("failed to sign in: %w", err)
	}

	c.token = string(res)

	return nil
}

// Info	returns the record of an authenticated record user.
// TODO: return parsed info struct
func (c *Client) Info(ctx context.Context) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodInfo,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth info: %w", err)
	}

	return res, nil
}

func (c *Client) Authenticate(ctx context.Context, token string) error {
	_, err := c.send(ctx,
		request{
			Method: methodAuthenticate,
			Params: []any{
				token,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to authenticate token: %w", err)
	}

	return nil
}

// Invalidate a user's session for the current connection.
func (c *Client) Invalidate(ctx context.Context) error {
	_, err := c.send(ctx,
		request{
			Method: methodInvalidate,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to invalidate user: %w", err)
	}

	return nil
}

//
// -- CRUD
//

// Create a record with a random or specified ID.
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
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	return res, nil
}

// Insert one or multiple records in a table.
// TODO: allow for fixed IDs.
// TODO: "INSERT RELATION" statement for relations.
func (c *Client) Insert(ctx context.Context, table string, data []any) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodInsert,
			Params: []any{
				table,
				data,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert records: %w", err)
	}

	return res, nil
}

// Update modifies either all records in a table or a single
// record with specified data if the record already exists.
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
		return nil, fmt.Errorf("failed to update record: %w", err)
	}

	return res, nil
}

// Upsert replaces either all records in a table or a single record with specified data.
// Note: Only supported by SurrealDB v2.0.0 and later.
func (c *Client) Upsert(ctx context.Context, id *ID, data any) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodUpsert,
			Params: []any{
				id,
				data,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert record: %w", err)
	}

	return res, nil
}

// Merge specified data into either all records in a table or a single record.
// TODO: support "all" records
func (c *Client) Merge(ctx context.Context, thing *ID, data any) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodMerge,
			Params: []any{
				thing,
				data,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to merge record(s): %w", err)
	}

	return res, nil
}

// Patch either all records in a table or a single record with specified patches.
// see: https://jsonpatch.com/
func (c *Client) Patch(ctx context.Context, thing *ID, patches []patch, diff bool) ([]byte, error) {
	res, err := c.send(ctx,
		request{
			Method: methodPatch,
			Params: []any{
				thing,
				patches,
				diff,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to patch record(s): %w", err)
	}

	return res, nil
}

// Delete either all records in a table or a single record.
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

// Select either all records in a table or a single record
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

// TODO: SelectRange
// https://surrealdb.com/docs/surrealdb/surrealql/functions/database/type#typerange

//
// -- RELATIONS
//

// Relate creates a graph relationship between two records.
// TODO: Use Query to create relations with additional data.
func (c *Client) Relate(ctx context.Context, in, out, relation string) error {
	_, err := c.send(ctx,
		request{
			Method: methodRelate,
			Params: []any{
				in,
				out,
				relation,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to relate records: %w", err)
	}

	return nil
}

//
// -- QUERY
//

// Query executes a custom query with optional variables.
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

//
// -- LIVE
//

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

// Kill an active live query.
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

//
// -- MISC
//

// Let defines a variable on the current connection.
func (c *Client) Let(ctx context.Context, name string, value any) error {
	_, err := c.send(ctx,
		request{
			Method: methodLet,
			Params: []any{
				name,
				value,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to set variable: %w", err)
	}

	return nil
}

// Unset removes a variable from the current connection.
func (c *Client) Unset(ctx context.Context, name string) error {
	_, err := c.send(ctx,
		request{
			Method: methodUnset,
			Params: []any{
				name,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to unset variable: %w", err)
	}

	return nil
}

//
// -- TYPES
//

type signInParams struct {
	User string `cbor:"user"`
	Pass string `cbor:"pass"`
}

type patch struct {
	Op    string `cbor:"op"`
	Path  string `cbor:"path"`
	Value any    `cbor:"value"`
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
