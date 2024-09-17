package sdbc

import (
	"context"
	"gotest.tools/v3/assert"
	"testing"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	version, err := client.Version(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, surrealDBVersion, version)
}

func TestAuth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	res, err := client.SignUp(ctx, "test", "test", "database", map[string]any{
		"username": "test",
		"password": "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "test", res)

	err = client.signIn(ctx, "test", "test")
	if err != nil {
		t.Fatal(err)
	}

	err = client.Invalidate(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
