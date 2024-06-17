package sdbc

import (
	"bytes"
	"gotest.tools/v3/assert"
	"testing"
)

func TestStoresGetInvalidAssert(t *testing.T) {
	var pool bufPool

	pool.Pool.Put("invalid type")

	res := pool.Get()

	assert.Check(t, res != nil)
	assert.Check(t, bytes.Equal(res.Bytes(), []byte{}))
}

func TestRequestsUnknownKey(t *testing.T) {
	var req requests

	_, ok := req.get("unknown key")

	assert.Check(t, !ok)
}

func TestRequestsReset(t *testing.T) {
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
	var lq liveQueries

	_, ok := lq.get("unknown_key", false)
	assert.Check(t, !ok)

	lq.store.Store("some_key", "invalid value")
	_, ok = lq.get("some_key", false)
	assert.Check(t, !ok)
}
