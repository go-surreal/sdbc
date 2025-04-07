package sdbc

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"
)

//
// -- BUFFERS
//

type bufPool struct {
	syncPool sync.Pool
}

// Get returns a buffer from the pool or
// creates a new one if the pool is empty.
func (p *bufPool) Get() *bytes.Buffer {
	buf := p.syncPool.Get()

	if buf == nil {
		return bytes.NewBuffer(make([]byte, 0, bytes.MinRead*2))
	}

	bytesBuf, ok := buf.(*bytes.Buffer)
	if !ok {
		return bytes.NewBuffer(make([]byte, 0, bytes.MinRead*2))
	}

	return bytesBuf
}

// Put returns a buffer into the pool.
func (p *bufPool) Put(buf *bytes.Buffer) {
	buf.Reset()

	p.syncPool.Put(buf)
}

//
// -- REQUESTS
//

type output struct {
	data []byte
	err  error
}

type requests struct {
	store sync.Map
}

func (r *requests) prepare() (string, <-chan *output) {
	key := newRequestKey() // TODO: generate multiple keys beforehand and reuse by using sync.Pool?
	ch := make(chan *output)

	r.store.Store(key, ch)

	return key, ch
}

func (r *requests) get(key string) (chan<- *output, bool) {
	val, ok := r.store.Load(key)
	if !ok {
		return nil, false
	}

	outChan, ok := val.(chan *output)
	if !ok {
		return nil, false
	}

	return outChan, true
}

func (r *requests) cleanup(key string) {
	if ch, ok := r.store.LoadAndDelete(key); ok {
		if outChan, ok := ch.(chan *output); ok {
			close(outChan)
		}
	}
}

func (r *requests) reset() {
	r.store.Range(func(key, ch any) bool {
		if outChan, ok := ch.(chan *output); ok {
			close(outChan)
		}

		r.store.Delete(key)

		return true
	})
}

func (r *requests) len() int {
	count := 0

	r.store.Range(func(key, ch any) bool {
		count++

		return true
	})

	return count
}

//
// -- LIVE QUERIES
//

type liveQueries struct {
	store sync.Map
}

func (l *liveQueries) get(key string, create bool) (chan []byte, bool) {
	val, ok := l.store.Load(key)

	if !ok && !create {
		return nil, false
	}

	if !ok {
		ch := make(chan []byte)

		l.store.Store(key, ch)

		return ch, true
	}

	liveChan, ok := val.(chan []byte)
	if !ok {
		return nil, false
	}

	return liveChan, true
}

func (l *liveQueries) del(key string) {
	if ch, ok := l.store.LoadAndDelete(key); ok {
		if liveChan, ok := ch.(chan []byte); ok {
			close(liveChan)
		}
	}
}

func (l *liveQueries) reset() {
	l.store.Range(func(key, ch any) bool {
		if liveChan, ok := ch.(chan []byte); ok {
			close(liveChan)
		}

		l.store.Delete(key)

		return true
	})
}

//
// -- HELPER
//

const (
	requestKeyLength = 16
)

func newRequestKey() string {
	key := make([]byte, requestKeyLength)

	if _, err := rand.Read(key); err != nil {
		return "" // TODO: error?
	}

	return fmt.Sprintf("%X-%X-%X-%X-%X", key[0:4], key[4:6], key[6:8], key[8:10], key[10:])
}
