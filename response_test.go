package sdbc

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"gotest.tools/v3/assert"
)

func TestClientSubscribeErrorCases(t *testing.T) {
	t.Parallel()
	prepare(t)

	ctx := context.Background()

	logger := newLogger(t, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	username := gofakeit.Username()
	password := gofakeit.Password(true, true, true, true, true, 32)
	namespace := gofakeit.FirstName()
	database := gofakeit.LastName()

	db, dbCleanup := prepareDatabase(ctx, t, username, password)
	defer dbCleanup()

	client, cleanup := prepareClient(ctx, t, db, username, password, namespace, database, WithLogger(slog.New(logger)))
	defer cleanup()

	testCtx := &testContext{
		err: errors.New("test error"),
	}

	go func() {
		time.Sleep(time.Millisecond)
		testCtx.setErr(context.Canceled)
	}()

	client.subscribe(testCtx)

	assert.Check(t, logger.hasRecordMsg("Could not read from websocket."))
}

func TestClientReadContextNil(t *testing.T) {
	t.Parallel()
	prepare(t)

	client := &Client{}

	_, err := client.read(nil)
	assert.Check(t, errors.Is(err, ErrContextNil))
}

//func TestClientSubscribeContextCanceled(t *testing.T) {
//	t.Parallel()
//
//	done := make(chan struct{})
//
//	go func() {
//		defer close(done)
//
//		ctx := context.Background()
//
//		client, cleanup := prepareSurreal(ctx, t)
//		cleanup()
//
//		ctx, cancel := context.WithCancel(ctx)
//		cancel()
//
//		client.connCtx = ctx
//
//		client.subscribe()
//	}()
//
//	select {
//	case <-done:
//	case <-time.After(5 * time.Second):
//		t.Fatal("test timed out")
//	}
//}
//
//func TestClientSubscribeContextDeadlineExceeded(t *testing.T) {
//	t.Parallel()
//
//	done := make(chan struct{})
//
//	go func() {
//		defer close(done)
//
//		ctx := context.Background()
//
//		client, cleanup := prepareSurreal(ctx, t)
//		cleanup()
//
//		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second))
//		defer cancel()
//
//		client.connCtx = ctx
//
//		client.subscribe()
//	}()
//
//	select {
//	case <-done:
//	case <-time.After(5 * time.Second):
//		t.Fatal("test timed out")
//	}
//}

func TestClientHandleLiveQueryErrorCases(t *testing.T) {
	t.Parallel()
	prepare(t)

	ctx := context.Background()

	logger := newLogger(t, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	client, cleanup := prepareSurreal(ctx, t, WithLogger(slog.New(logger)))
	defer cleanup()

	client.handleLiveQuery(&response{
		Result: []byte("invalid result"),
	})

	assert.Check(t, logger.hasRecordMsg("Could not unmarshal websocket message."))

	result, err := client.marshal(liveQueryID{ID: []byte("unknown_id")})
	if err != nil {
		t.Fatal(err)
	}

	client.handleLiveQuery(&response{
		Result: result,
	})

	assert.Check(t, logger.hasRecordMsg("Could not find live query channel."))
}

func TestClientHandleLiveQueryContextDone(t *testing.T) {
	t.Parallel()
	prepare(t)

	ctx := context.Background()

	logger := newLogger(t, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	username := gofakeit.Username()
	password := gofakeit.Password(true, true, true, true, true, 32)
	namespace := gofakeit.FirstName()
	database := gofakeit.LastName()

	db, dbCleanup := prepareDatabase(ctx, t, username, password)
	defer dbCleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, cleanup := prepareClient(ctx, t, db, username, password, namespace, database, WithLogger(slog.New(logger)))
	defer cleanup()

	result, err := client.marshal(liveQueryID{ID: []byte("known_id")})
	if err != nil {
		t.Fatal(err)
	}

	_, ok := client.liveQueries.get("known_id", true)
	if !ok {
		t.Fatal("Could not create live query channel.")
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		client.handleLiveQuery(&response{
			Result: result,
		})
	}()

	cancel()
	wg.Wait()

	assert.Check(t, !logger.hasRecordMsg("Could not unmarshal websocket message."))
	assert.Check(t, !logger.hasRecordMsg("Could not find live query channel."))

	assert.Check(t, logger.hasRecordMsg("Context done, ignoring live query result."))
}

func TestClientHandleLiveQueryTimeout(t *testing.T) {
	t.Parallel()
	prepare(t)

	ctx := context.Background()

	logger := newLogger(t, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	client, cleanup := prepareSurreal(ctx, t,
		WithTimeout(time.Second),
		WithLogger(slog.New(logger)),
	)
	defer cleanup()

	result, err := client.marshal(liveQueryID{ID: []byte("known_id")})
	if err != nil {
		t.Fatal(err)
	}

	resChan, ok := client.liveQueries.get("known_id", true)
	if !ok {
		t.Fatal("Could not create live query channel.")
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		client.handleLiveQuery(&response{
			Result: result,
		})
	}()

	wg.Wait()

	assert.Check(t, !logger.hasRecordMsg("Could not unmarshal websocket message."))
	assert.Check(t, !logger.hasRecordMsg("Could not find live query channel."))

	assert.Check(t, logger.hasRecordMsg("Timeout while sending result to channel."))

	select {
	case <-resChan:
		t.Fatal("live query channel should not receive anything")
	default:
	}
}
