package sdbc

import (
	"github.com/fxamacker/cbor/v2"
	"gotest.tools/v3/assert"
	"testing"
	"time"
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

	err := cbor.Unmarshal([]byte("123"), &dur)

	assert.ErrorContains(t, err, "could not unmarshal duration")

	err = cbor.Unmarshal([]byte("invalid value"), &dur)

	assert.ErrorContains(t, err, "could not parse duration")
}
