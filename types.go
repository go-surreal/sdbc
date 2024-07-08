package sdbc

import (
	"fmt"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
)

type ID interface {
	id()
}

type StringID string

func (StringID) id() {}

type IntID int

func (IntID) id() {}

type RecordID struct {
	_          struct{} `cbor:",toarray"`
	Table      string
	Identifier ID
}

func (id *RecordID) id() {}

func (id *RecordID) MarshalCBOR() ([]byte, error) {
	if id.Table == "" {
		return nil, fmt.Errorf("table name is required")
	}

	if id.Identifier == nil {
		tagNumber := cborTagTable

		if strings.Contains(id.Table, ":") {
			tagNumber = cborTagRecordID
		}

		data, err := cbor.Marshal(id.Table)
		if err != nil {
			return nil, err
		}

		tag := cbor.RawTag{
			Number:  uint64(tagNumber),
			Content: data,
		}

		return cbor.Marshal(tag)
	}

	data, err := cbor.Marshal([]any{id.Table, id.Identifier})
	if err != nil {
		return nil, err
	}

	return cbor.Marshal(cbor.RawTag{
		Number:  cborTagRecordID,
		Content: data,
	})
}

func (id *RecordID) UnmarshalCBOR(data []byte) error {
	var val []any

	err := cbor.Unmarshal(data, &val)
	if err != nil {
		return err
	}

	if len(val) != 2 {
		return fmt.Errorf("expected 2 elements, got %d", len(val))
	}

	table, ok := val[0].(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", val[0])
	}

	id.Table = table

	switch identifier := val[1].(type) {

	case string:
		id.Identifier = StringID(identifier)

	case int:
		id.Identifier = IntID(identifier)

	default:
		return fmt.Errorf("recordID identifier is of unsupported type %T", val[1])
	}

	return nil
}

// -- INTERNAL

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
