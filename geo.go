package sdbc

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/twpayne/go-geom"
)

type GeoPoint geom.Point

func (p *GeoPoint) MarshalCBOR() ([]byte, error) {
	if p == nil {
		return nil, nil
	}

	content, err := cbor.Marshal(p.Coords())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoPoint: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagGeoPoint,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoPoint: %w", err)
	}

	return data, nil
}

type GeoLineString geom.LineString

func (l *GeoLineString) MarshalCBOR() ([]byte, error) {
	if l == nil {
		return nil, nil
	}

	content, err := cbor.Marshal(l.Coords())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoLineString: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagGeoLine,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoLineString: %w", err)
	}

	return data, nil
}

type GeoPolygon geom.Polygon

func (p *GeoPolygon) MarshalCBOR() ([]byte, error) {
	if p == nil {
		return nil, nil
	}

	content, err := cbor.Marshal(p.Coords())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoPolygon: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagGeoPolygon,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoPolygon: %w", err)
	}

	return data, nil
}

type GeoMultiPoint geom.MultiPoint

func (p *GeoMultiPoint) MarshalCBOR() ([]byte, error) {
	if p == nil {
		return nil, nil
	}

	content, err := cbor.Marshal(p.Coords())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoMultiPoint: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagGeoMultiPoint,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoMultiPoint: %w", err)
	}

	return data, nil
}

type GeoMultiLineString geom.MultiLineString

func (l *GeoMultiLineString) MarshalCBOR() ([]byte, error) {
	if l == nil {
		return nil, nil
	}

	content, err := cbor.Marshal(l.Coords())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoMultiLineString: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagGeoMultiLine,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoMultiLineString: %w", err)
	}

	return data, nil
}

type GeoMultiPolygon geom.MultiPolygon

func (p *GeoMultiPolygon) MarshalCBOR() ([]byte, error) {
	if p == nil {
		return nil, nil
	}

	content, err := cbor.Marshal(p.Coords())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoMultiPolygon: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagGeoMultiPolygon,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoMultiPolygon: %w", err)
	}

	return data, nil
}

type GeoCollection struct {
	geom.GeometryCollection
}

func (c *GeoCollection) MarshalCBOR() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	content, err := cbor.Marshal(c.Geoms())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoCollection: %w", err)
	}

	data, err := cbor.Marshal(cbor.RawTag{
		Number:  cborTagGeoCollection,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoCollection: %w", err)
	}

	return data, nil
}
