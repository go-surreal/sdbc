package sdbc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

const (
	surrealDBVersion    = "1.5.4"
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

func TestClientReadVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("surrealdb-" + surrealDBVersion))
	}))
	defer testServer.Close()

	hostUrl, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := &Client{
		options: applyOptions(nil),
		conf: Config{
			Host: fmt.Sprintf("%s:%s", hostUrl.Hostname(), hostUrl.Port()),
		},
	}

	if err = client.readVersion(ctx); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, surrealDBVersion, client.DatabaseVersion())

	err = client.readVersion(nil)
	assert.ErrorContains(t, err, "failed to create request")

	client.options.httpClient = &mockHttpClientWithError{}

	err = client.readVersion(ctx)
	assert.ErrorContains(t, err, "failed to send request")
}

func TestClientCRUD(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
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

	client, cleanup := prepareSurreal(ctx, t)
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

	client, cleanup := prepareSurreal(ctx, t)
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
