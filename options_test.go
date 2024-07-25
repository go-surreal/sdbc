package sdbc

import (
	"gotest.tools/v3/assert"
	"log/slog"
	"testing"
)

func TestEmptyLogHandler(t *testing.T) {
	t.Parallel()

	handler := emptyLogHandler{}.
		WithAttrs(nil).
		WithGroup("group")

	assert.Check(t, !handler.Enabled(nil, 0))
	assert.NilError(t, handler.Handle(nil, slog.Record{}))
}
