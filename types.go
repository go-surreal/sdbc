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
	recordSeparator = ":"

	newRand = "rand()"
	newULID = "ulid()"
	newUUID = "uuid()"
)

type AnyID interface {
	id()
}

type ID struct {

	// table is the name of the table in the database.
	table string

	// identifier is the unique identifier of the record.
	// It can be a string, an integer, an array or an object.
	identifier any

	// new indicates whether the ID is about to be created.
	new bool
}

func newRecord(table string, identifier any) ID {
	return ID{
		table:      table,
		identifier: identifier,
		new:        true,
	}
}

func NewRecord(table string) ID {
	return newRecord(table, newRand)
}

func NewRecordULID(table string) ID {
	return newRecord(table, newULID)
}

func NewRecordUUID(table string) ID {
	return newRecord(table, newUUID)
}

func NewRecordCustom(table string, identifier any) ID {
	return newRecord(table, identifier)
}

func ParseRecord(record string) (ID, bool) {
	table, identifier, ok := strings.Cut(record, recordSeparator)

	return ID{
		table:      table,
		identifier: identifier,
	}, ok
}

func (id *ID) MarshalCBOR() ([]byte, error) {
	if id.table == "" {
		return nil, fmt.Errorf("table name is required")
	}

	if id.new {
		data, err := cbor.Marshal(id.table)
		if err != nil {
			return nil, err
		}

		return cbor.Marshal(cbor.RawTag{
			Number:  cborTagRecordID,
			Content: data,
		})
	}

	if id.identifier == nil {
		data, err := cbor.Marshal(id.table)
		if err != nil {
			return nil, err
		}

		return cbor.Marshal(cbor.RawTag{
			Number:  cborTagTable,
			Content: data,
		})
	}

	data, err := cbor.Marshal([]any{id.table, id.identifier})
	if err != nil {
		return nil, err
	}

	return cbor.Marshal(cbor.RawTag{
		Number:  cborTagRecordID,
		Content: data,
	})
}

func (id *ID) UnmarshalCBOR(data []byte) error {
	var val []any

	err := cbor.Unmarshal(data, &val)
	if err != nil {
		return err
	}

	if val == nil {
		return nil
	}

	if len(val) != 2 {
		return fmt.Errorf("b expected 2 elements, got %d", len(val))
	}

	table, ok := val[0].(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", val[0])
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
// -- DATETIME
//

type DateTime struct {
	time.Time
}

func (dt *DateTime) MarshalCBOR() ([]byte, error) {
	if dt == nil {
		return cbor.Marshal(nil) // TODO: is this correct?
	}

	data, err := cbor.Marshal([]int64{dt.Unix(), int64(dt.Nanosecond())})
	if err != nil {
		return nil, err
	}

	return cbor.Marshal(cbor.RawTag{
		Number:  cborTagDatetime,
		Content: data,
	})
}

func (dt *DateTime) UnmarshalCBOR(data []byte) error {
	var val []int64

	err := cbor.Unmarshal(data, &val)
	if err != nil {
		return err
	}

	if len(val) < 1 || len(val) > 2 {
		return fmt.Errorf("expected 1-2 elements, got %d", len(val))
	}

	if len(val) < 1 {
		return errors.New("expected at least one element, got none")
	}

	secs := val[0]
	nano := int64(0)

	if len(val) > 1 {
		nano = val[1]
	}

	if len(val) > 2 {
		return fmt.Errorf("expected maximum 2 elements, got %d", len(val))
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
		return cbor.Marshal(nil) // TODO: is this correct?
	}

	totalSeconds := int64(d.Seconds())
	totalNanoseconds := d.Nanoseconds()
	remainingNanoseconds := totalNanoseconds - (totalSeconds * 1e9)

	data, err := cbor.Marshal([]int64{totalSeconds, remainingNanoseconds})
	if err != nil {
		return nil, err
	}

	return cbor.Marshal(cbor.RawTag{
		Number:  cborTagDuration,
		Content: data,
	})
}

func (d *Duration) UnmarshalCBOR(data []byte) error {
	var val []int64

	err := cbor.Unmarshal(data, &val)
	if err != nil {
		return err
	}

	var dur time.Duration

	if len(val) > 0 {
		dur = time.Duration(val[0]) * time.Second
	}

	if len(val) > 1 {
		dur += time.Duration(val[1])
	}

	if len(val) > 2 {
		return fmt.Errorf("expected 2 elements, got %d", len(val))
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
		return cbor.Marshal(nil) // TODO: is this correct?
	}

	return cbor.Marshal(cbor.RawTag{
		Number:  cborTagDecimal,
		Content: []byte(fmt.Sprintf("%f", d.float64)),
	})
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
	return cbor.Marshal(time.Duration(t).String())
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
