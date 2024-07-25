package sdbc

import (
	"errors"
	"fmt"

	"nhooyr.io/websocket"
)

var (
	ErrChannelClosed               = errors.New("channel closed")
	ErrCouldNotGetLiveQueryChannel = errors.New("could not get live query channel")
	ErrCouldNotSelectDatabase      = errors.New("could not select database")
	ErrEmptyResponse               = errors.New("empty response")
	ErrExpectedTextMessage         = fmt.Errorf("expected message of type text (%d)", websocket.MessageBinary)
	ErrResponseNotOkay             = errors.New("response status is not OK")
	ErrResultWithError             = errors.New("result contains error")
	ErrTimeoutWaitingForGoroutines = errors.New("internal goroutines did not finish in time")
)
