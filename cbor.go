package sdbc

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// IANA spec: https://www.iana.org/assignments/cbor-tags/cbor-tags.xhtml
// SurrealDB spec: https://surrealdb.com/docs/surrealdb/integration/cbor

const (
	// CBORTagNone represents a NONE value.
	// The value passed to the tagged value is null, as it cannot be empty.
	CBORTagNone = 6

	// cborTagTable represents a table name as a string.
	cborTagTable = 7

	// cborTagRecordID represents a RecordID as a two-value array, containing
	// a table part (string) and an id part (string, number, object or array).
	cborTagRecordID = 8

	// cborTagDecimal represents a Decimal in a string format.
	cborTagDecimal = 10

	// cborTagDatetime represents a DateTime as a two-value array,
	// containing seconds (number) and optionally nanoseconds (number).
	// It is preferred by SurrealDB over the standard IANA tag 0.
	cborTagDatetime = 12

	// cborTagDuration represents a Duration as a two-value array, containing
	// optionally seconds (number) and optionally nanoseconds (number).
	// An empty array will be considered a Duration of 0.
	// It is used instead of custom tag 13 (string representation).
	cborTagDuration = 14

	// CBORTagUUID represents a UUID in binary form.
	// It is adopted from the IANA specification.
	// It is preferred by SurrealDB over custom tag 9 (string).
	//
	// Please note: This const is exposed and not implemented by this package.
	// This decision was made to minimize the number of dependencies.
	// There is no UUID type in the standard library.
	// As a third-party package, github.com/google/uuid is recommended.
	CBORTagUUID = 37

	// Custom Geometries:.

	// A Geometry Point represented by a two-value array containing a lat (float) and lon (float).
	cborTagGeoPoint = 88.

	// A Geometry Line represented by an array with two or more points (Tag 88).
	cborTagGeoLine = 89.

	// A Geometry Polygon represented by an array with one or more closed lines (Tag 89).
	// If the lines are not closed, meaning that the first and last point are equal,
	// then SurrealDB will automatically suffix the line with its first point.
	cborTagGeoPolygon = 90.

	// A Geometry MultiPoint represented by an array with one or more points (Tag 88).
	cborTagGeoMultiPoint = 91.

	// A Geometry MultiLine represented by an array with one or more lines (Tag 89).
	cborTagGeoMultiLine = 92.

	// A Geometry MultiPolygon represented by an array with one or more polygons (Tag 90).
	cborTagGeoMultiPolygon = 93.

	// A Geometry Collection represented by an array with one or more geometry values
	// (Tag 88, Tag 89, Tag 90, Tag 91, Tag 92, Tag 93 or Tag 94).
	cborTagGeoCollection = 94.
)

var encodedNull = []byte{0xf6}

type ZeroAsNone[T comparable] struct {
	Value T
}

func (n *ZeroAsNone[T]) MarshalCBOR() ([]byte, error) {
	var zero T

	if n.Value == zero {
		data, err := cbor.Marshal(cbor.RawTag{
			Number:  CBORTagNone,
			Content: encodedNull,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal none: %w", err)
		}

		return data, nil
	}

	data, err := cbor.Marshal(n.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value: %w", err)
	}

	return data, nil
}

func (n *ZeroAsNone[T]) UnmarshalCBOR(data []byte) error {
	var tag cbor.RawTag

	if err := cbor.Unmarshal(data, &tag); err != nil {
		return fmt.Errorf("failed to unmarshal tag: %w", err)
	}

	if tag.Number == CBORTagNone {
		return nil
	}

	if err := cbor.Unmarshal(tag.Content, &n.Value); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}
