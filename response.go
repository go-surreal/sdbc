package sdbc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"nhooyr.io/websocket"
)

const (
	logArgID = "id"
)

func (c *Client) subscribe(ctx context.Context) {
	resChan := make(resultChannel[[]byte])

	c.waitGroup.Add(1)
	go func(resChan resultChannel[[]byte]) {
		defer c.waitGroup.Done()

		defer close(resChan)

		for {
			buf, err := c.read(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}

				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) || websocket.CloseStatus(err) != -1 {
					c.logger.InfoContext(ctx, "Websocket closed.")

					return
				}

				c.logger.ErrorContext(ctx, "Could not read from websocket.", "error", err)

				continue
			}

			resChan <- result(buf, nil)
		}
	}(resChan)

	c.handleMessages(resChan)
}

// read reads a single websocket message.
// It will reuse buffers in between calls to avoid allocations.
func (c *Client) read(ctx context.Context) ([]byte, error) {
	if ctx == nil {
		return nil, ErrContextNil
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context done: %w", ctx.Err())
	}

	var err error
	defer c.checkWebsocketConn(err)

	msgType, reader, err := c.conn.Reader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get reader: %w", err)
	}

	if msgType != websocket.MessageText {
		return nil, fmt.Errorf("%w, got %v", ErrExpectedTextMessage, msgType)
	}

	buff := c.buffers.Get()
	defer c.buffers.Put(buff)

	if _, err = buff.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	return buff.Bytes(), nil
}

func (c *Client) handleMessages(resultCh resultChannel[[]byte]) {
	for {
		select {

		case <-c.connCtx.Done():
			{
				c.logger.DebugContext(c.connCtx, "Context done. Stopping message handler.")

				return
			}

		case result, more := <-resultCh:
			{
				if !more {
					c.logger.DebugContext(c.connCtx, "Result channel closed. Stopping message handler.")

					return
				}

				c.waitGroup.Add(1)
				go func() {
					defer c.waitGroup.Done()

					data, err := result()
					if err != nil {
						c.logger.ErrorContext(c.connCtx, "Could not get result from channel.", "error", err)

						return
					}

					c.handleMessage(data)
				}()
			}
		}
	}
}

func (c *Client) handleMessage(data []byte) {
	var res *response

	if err := c.jsonUnmarshal(data, &res); err != nil {
		c.logger.ErrorContext(c.connCtx, "Could not unmarshal websocket message.", "error", err)

		return
	}

	c.logger.DebugContext(c.connCtx, "Received message.",
		"id", res.ID,
		"result", string(res.Result),
		"error", res.Error,
	)

	if res.ID == "" {
		c.handleLiveQuery(res)

		return
	}

	c.handleResult(res)
}

func (c *Client) handleResult(res *response) {
	outCh, ok := c.requests.get(res.ID)
	if !ok {
		c.logger.ErrorContext(c.connCtx, "Could not find pending request for ID.", logArgID, res.ID)

		return
	}

	var err error
	if res.Error != nil {
		err = fmt.Errorf("%w: (%d) %s", ErrResultWithError, res.Error.Code, res.Error.Message)
	}

	select {

	case outCh <- &output{data: res.Result, err: err}:
		return

	case <-c.connCtx.Done():
		return

	case <-time.After(c.timeout):
		c.logger.ErrorContext(c.connCtx, "Timeout while sending result to channel.", logArgID, res.ID)
	}
}

func (c *Client) handleLiveQuery(res *response) {
	var rawID liveQueryID

	if err := c.jsonUnmarshal(res.Result, &rawID); err != nil {
		c.logger.ErrorContext(c.connCtx, "Could not unmarshal websocket message.", "error", err)

		return
	}

	outCh, ok := c.liveQueries.get(rawID.ID, false)
	if !ok {
		c.logger.ErrorContext(c.connCtx, "Could not find live query channel.", logArgID, rawID.ID)

		return
	}

	select {

	case outCh <- res.Result:
		c.logger.DebugContext(c.connCtx, "Sent live query result to channel.", logArgID, rawID.ID)

	case <-c.connCtx.Done():
		c.logger.DebugContext(c.connCtx, "Context done, ignoring live query result.", logArgID, rawID.ID)

	case <-time.After(c.timeout):
		c.logger.ErrorContext(c.connCtx, "Timeout while sending result to channel.", logArgID, res.ID)
	}
}
