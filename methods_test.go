package sdbc

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
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
		Duration:  &Duration{time.Second + (5 * time.Nanosecond)},
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
	assert.Check(t, cmp.Equal(*modelIn.Duration, *modelCreate.Duration))

	// QUERY

	res, err = client.Query(ctx, "SELECT * FROM some WHERE id = $id;", map[string]any{
		"id": modelCreate.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	var modelQuery1 []baseResponse[[]someModel]

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

func TestInsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	// DEFINE TABLE

	tableName := "some"

	_, err := client.Query(ctx, "DEFINE TABLE "+tableName+" SCHEMALESS;", nil)
	if err != nil {
		t.Fatal(err)
	}

	// DEFINE MODEL

	modelIn1 := someModel{
		Name: "modelIn1",
	}

	modelIn2 := someModel{
		Name: "modelIn2",
	}

	// INSERT

	res, err := client.Insert(ctx, tableName, []any{modelIn1, modelIn2})
	if err != nil {
		t.Fatal(err)
	}

	var modelInsert []someModel

	err = client.unmarshal(res, &modelInsert)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.Equal(modelIn1.Name, modelInsert[0].Name))
	assert.Check(t, cmp.Equal(modelIn2.Name, modelInsert[1].Name))
}

func TestUpsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	// DEFINE TABLE

	tableName := "some"

	_, err := client.Query(ctx, "DEFINE TABLE "+tableName+" SCHEMALESS;", nil)
	if err != nil {
		t.Fatal(err)
	}

	// DEFINE MODEL

	modelIn := someModel{
		Name: "create",
	}

	// CREATE

	res1, err := client.Upsert(ctx, NewULID(tableName), modelIn)
	if err != nil {
		t.Fatal(err)
	}

	var modelCreated someModel

	err = client.unmarshal(res1, &modelCreated)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.Equal(modelIn.Name, modelCreated.Name))

	// UPDATE

	modelIn.Name = "update"

	res2, err := client.Upsert(ctx, modelCreated.ID, modelIn)
	if err != nil {
		t.Fatal(err)
	}

	var modelUpdated someModel

	err = client.unmarshal(res2, &modelUpdated)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.Equal(modelIn.Name, modelUpdated.Name))
}

func TestMerge(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	// DEFINE TABLE

	tableName := "some"

	_, err := client.Query(ctx, "DEFINE TABLE "+tableName+" SCHEMALESS;", nil)
	if err != nil {
		t.Fatal(err)
	}

	// DEFINE MODEL

	modelCreate := someModel{
		Name:  "create",
		Value: 42,
	}

	modelMerge := map[string]any{
		"name": "merge",
	}

	// CREATE

	res1, err := client.Create(ctx, NewULID(tableName), modelCreate)
	if err != nil {
		t.Fatal(err)
	}

	var modelCreated someModel

	if err = client.unmarshal(res1, &modelCreated); err != nil {
		t.Fatal(err)
	}

	modelCreate.ID = modelCreated.ID

	assert.Equal(t, modelCreate.Name, modelCreated.Name)
	assert.Equal(t, modelCreate.Value, modelCreated.Value)

	// MERGE

	res2, err := client.Merge(ctx, modelCreate.ID, modelMerge)
	if err != nil {
		t.Fatal(err)
	}

	var modelMerged someModel

	err = client.unmarshal(res2, &modelMerged)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, modelMerge["name"], modelMerged.Name)
	assert.Equal(t, modelCreated.Value, modelMerged.Value)
}

func TestPatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	// DEFINE TABLE

	tableName := "some"

	_, err := client.Query(ctx, "DEFINE TABLE "+tableName+" SCHEMALESS;", nil)
	if err != nil {
		t.Fatal(err)
	}

	// DEFINE MODEL

	modelCreate := someModel{
		Name:      "create",
		Slice:     []string{"merge", "copy", ""},
		Duration:  &Duration{time.Minute},
		CreatedAt: DateTime{time.Now()},
	}

	modelPatch := []Patch{
		{
			Op:    OpTest,
			Path:  "name",
			Value: "create",
		},
		{
			Op:    OpAdd,
			Path:  "value",
			Value: 42,
		},
		{
			Op:   OpRemove,
			Path: "duration",
		},
		{
			Op:    OpReplace,
			Path:  "created_at",
			Value: DateTime{modelCreate.CreatedAt.Add(time.Second)},
		},
		{
			Op:   OpCopy,
			From: "slice/1",
			Path: "slice/2",
		},
		{
			Op:   OpMove,
			From: "slice/0",
			Path: "name",
		},
	}

	// CREATE

	res1, err := client.Create(ctx, NewULID(tableName), modelCreate)
	if err != nil {
		t.Fatal(err)
	}

	var modelCreated someModel

	if err = client.unmarshal(res1, &modelCreated); err != nil {
		t.Fatal(err)
	}

	modelCreate.ID = modelCreated.ID

	assert.Equal(t, modelCreate.Name, modelCreated.Name)
	assert.Equal(t, modelCreate.Value, modelCreated.Value)
	assert.Equal(t, modelCreate.Slice[0], modelCreated.Slice[0])
	assert.Equal(t, modelCreate.Slice[1], modelCreated.Slice[1])
	assert.Equal(t, *modelCreate.Duration, *modelCreated.Duration)
	assert.Equal(t, modelCreate.CreatedAt.Format(time.RFC3339), modelCreated.CreatedAt.Format(time.RFC3339))

	// PATCH

	res2, err := client.Patch(ctx, modelCreate.ID, modelPatch, false)
	if err != nil {
		t.Fatal(err)
	}

	var modelPatched someModel

	if err = client.unmarshal(res2, &modelPatched); err != nil {
		t.Fatal(err)
	}

	// map[created_at:1727683084 id:{8 [some 01J90YZFAJNY9F232YTEJ7YMXH]} name:merge slice:[copy] value:42] (modelPatched map[string]interface {})

	assert.Equal(t, 42, modelPatched.Value)
	assert.Check(t, modelPatched.Duration == nil)
	assert.Equal(t, modelCreate.CreatedAt.Add(time.Second).Format(time.RFC3339), modelPatched.CreatedAt.Format(time.RFC3339))
	assert.Equal(t, modelPatched.Slice[1], modelPatched.Slice[0])
	assert.Equal(t, modelPatched.Name, modelCreate.Slice[0])
}

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

func TestRelate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	// DEFINE TABLES

	_, err := client.Query(ctx, "DEFINE TABLE some TYPE NORMAL SCHEMALESS;", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Query(ctx, "DEFINE TABLE other TYPE RELATION SCHEMALESS;", nil)
	if err != nil {
		t.Fatal(err)
	}

	// DEFINE MODEL

	modelIn1 := someModel{
		Name: "modelIn1",
	}

	modelIn2 := someModel{
		Name: "modelIn2",
	}

	// INSERT

	res1, err := client.Insert(ctx, "some", []any{modelIn1, modelIn2})
	if err != nil {
		t.Fatal(err)
	}

	var modelInsert []someModel

	err = client.unmarshal(res1, &modelInsert)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.Equal(modelIn1.Name, modelInsert[0].Name))
	assert.Check(t, cmp.Equal(modelIn2.Name, modelInsert[1].Name))

	// RELATE

	res2, err := client.Relate(ctx, modelInsert[0].ID, NewULID("other"), modelInsert[1].ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	var modelRelate relation

	err = client.unmarshal(res2, &modelRelate)
	if err != nil {
		t.Fatal(err)
	}

	res3, err := client.Select(ctx, modelRelate.ID)
	if err != nil {
		t.Fatal(err)
	}

	var modelSelect relation

	err = client.unmarshal(res3, &modelSelect)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, modelRelate.ID.String(), modelSelect.ID.String())
}

func TestInsertRelation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	// DEFINE TABLES

	_, err := client.Query(ctx, "DEFINE TABLE some TYPE NORMAL SCHEMALESS;", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Query(ctx, "DEFINE TABLE other TYPE RELATION SCHEMALESS;", nil)
	if err != nil {
		t.Fatal(err)
	}

	// DEFINE MODEL

	modelIn1 := someModel{
		Name: "modelIn1",
	}

	modelIn2 := someModel{
		Name: "modelIn2",
	}

	// INSERT

	res1, err := client.Insert(ctx, "some", []any{modelIn1, modelIn2})
	if err != nil {
		t.Fatal(err)
	}

	var modelInsert []someModel

	err = client.unmarshal(res1, &modelInsert)
	if err != nil {
		t.Fatal(err)
	}

	assert.Check(t, cmp.Equal(modelIn1.Name, modelInsert[0].Name))
	assert.Check(t, cmp.Equal(modelIn2.Name, modelInsert[1].Name))

	// INSERT RELATION

	rel := relation{
		In:  modelInsert[0].ID,
		Out: modelInsert[1].ID,
	}

	table := "other"

	res2, err := client.InsertRelation(ctx, &table, rel)
	if err != nil {
		t.Fatal(err)
	}

	var modelRelation []relation

	err = client.unmarshal(res2, &modelRelation)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, modelRelation[0].In.String(), modelInsert[0].ID.String())
	assert.Equal(t, modelRelation[0].Out.String(), modelInsert[1].ID.String())

	// INSERT NEXT RELATION

	rel.ID = modelRelation[0].ID

	res3, err := client.InsertRelation(ctx, nil, rel)
	if err != nil {
		t.Fatal(err)
	}

	var modelRelation2 []relation

	err = client.unmarshal(res3, &modelRelation2)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLetUnset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	if err := client.Let(ctx, "some_var", 42); err != nil {
		t.Fatal(err)
	}

	raw1, err := client.Query(ctx, "RETURN $some_var;", nil)
	if err != nil {
		t.Fatal(err)
	}

	var res1 []baseResponse[int]

	if err = client.unmarshal(raw1, &res1); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 42, res1[0].Result)

	if err := client.Let(ctx, "some_var", 21); err != nil {
		t.Fatal(err)
	}

	raw2, err := client.Query(ctx, "RETURN $some_var;", nil)
	if err != nil {
		t.Fatal(err)
	}

	var res2 []baseResponse[int]

	if err = client.unmarshal(raw2, &res2); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 21, res2[0].Result)

	if err := client.Unset(ctx, "some_var"); err != nil {
		t.Fatal(err)
	}

	raw3, err := client.Query(ctx, "RETURN $some_var;", nil)
	if err != nil {
		t.Fatal(err)
	}

	var res3 []baseResponse[int]

	if err = client.unmarshal(raw3, &res3); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, res3[0].Result)
}

func TestRun(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, cleanup := prepareSurreal(ctx, t)
	defer cleanup()

	raw, err := client.Run(ctx, "time::now", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	var res DateTime

	if err = client.unmarshal(raw, &res); err != nil {
		t.Fatal(err)
	}

	assert.Check(t, time.Since(res.Time) < time.Second)
}

func TestGraphQL(t *testing.T) {
	// TODO: test graphql implementation (not so important right now)
}

//
// -- TYPES
//

type baseResponse[T any] struct {
	Result T      `cbor:"result"`
	Status string `cbor:"status"`
	Time   string `cbor:"time"`
}

type liveResponse[T any] struct {
	ID     []byte `cbor:"id"`
	Action string `cbor:"action"`
	Result T      `cbor:"result"`
}

type someModel struct {
	ID    *ID      `cbor:"id,omitempty"`
	Name  string   `cbor:"name"`
	Value int      `cbor:"value"`
	Slice []string `cbor:"slice"`

	CreatedAt DateTime  `cbor:"created_at"`
	Duration  *Duration `cbor:"duration"`
}

type relation struct {
	ID  *ID `cbor:"id,omitempty"`
	In  *ID `cbor:"in"`
	Out *ID `cbor:"out"`
}
