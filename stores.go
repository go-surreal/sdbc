package sdbc

import (
	"bytes"
	"sync"

	"github.com/google/uuid"
)

//
// -- BUFFERS
//

type bufPool struct {
	sync.Pool
}

// Get returns a buffer from the pool or
// creates a new one if the pool is empty.
func (p *bufPool) Get() *bytes.Buffer {
	buf := p.Pool.Get()

	if buf == nil {
		return &bytes.Buffer{}
	}

	bytesBuf, ok := buf.(*bytes.Buffer)
	if !ok {
		return &bytes.Buffer{}
	}

	return bytesBuf
}

// Put returns a buffer into the pool.
func (p *bufPool) Put(buf *bytes.Buffer) {
	buf.Reset()

	p.Pool.Put(buf)
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
	key := uuid.New()
	ch := make(chan *output)

	r.store.Store(key.String(), ch)

	return key.String(), ch
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
