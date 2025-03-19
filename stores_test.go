package sdbc

import (
	"bytes"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestStoresGetInvalidAssert(t *testing.T) {
	t.Parallel()

	var pool bufPool

	pool.Pool.Put("invalid type")

	res := pool.Get()

	assert.Check(t, res != nil)
	assert.Check(t, bytes.Equal(res.Bytes(), []byte{}))
}

func TestRequestsUnknownKey(t *testing.T) {
	t.Parallel()

	var req requests

	_, ok := req.get("unknown key")

	assert.Check(t, !ok)
}

func TestRequestsInvalidTypeCast(t *testing.T) {
	t.Parallel()

	var req requests

	req.store.Store("some_key", "invalid value")

	_, ok := req.get("some_key")

	assert.Check(t, !ok)
}

func TestRequestsReset(t *testing.T) {
	t.Parallel()

	var req requests

	req.prepare()
	req.store.Store("some_key", "invalid value")

	assert.Equal(t, 2, req.len())

	req.reset()

	_, ok := req.get("some_key")

	assert.Check(t, !ok)
	assert.Equal(t, 0, req.len())
}

func TestLiveQueriesGetErrorCases(t *testing.T) {
	t.Parallel()

	var lq liveQueries

	_, ok := lq.get("unknown_key", false)
	assert.Check(t, !ok)

	lq.store.Store("some_key", "invalid value")
	_, ok = lq.get("some_key", false)
	assert.Check(t, !ok)
}

func TestLiveQueriesDel(t *testing.T) {
	t.Parallel()

	var lq liveQueries

	ch, ok := lq.get("some_key", true)
	assert.Check(t, ok)

	lq.del("some_key")

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		t.Fatal("channel not closed")
	}
}
