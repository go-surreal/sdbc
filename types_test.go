package sdbc

import (
	"encoding/json"
	"fmt"
	"gotest.tools/v3/assert"
	"testing"
	"time"
)

func TestDuration(t *testing.T) {
	testValue := fmt.Sprintf(`"%s"`, time.Hour.String())

	var dur duration

	if err := json.Unmarshal([]byte(testValue), &dur); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, duration(time.Hour), dur)
}

func TestDurationErrNotNil(t *testing.T) {
	var dur duration

	err := json.Unmarshal([]byte("123"), &dur)

	assert.ErrorContains(t, err, "could not unmarshal duration")

	err = json.Unmarshal([]byte(`"invalid value"`), &dur)

	assert.ErrorContains(t, err, "could not parse duration")
}
