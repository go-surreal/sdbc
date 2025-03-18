package sdbc

import (
	"context"
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"gotest.tools/v3/assert"
)

const (
	surrealDBVersion    = "2.2.1"
	containerStartedMsg = "Started web server on "
)

const (
	thingSome = "some"
)

func TestClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)

	defer cleanup()

	_, err := client.Query(ctx, "DEFINE TABLE test SCHEMALESS;", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Create(ctx, NewID("test"), nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvalidNamespaceAndDatabaseNames(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	username := gofakeit.Username()
	password := gofakeit.Password(true, true, true, true, true, 32)

	dbHost, dbCleanup := prepareDatabase(ctx, t, username, password)
	defer dbCleanup()

	namespace := gofakeit.FirstName()
	database := gofakeit.LastName()

	// test invalid namespace name

	invalidNamespace := gofakeit.Name()

	_, err := NewClient(ctx,
		Config{
			Host:      dbHost,
			Username:  username,
			Password:  password,
			Namespace: invalidNamespace,
			Database:  database,
		},
	)

	assert.Check(t, errors.Is(err, ErrInvalidNamespaceName))

	// test invalid database name

	invalidDatabase := gofakeit.Name()

	_, err = NewClient(ctx,
		Config{
			Host:      dbHost,
			Username:  username,
			Password:  password,
			Namespace: namespace,
			Database:  invalidDatabase,
		},
	)

	assert.Check(t, errors.Is(err, ErrInvalidDatabaseName))
}
