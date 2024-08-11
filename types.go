package sdbc

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
)

//
// -- ID
//

const (
	expectedArrayLength = 2
	nanosecond          = 1e9
)

const (
	recordSeparator = ":"

	newRand = "rand()"
	newULID = "ulid()"
	newUUID = "uuid()" // TODO: schema type for ID field and cbor tag
)

var (
	ErrTableNameRequired     = errors.New("table name is required")
	ErrDataInvalid           = errors.New("data is invalid")
	ErrUnmarshalNotSupported = errors.New("unmarshal not supported")
)

type ID struct {
	// table is the name of the table in the database.
	table string

	// identifier is the unique identifier of the record.
	// It can be a string, an integer, an array or an object.
	identifier any
}

func (id *ID) recordID() {}

func (id *ID) String() string {
	return fmt.Sprintf("%s%s%s", id.table, recordSeparator, id.identifier)
}

func (id *ID) MarshalCBOR() ([]byte, error) {
	if id.table == "" {
		return nil, ErrTableNameRequired
	}

	if id.identifier == nil {
		content, err := cbor.Marshal(id.table)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal table: %w", err)
		}

		data, err := cbor.Marshal(cbor.RawTag{
			Number:  cborTagTable,
			Content: content,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal raw tag: %w", err)
		}

		return data, nil
	}

	content, err := cbor.Marshal([]any{id.table, id.identifier})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal recordID: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagRecordID,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw tag: %w", err)
	}

	return data, nil
}

func (id *ID) UnmarshalCBOR(data []byte) error {
	var val []any

	err := cbor.Unmarshal(data, &val)
	if err != nil {
		return fmt.Errorf("failed to unmarshal recordID: %w", err)
	}

	if val == nil {
		return nil
	}

	if len(val) != expectedArrayLength {
		return fmt.Errorf("%w: expected %d elements, got %d", ErrDataInvalid, expectedArrayLength, len(val))
	}

	table, ok := val[0].(string)
	if !ok {
		return fmt.Errorf("%w: expected string, got %T", ErrDataInvalid, val[0])
	}

	id.table = table
	id.identifier = val[1]

	//switch identifier := val[1].(type) {
	//
	//case string:
	//	id.identifier = StringID(identifier)
	//
	//case int:
	//	id.identifier = IntID(identifier)
	//
	//default:
	//	return fmt.Errorf("recordID identifier is of unsupported type %T", val[1])
	//}

	return nil
}

//
// -- RECORD ID
//

type RecordID interface {
	recordID()
}

type newRecordID struct {
	table       string
	constructor string
}

func (id *newRecordID) recordID() {}

func (id *newRecordID) MarshalCBOR() ([]byte, error) {
	content, err := cbor.Marshal(id.table + recordSeparator + id.constructor)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal recordID: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagRecordID,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw tag: %w", err)
	}

	return data, nil
}

func (id *newRecordID) UnmarshalCBOR(_ []byte) error {
	return ErrUnmarshalNotSupported
}

func NewID(table string) RecordID {
	return &newRecordID{
		table:       table,
		constructor: newRand,
	}
}

func NewULID(table string) RecordID {
	return &newRecordID{
		table:       table,
		constructor: newULID,
	}
}

func NewUUID(table string) RecordID {
	return &newRecordID{
		table:       table,
		constructor: newUUID,
	}
}

func MakeID(table string, identifier any) *ID {
	return &ID{
		table:      table,
		identifier: identifier,
	}
}

func ParseRecord(record string) (*ID, bool) {
	table, identifier, ok := strings.Cut(record, recordSeparator)

	return &ID{
		table:      table,
		identifier: identifier,
	}, ok
}

//
// -- DATETIME
//

type DateTime struct {
	time.Time
}

func (dt *DateTime) MarshalCBOR() ([]byte, error) {
	if dt == nil {
		data, err := cbor.Marshal(nil) // TODO: is this correct?
		if err != nil {
			return nil, fmt.Errorf("failed to marshal nil: %w", err)
		}

		return data, nil
	}

	content, err := cbor.Marshal([]int64{dt.Unix(), int64(dt.Nanosecond())})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal datetime slice: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagDatetime,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw tag: %w", err)
	}

	return data, nil
}

func (dt *DateTime) UnmarshalCBOR(data []byte) error {
	var val []int64

	if err := cbor.Unmarshal(data, &val); err != nil {
		return fmt.Errorf("failed to unmarshal datetime: %w", err)
	}

	if len(val) < 1 || len(val) > expectedArrayLength {
		return fmt.Errorf("%w: expected 1-2 elements, got %d", ErrDataInvalid, len(val))
	}

	if len(val) < 1 {
		return fmt.Errorf("%w: expected at least one element, got none", ErrDataInvalid)
	}

	secs := val[0]
	nano := int64(0)

	if len(val) > 1 {
		nano = val[1]
	}

	if len(val) > expectedArrayLength {
		return fmt.Errorf("%w: expected at most %d elements, got %d", ErrDataInvalid, expectedArrayLength, len(val))
	}

	dt.Time = time.Unix(secs, nano)

	return nil
}

//
// -- DURATION
//

type Duration struct {
	time.Duration
}

func (d *Duration) MarshalCBOR() ([]byte, error) {
	if d == nil {
		data, err := cbor.Marshal(nil) // TODO: is this correct?
		if err != nil {
			return nil, fmt.Errorf("failed to marshal nil: %w", err)
		}

		return data, nil
	}

	totalSeconds := int64(d.Seconds())
	totalNanoseconds := d.Nanoseconds()
	remainingNanoseconds := totalNanoseconds - (totalSeconds * nanosecond)

	content, err := cbor.Marshal([]int64{totalSeconds, remainingNanoseconds})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal duration slice: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagDuration,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw tag: %w", err)
	}

	return data, nil
}

func (d *Duration) UnmarshalCBOR(data []byte) error {
	var val []int64

	if err := cbor.Unmarshal(data, &val); err != nil {
		return fmt.Errorf("failed to unmarshal duration: %w", err)
	}

	var dur time.Duration

	if len(val) > 0 {
		dur = time.Duration(val[0]) * time.Second
	}

	if len(val) > 1 {
		dur += time.Duration(val[1])
	}

	if len(val) > expectedArrayLength {
		return fmt.Errorf("%w: expected at most %d elements, got %d", ErrDataInvalid, expectedArrayLength, len(val))
	}

	d.Duration = dur

	return nil
}

//
// -- DECIMAL
//

type Decimal struct {
	float64
}

func (d *Decimal) MarshalCBOR() ([]byte, error) {
	if d == nil {
		data, err := cbor.Marshal(nil) // TODO: is this correct?
		if err != nil {
			return nil, fmt.Errorf("failed to marshal nil: %w", err)
		}

		return data, nil
	}

	content, err := cbor.Marshal(d.float64)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal float64: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagDecimal,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw tag: %w", err)
	}

	return data, nil
}

//
// -- BIG INT
//

type BigInt struct {
	big.Int
}

//
// -- BIG FLOAT
//

type BigFloat struct {
	big.Float
}

//
// -- INTERNAL
//

type request struct {
	ID     string `json:"id" cbor:"id"`
	Method string `json:"method" cbor:"method"`
	Params []any  `json:"params" cbor:"params"`
}

type response struct {
	ID     string          `json:"id"`
	Result cbor.RawMessage `json:"result"`
	Error  *responseError  `json:"error"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type liveQueryID struct {
	ID []byte `json:"id"`
}

//
// -- INTERNAL
//

type basicResponse[R any] struct {
	Status string   `json:"status"`
	Result R        `json:"result"`
	Time   duration `json:"time"`
}

type duration time.Duration

func (t duration) MarshalCBOR() ([]byte, error) {
	data, err := cbor.Marshal(time.Duration(t).String())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal duration: %w", err)
	}

	return data, nil
}

func (t *duration) UnmarshalCBOR(data []byte) error {
	var str string

	if err := cbor.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("could not unmarshal duration: %w", err)
	}

	d, err := time.ParseDuration(str)
	if err != nil {
		return fmt.Errorf("could not parse duration: %w", err)
	}

	*t = duration(d)

	return nil
}

func result[T any](t T, err error) resultFunc[T] {
	return func() (T, error) {
		return t, err
	}
}

type resultFunc[T any] func() (T, error)

type resultChannel[T any] chan resultFunc[T]
