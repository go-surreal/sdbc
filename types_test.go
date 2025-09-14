package sdbc

import (
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"gotest.tools/v3/assert"
)

func TestDuration(t *testing.T) {
	t.Parallel()

	in := duration(time.Hour)

	data, err := cbor.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	var out duration

	if err := cbor.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, in, out)
}

func TestDurationErrorCases(t *testing.T) {
	t.Parallel()

	var dur duration

	data, err := cbor.Marshal([]byte("123"))
	if err != nil {
		t.Fatal(err)
	}

	err = cbor.Unmarshal(data, &dur)

	assert.ErrorContains(t, err, "could not unmarshal duration")

	data, err = cbor.Marshal("invalid value")
	if err != nil {
		t.Fatal(err)
	}

	err = cbor.Unmarshal(data, &dur)

	assert.ErrorContains(t, err, "could not parse duration")
}
