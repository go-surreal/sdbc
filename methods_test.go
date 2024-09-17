package sdbc

import (
	"context"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"testing"
	"time"
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

func TestCRUD(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	// DEFINE TABLE

	sql := `
		DEFINE TABLE some SCHEMAFULL TYPE NORMAL;

		DEFINE FIELD name ON some TYPE string;
		DEFINE FIELD value ON some TYPE int;
		DEFINE FIELD slice ON some TYPE array<string>;

		DEFINE FIELD created_at ON some TYPE datetime;
		DEFINE FIELD duration ON some TYPE duration;
	`

	_, err := client.Query(ctx, sql, nil)
	if err != nil {
		t.Fatal(err)
	}

	// CREATE

	modelIn := someModel{
		Name:      "some_name",
		Value:     42, //nolint:revive // test value
		Slice:     []string{"a", "b", "c"},
		CreatedAt: DateTime{time.Now()},
		Duration:  Duration{time.Second + (5 * time.Nanosecond)},
	}

	res, err := client.Create(ctx, NewULID(thingSome), modelIn)
	if err != nil {
		t.Fatal(err)
	}

	var modelCreate someModel

	err = client.unmarshal(res, &modelCreate)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.Equal(modelIn.Name, modelCreate.Name))
	assert.Check(t, cmp.Equal(modelIn.Value, modelCreate.Value))
	assert.Check(t, cmp.DeepEqual(modelIn.Slice, modelCreate.Slice))
	assert.Check(t, cmp.Equal(modelIn.CreatedAt.Format(time.RFC3339), modelCreate.CreatedAt.Format(time.RFC3339)))
	assert.Check(t, cmp.Equal(modelIn.Duration, modelCreate.Duration))

	// QUERY

	res, err = client.Query(ctx, "SELECT * FROM some WHERE id = $id;", map[string]any{
		"id": modelCreate.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	var modelQuery1 []baseResponse[someModel]

	err = client.unmarshal(res, &modelQuery1)
	if err != nil {
		t.Fatal(err)
	}

	if len(modelQuery1[0].Result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(modelQuery1[0].Result))
	}

	assert.Check(t, cmp.DeepEqual(modelCreate, modelQuery1[0].Result[0], cmpopts.IgnoreUnexported(ID{}, DateTime{})))

	// UPDATE

	modelIn.Name = "some_other_name"

	res, err = client.Update(ctx, modelCreate.ID, modelIn)
	if err != nil {
		t.Fatal(err)
	}

	var modelUpdate someModel

	err = client.unmarshal(res, &modelUpdate)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.Equal(modelIn.Name, modelUpdate.Name))

	// SELECT

	res, err = client.Select(ctx, modelUpdate.ID)
	if err != nil {
		t.Fatal(err)
	}

	var modelSelect someModel

	err = client.unmarshal(res, &modelSelect)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.DeepEqual(modelIn.Name, modelSelect.Name))

	// DELETE

	res, err = client.Delete(ctx, modelCreate.ID)
	if err != nil {
		t.Fatal(err)
	}

	var modelDelete someModel

	err = client.unmarshal(res, &modelDelete)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.DeepEqual(modelUpdate.ID, modelDelete.ID, cmpopts.IgnoreUnexported(ID{})))
}

func TestInsert(t *testing.T) {}

func TestUpsert(t *testing.T) {}

func TestMerge(t *testing.T) {}

func TestPatch(t *testing.T) {}

func TestLive(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	// DEFINE TABLE

	_, err := client.Query(ctx, "DEFINE TABLE some SCHEMALESS;", nil)
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

	live, err := client.Live(ctx, "SELECT * FROM some;", nil)
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

			if err := client.unmarshal(liveOut, &liveRes); err != nil {
				liveErrChan <- err
				liveResChan <- nil

				return
			}

			liveErrChan <- nil
			liveResChan <- &liveRes.Result
		}
	}()

	// CREATE

	res, err := client.Create(ctx, NewID(thingSome), modelIn)
	if err != nil {
		t.Fatal(err)
	}

	var modelCreate someModel

	err = client.unmarshal(res, &modelCreate)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.Equal(modelIn.Name, modelCreate.Name))
	assert.Check(t, cmp.Equal(modelIn.Value, modelCreate.Value))
	assert.Check(t, cmp.DeepEqual(modelIn.Slice, modelCreate.Slice))

	if liveErr := <-liveErrChan; liveErr != nil {
		t.Fatal(liveErr)
	}

	liveRes := <-liveResChan

	assert.Check(t, cmp.DeepEqual(modelCreate, *liveRes, cmpopts.IgnoreUnexported(ID{})))
}

func TestLiveFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	// DEFINE TABLE

	_, err := client.Query(ctx, "DEFINE TABLE some SCHEMALESS;", nil)
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

	live, err := client.Live(ctx, "SELECT * FROM some WHERE name IN $a;", map[string]any{
		"a": []string{"some_name", "some_other_name"},
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

			if err := client.unmarshal(liveOut, &liveRes); err != nil {
				liveResChan <- nil
				liveErrChan <- err

				return
			}

			liveResChan <- &liveRes.Result
			liveErrChan <- nil
		}
	}()

	// CREATE

	res, err := client.Create(ctx, NewID(thingSome), modelIn)
	if err != nil {
		t.Fatal(err)
	}

	var modelCreate someModel

	err = client.unmarshal(res, &modelCreate)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.Equal(modelIn.Name, modelCreate.Name))
	assert.Check(t, cmp.Equal(modelIn.Value, modelCreate.Value))
	assert.Check(t, cmp.DeepEqual(modelIn.Slice, modelCreate.Slice))

	select {

	case liveRes := <-liveResChan:
		assert.Check(t, cmp.DeepEqual(modelCreate, *liveRes, cmpopts.IgnoreUnexported(ID{})))

	case <-time.After(1 * time.Second):
		t.Fatal("timeout")
	}

	select {

	case liveErr := <-liveErrChan:
		assert.Check(t, cmp.Nil(liveErr))

	case <-time.After(1 * time.Second):
		t.Fatal("timeout")
	}
}

func TestKill(t *testing.T) {}

func TestRelate(t *testing.T) {}

func TestInsertRelation(t *testing.T) {}

func TestLetUnset(t *testing.T) {}

func TestRun(t *testing.T) {}

func TestGraphQL(t *testing.T) {}

//
// -- TYPES
//

type baseResponse[T any] struct {
	Result []T    `json:"result"`
	Status string `json:"status"`
	Time   string `json:"time"`
}

type liveResponse[T any] struct {
	ID     []byte `json:"id"`
	Action string `json:"action"`
	Result T      `json:"result"`
}

type someModel struct {
	//_     struct{} `cbor:",toarray"`
	ID    *ID      `cbor:"id,omitempty"`
	Name  string   `json:"name"`
	Value int      `json:"value"`
	Slice []string `json:"slice"`

	CreatedAt DateTime `json:"created_at"`
	Duration  Duration `json:"duration"`
}
