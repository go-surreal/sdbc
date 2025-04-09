package sdbc

import (
	"bytes"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestStoresGetInvalidAssert(t *testing.T) {
	t.Parallel()

	var pool bufPool

	pool.syncPool.Put("invalid type")

	res := pool.Get()

	assert.Check(t, res != nil)
	assert.Check(t, bytes.Equal(res.Bytes(), []byte{}))
}

func TestRequestsUnknownKey(t *testing.T) {
	t.Parallel()

	var req *requests = NewRequests()

	_, ok := req.get("unknown key")

	assert.Check(t, !ok)
}

func TestRequestsReset(t *testing.T) {
	t.Parallel()

	var req *requests = NewRequests()

	req.prepare()
	req.prepare()

	assert.Equal(t, 2, req.len())

	req.reset()

	_, ok := req.get("some_key")

	assert.Check(t, !ok)
	assert.Equal(t, 0, req.len())
}

func TestLiveQueriesGetErrorCases(t *testing.T) {
	t.Parallel()

	var lq *liveQueries = NewLiveQueries()

	_, ok := lq.get("unknown_key", false)
	assert.Check(t, !ok)
}

func TestLiveQueriesDel(t *testing.T) {
	t.Parallel()

	var lq *liveQueries = NewLiveQueries()

	ch, ok := lq.get("some_key", true)
	assert.Check(t, ok)

	lq.del("some_key")

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		t.Fatal("channel not closed")
	}
}

func TestNewRandBytes(t *testing.T) {
	t.Parallel()
	// Basic test to ensure that NewRandBytes doesn't panic.
	_ = NewRandBytes()
}

func TestRandBytes_Read(t *testing.T) {
	rb := NewRandBytes()

	t.Run("Full Uint64s", func(t *testing.T) {
		t.Parallel()

		origBytes := make([]byte, 16)
		b := make([]byte, len(origBytes))
		copy(b, origBytes)

		rb.Read(b)

		assert.Check(t,
			!bytes.Equal(b, origBytes),
			"Bytes were not modified",
		)
	})

	t.Run("Partial Uint64", func(t *testing.T) {
		t.Parallel()

		origBytes := make([]byte, 5)
		b := make([]byte, len(origBytes))
		copy(b, origBytes)

		rb.Read(b)

		assert.Check(t,
			!bytes.Equal(b, origBytes),
			"Bytes were not modified",
		)
	})

	t.Run("Zero Length", func(t *testing.T) {
		t.Parallel()

		b := make([]byte, 0)

		rb.Read(b) // Should not panic

		assert.Check(t,
			len(b) == 0,
			"Length of bytes should be 0, but is %d", len(b),
		)
	})

	t.Run("Multiple Reads", func(t *testing.T) {
		t.Parallel()

		numBytes := rand.Uint32N(256) + 1

		bytes1 := make([]byte, numBytes)
		bytes2 := make([]byte, numBytes)
		rb.Read(bytes1)
		rb.Read(bytes2)
		// Check if the two byte slices are different.
		// This doesn't guarantee randomness, but it does ensure the generator is advancing.
		assert.Check(t,
			!bytes.Equal(bytes1, bytes2),
			"Multiple reads returned the same bytes",
		)
	})

	t.Run("Large Read", func(t *testing.T) {
		t.Parallel()

		origBytes := make([]byte, 1024)
		b := make([]byte, len(origBytes))
		copy(b, origBytes)

		rb.Read(b)

		assert.Check(t,
			!bytes.Equal(b, origBytes),
			"Bytes were not modified",
		)
	})

	t.Run("Concurrency", func(t *testing.T) {
		t.Parallel()

		const numGoRoutines = 24
		wg := sync.WaitGroup{}
		for range numGoRoutines {
			wg.Add(1)
			go func() {
				defer wg.Done()

				origBytes := make([]byte, rand.Uint32N(256)+1)
				origLen := len(origBytes)
				origCap := cap(origBytes)

				b := make([]byte, origLen)
				copy(b, origBytes)

				rb.Read(b)

				assert.Equal(t, len(b), origLen)
				assert.Equal(t, cap(b), origCap)
				assert.Check(t,
					!bytes.Equal(b, origBytes),
					"Bytes were not modified",
				)
			}()
		}
		wg.Wait()
	})
}

func TestRandBytes_Base62Str(t *testing.T) {
	rb := NewRandBytes()

	t.Run("Concurrency", func(t *testing.T) {
		t.Parallel()

		const numGoRoutines = 24
		wg := sync.WaitGroup{}
		for range numGoRoutines {
			wg.Add(1)
			go func() {
				defer wg.Done()

				strLen := int(rand.Int32N(256) + 1)
				str := rb.Base62Str(strLen)

				assert.Equal(t, len(str), strLen)
			}()
		}
		wg.Wait()
	})
}
