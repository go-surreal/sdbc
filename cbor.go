package sdbc

import (
	"fmt"
	"reflect"

	"github.com/fxamacker/cbor/v2"
	"github.com/google/uuid"
)

const (
	// Tags from the spec - https://www.iana.org/assignments/cbor-tags/cbor-tags.xhtml

	cborTagDateTime = 0
	cborTagUUID     = 37

	// Custom tags.

	cborTagNone           = 6
	cborTagTable          = 7
	cborTagRecordID       = 8
	cborTagStringUUID     = 9
	cborTagStringDecimal  = 10
	cborTagBinaryDecimal  = 11
	cborTagCustomDatetime = 12
	cborTagStringDuration = 13
	cborTagCustomDuration = 14

	// Custom Geometries.

	cborTagGeometryPoint        = 88
	cborTagGeometryLine         = 89
	cborTagGeometryPolygon      = 90
	cborTagGeometryMultiPoint   = 91
	cborTagGeometryMultiLine    = 92
	cborTagGeometryMultiPolygon = 93
	cborTagGeometryCollection   = 94
)

func tags() (cbor.TagSet, error) {
	opts := cbor.TagOptions{cbor.DecTagRequired, cbor.EncTagRequired}

	tags := cbor.NewTagSet()

	//err := tags.Add(opts, reflect.TypeOf(table("")), cborTagTable)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to add tag: %w", err)
	//}

	err := tags.Add(opts, reflect.TypeOf(uuid.UUID{}), cborTagUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to add tag: %w", err)
	}

	return tags, nil
}
