package sdbc

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/binary"
	"math/rand/v2"
	"sync"
)

const (
	requestKeyLength = 16
	bytesInUint64    = 8
	charset          = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789" // reduced base64
)

var (
	charsetLen     = len(charset)
	unbiasedMaxVal = byte((256 / charsetLen) * charsetLen)
)

//
// -- BUFFERS
//

type bufPool struct {
	syncPool sync.Pool
}

// get returns a buffer from the pool or
// creates a new one if the pool is empty.
func (p *bufPool) get() *bytes.Buffer {
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

// put returns a buffer into the pool.
func (p *bufPool) put(buf *bytes.Buffer) {
	buf.Reset()

	p.syncPool.Put(buf)
}

//
// -- REQUESTS
//

func newRequests() *requests {
	return &requests{
		store: map[string]chan *output{},
	}
}

type output struct {
	data []byte
	err  error
}

type requests struct {
	mut   sync.RWMutex
	store map[string]chan *output
}

func (r *requests) prepare() (string, <-chan *output) {
	key := newRequestKey()
	outChan := make(chan *output)

	r.mut.Lock()
	defer r.mut.Unlock()

	r.store[key] = outChan

	return key, outChan
}

func (r *requests) get(key string) (chan<- *output, bool) {
	r.mut.RLock()
	defer r.mut.RUnlock()

	outChan, ok := r.store[key]
	if !ok {
		return nil, false
	}

	return outChan, true
}

func (r *requests) del(key string) {
	r.mut.Lock()
	defer r.mut.Unlock()

	if outChan, ok := r.store[key]; ok {
		close(outChan)
		delete(r.store, key)
	}
}

func (r *requests) reset() {
	r.mut.Lock()
	defer r.mut.Unlock()

	for _, outChan := range r.store {
		close(outChan)
	}
	r.store = map[string]chan *output{}
}

func (r *requests) len() int {
	r.mut.RLock()
	defer r.mut.RUnlock()

	return len(r.store)
}

//
// -- LIVE QUERIES
//

func newLiveQueries() *liveQueries {
	return &liveQueries{
		store: map[string]chan []byte{},
	}
}

type liveQueries struct {
	mut   sync.RWMutex
	store map[string]chan []byte
}

func (l *liveQueries) get(key string, create bool) (chan []byte, bool) {
	l.mut.RLock()
	liveChan, ok := l.store[key]
	l.mut.RUnlock()

	if !ok && !create {
		return nil, false
	}

	if !ok {
		newLiveChan := make(chan []byte)

		l.mut.Lock()
		l.store[key] = newLiveChan
		l.mut.Unlock()

		return newLiveChan, true
	}

	return liveChan, true
}

func (l *liveQueries) del(key string) {
	l.mut.Lock()
	defer l.mut.Unlock()

	if liveChan, ok := l.store[key]; ok {
		close(liveChan)
		delete(l.store, key)
	}
}

func (l *liveQueries) reset() {
	l.mut.Lock()
	defer l.mut.Unlock()

	for _, outChan := range l.store {
		close(outChan)
	}
	l.store = map[string]chan []byte{}
}

//
// -- HELPER
//

var defaultRandBytes = newRandBytes()

func newRandBytes() *randBytes {
	randomBytes := make([]byte, bytesInUint64*2)

	if _, err := cryptorand.Read(randomBytes); err != nil {
		panic("unreachable")
	}

	return &randBytes{
		//nolint:gosec // no security required
		rng: rand.New(rand.NewPCG(
			binary.LittleEndian.Uint64(randomBytes[:8]),
			binary.LittleEndian.Uint64(randomBytes[8:]),
		)),
		bytesForUint64: make([]byte, bytesInUint64),
	}
}

type randBytes struct {
	mut            sync.Mutex
	rng            *rand.Rand
	bytesForUint64 []byte
}

// Read fills bytes with random bytes. It never returns an error, and always fills bytes entirely.
func (rb *randBytes) read(bytes []byte) {
	numBytes := len(bytes)
	numUint64s := numBytes / bytesInUint64
	remainingBytes := numBytes % bytesInUint64

	rb.mut.Lock()
	defer rb.mut.Unlock()

	// Fill the slice with 8-byte chunks
	for i := range numUint64s {
		from := i * bytesInUint64
		to := (i + 1) * bytesInUint64
		binary.LittleEndian.PutUint64(bytes[from:to], rb.rng.Uint64())
	}

	// Handle remaining bytes (if any)
	if remainingBytes > 0 {
		binary.LittleEndian.PutUint64(rb.bytesForUint64[0:], rb.rng.Uint64())
		copy(bytes[numUint64s*bytesInUint64:], rb.bytesForUint64[:remainingBytes])
	}
}

func (rb *randBytes) base62Str(length int) string {
	buf := make([]byte, length)

	rb.mut.Lock()
	for i := range buf {
		buf[i] = charset[rb.rng.IntN(charsetLen)]
	}
	rb.mut.Unlock()

	str := string(buf)

	return str
}

// Fastest, but random distribution is not uniform.
// Not security-critical in this case, so acceptable.
func newRequestKey() string {
	key := make([]byte, requestKeyLength)
	defaultRandBytes.read(key)

	for i, b := range key {
		key[i] = charset[int(b)%charsetLen]
	}

	return string(key)
}
