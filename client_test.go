package sdbc

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

const (
	surrealDBVersion    = "2.0.0-alpha.1"
	containerStartedMsg = "Started web server on "
)

const (
	thingSome = "some"
)

func TestClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareDatabase(ctx, t)
	defer cleanup()

	assert.Equal(t, surrealDBVersion, client.DatabaseVersion())

	_, err := client.Query(ctx, "define table test schemaless", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Create(ctx, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientCRUD(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareDatabase(ctx, t)
	defer cleanup()

	// DEFINE TABLE

	_, err := client.Query(ctx, "define table some schemaless", nil)
	if err != nil {
		t.Fatal(err)
	}

	// CREATE

	modelIn := someModel{
		Name:  "some_name",
		Value: 42, //nolint:revive // test value
		Slice: []string{"a", "b", "c"},
	}

	res, err := client.Create(ctx, thingSome, modelIn)
	if err != nil {
		t.Fatal(err)
	}

	var modelCreate []someModel

	err = json.Unmarshal(res, &modelCreate)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, is.Equal(modelIn.Name, modelCreate[0].Name))
	assert.Check(t, is.Equal(modelIn.Value, modelCreate[0].Value))
	assert.Check(t, is.DeepEqual(modelIn.Slice, modelCreate[0].Slice))

	// QUERY

	res, err = client.Query(ctx, "select * from some where id = $id", map[string]any{
		"id": modelCreate[0].ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	var modelQuery1 []baseResponse[someModel]

	err = json.Unmarshal(res, &modelQuery1)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, is.DeepEqual(modelCreate[0], modelQuery1[0].Result[0]))

	// UPDATE

	modelIn.Name = "some_other_name"

	res, err = client.Update(ctx, thingSome, modelIn)
	if err != nil {
		t.Fatal(err)
	}

	var modelUpdate []someModel

	err = json.Unmarshal(res, &modelUpdate)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, is.Equal(modelIn.Name, modelUpdate[0].Name))

	// SELECT

	res, err = client.Select(ctx, modelUpdate[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	var modelSelect someModel

	err = json.Unmarshal(res, &modelSelect)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, is.DeepEqual(modelIn.Name, modelSelect.Name))

	// DELETE

	res, err = client.Delete(ctx, modelCreate[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	var modelDelete someModel

	err = json.Unmarshal(res, &modelDelete)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, is.DeepEqual(modelUpdate[0], modelDelete))
}

func TestClientLive(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareDatabase(ctx, t)
	defer cleanup()

	// DEFINE TABLE

	_, err := client.Query(ctx, "define table some schemaless", nil)
	if err != nil {
		t.Fatal(err)
	}

	// DEFINE MODEL

	modelIn := someModel{
		Name:  "some_name",
		Value: 42, //nolint:revive // test value
		Slice: []string{"a", "b", "c"},
	}

	// LIVE QUERY

	live, err := client.Live(ctx, "select * from some", nil)
	if err != nil {
		t.Fatal(err)
	}

	liveResChan := make(chan *someModel)
	liveErrChan := make(chan error)

	go func() {
		defer close(liveResChan)
		defer close(liveErrChan)

		for liveOut := range live {
			var liveRes liveResponse[someModel]

			if err := json.Unmarshal(liveOut, &liveRes); err != nil {
				liveResChan <- nil
				liveErrChan <- err

				return
			}

			liveResChan <- &liveRes.Result
			liveErrChan <- nil
		}
	}()

	// CREATE

	res, err := client.Create(ctx, thingSome, modelIn)
	if err != nil {
		t.Fatal(err)
	}

	var modelCreate []someModel

	err = json.Unmarshal(res, &modelCreate)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, is.Equal(modelIn.Name, modelCreate[0].Name))
	assert.Check(t, is.Equal(modelIn.Value, modelCreate[0].Value))
	assert.Check(t, is.DeepEqual(modelIn.Slice, modelCreate[0].Slice))

	liveRes := <-liveResChan
	liveErr := <-liveErrChan

	assert.Check(t, is.Nil(liveErr))
	assert.Check(t, is.DeepEqual(modelCreate[0], *liveRes))
}

func TestClientLiveFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareDatabase(ctx, t)
	defer cleanup()

	// DEFINE TABLE

	_, err := client.Query(ctx, "define table some schemaless", nil)
	if err != nil {
		t.Fatal(err)
	}

	// DEFINE MODEL

	modelIn := someModel{
		Name:  "some_name",
		Value: 42, //nolint:revive // test value
		Slice: []string{"a", "b", "c"},
	}

	// LIVE QUERY

	live, err := client.Live(ctx, "select * from some where name in $0", map[string]any{
		"0": []string{"some_name", "some_other_name"},
	})
	if err != nil {
		t.Fatal(err)
	}

	liveResChan := make(chan *someModel)
	liveErrChan := make(chan error)

	go func() {
		defer close(liveResChan)
		defer close(liveErrChan)

		for liveOut := range live {
			var liveRes liveResponse[someModel]

			if err := json.Unmarshal(liveOut, &liveRes); err != nil {
				liveResChan <- nil
				liveErrChan <- err

				return
			}

			liveResChan <- &liveRes.Result
			liveErrChan <- nil
		}
	}()

	// CREATE

	res, err := client.Create(ctx, thingSome, modelIn)
	if err != nil {
		t.Fatal(err)
	}

	var modelCreate []someModel

	err = json.Unmarshal(res, &modelCreate)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, is.Equal(modelIn.Name, modelCreate[0].Name))
	assert.Check(t, is.Equal(modelIn.Value, modelCreate[0].Value))
	assert.Check(t, is.DeepEqual(modelIn.Slice, modelCreate[0].Slice))

	liveRes := <-liveResChan
	liveErr := <-liveErrChan

	assert.Check(t, is.Nil(liveErr))
	assert.Check(t, is.DeepEqual(modelCreate[0], *liveRes))
}

//
// -- TYPES
//

type baseResponse[T any] struct {
	Result []T    `json:"result"`
	Status string `json:"status"`
	Time   string `json:"time"`
}

type liveResponse[T any] struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Result T      `json:"result"`
}

type someModel struct {
	ID    string   `json:"id,omitempty"`
	Name  string   `json:"name"`
	Value int      `json:"value"`
	Slice []string `json:"slice"`
}

//
// -- HELPER
//

func prepareDatabase(ctx context.Context, tb testing.TB) (*Client, func()) {
	tb.Helper()

	username := gofakeit.Username()
	password := gofakeit.Password(true, true, true, true, true, 32)
	namespace := gofakeit.FirstName()
	database := gofakeit.LastName()

	req := testcontainers.ContainerRequest{
		Name:  "sdbc_" + toSlug(tb.Name()),
		Image: "surrealdb/surrealdb:v" + surrealDBVersion,
		Env: map[string]string{
			"SURREAL_PATH":   "memory",
			"SURREAL_STRICT": "true",
			"SURREAL_AUTH":   "true",
			"SURREAL_USER":   username,
			"SURREAL_PASS":   password,
		},
		Cmd: []string{
			"start", "--allow-funcs", "--log", "trace",
		},
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor:   wait.ForLog(containerStartedMsg),
		HostConfigModifier: func(conf *container.HostConfig) {
			conf.AutoRemove = true
		},
	}

	surreal, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
			Reuse:            true,
		},
	)
	if err != nil {
		tb.Fatal(err)
	}

	host, err := surreal.Endpoint(ctx, "")
	if err != nil {
		tb.Fatal(err)
	}

	client, err := NewClient(ctx,
		Config{
			Host:      host,
			Username:  username,
			Password:  password,
			Namespace: namespace,
			Database:  database,
		},
		WithLogger(
			slog.New(
				slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
			),
		),
	)
	if err != nil {
		tb.Fatal(err)
	}

	cleanup := func() {
		if err := client.Close(); err != nil {
			tb.Fatalf("failed to close client: %s", err.Error())
		}

		if err := surreal.Terminate(ctx); err != nil {
			tb.Fatalf("failed to terminate container: %s", err.Error())
		}
	}

	return client, cleanup
}

func toSlug(input string) string {
	// Remove special characters
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		panic(err)
	}
	processedString := reg.ReplaceAllString(input, " ")

	// Remove leading and trailing spaces
	processedString = strings.TrimSpace(processedString)

	// Replace spaces with dashes
	slug := strings.ReplaceAll(processedString, " ", "-")

	// Convert to lowercase
	slug = strings.ToLower(slug)

	return slug
}
