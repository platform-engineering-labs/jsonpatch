package jsonpatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var simpleObjEntitySet = `{"a":100, "t":[{"k":1, "v":1},{"k":2, "v":2}]}`
var simpleObjAddEntitySetItem = `{"t":[{"k":3, "v":3}]}`
var simpleObjModifyEntitySetItem = `{"t":[{"k":2, "v":3}]}`
var simpleObjAddDuplicateEntitySetItem = `{"t":[{"k":2, "v":2}]}`
var simpleObjAddMultipleDuplicateAndFailedItems = `{"t":[{"k":1, "v":1},{"k":2, "v":2},{"k":3, "v":3},{"k":4, "v":4}]}`
var simpleObjEntitySetRemoveItem = `{"a":100, "t":[{"k":1, "v":1}]}`
var complexNextedEntitySet = `{
    "a":100,
    "t":[
    {"k":1,
    "v":[
    {"nk":11, "c":"x", "d":[1,2], "e":"f"},
    {"nk":22, "c":"y", "d":[3,4], "e":"f"}
    ]
    },
    {"k":2,
    "v":[
    {"nk":33, "c":"z", "d":[5,6], "e":"f"}
    ]
    }
    ]}`
var complexNextedEntitySetModifyItem = `{
    "t":[
    {"k":2,
    "v":[
    {"nk":33, "c":"zz", "d":[7,8]}
    ]
    }
    ]}`

var entitySetTestCollections = Collections{
	EntitySets: EntitySets{
		Path("$.t"):      Key("k"),
		Path("$.t[*].v"): Key("nk"),
	},
	Arrays: []Path{}, // No arrays in this test, only sets
}

func TestCreatePatch_AddItemToEntitySet_InEnsureExistsMode_GeneratesAddOperation(t *testing.T) {
	patch, err := CreatePatch([]byte(simpleObjEntitySet), []byte(simpleObjAddEntitySetItem), entitySetTestCollections, nil, PatchStrategyEnsureExists)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "/t/2", change.Path, "they should be equal")
	var expected = map[string]any{"k": float64(3), "v": float64(3)}
	assert.Equal(t, expected, change.Value, "they should be equal")
}

func TestCreatePatch_AddItemToEntitySet_InExactMatchMode_GeneratesAddOperation(t *testing.T) {
	patch, err := CreatePatch([]byte(simpleObjEntitySet), []byte(simpleObjAddEntitySetItem), entitySetTestCollections, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1", change.Path, "they should be equal")
	change = patch[1]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/t/0", change.Path, "they should be equal")
	change = patch[2]
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "/t/0", change.Path, "they should be equal")
	var expected = map[string]any{"k": float64(3), "v": float64(3)}
	assert.Equal(t, expected, change.Value, "they should be equal")
}

func TestCreatePatch_ModifyItemInEntitySet_InEnsureExistsMode_GeneratesReplaceOperation(t *testing.T) {
	patch, err := CreatePatch([]byte(simpleObjEntitySet), []byte(simpleObjModifyEntitySetItem), entitySetTestCollections, nil, PatchStrategyEnsureExists)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "replace", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1/v", change.Path, "they should be equal")
	var expected float64 = 3
	assert.Equal(t, expected, change.Value, "they should be equal")
}

func TestCreatePatch_RemoveItemFromEntitySet_InExactMatchExistsMode_GeneratesRemoveOperation(t *testing.T) {
	patch, err := CreatePatch([]byte(simpleObjEntitySet), []byte(simpleObjEntitySetRemoveItem), entitySetTestCollections, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1", change.Path, "they should be equal")
}

func TestCreatePatch_ModifyItemInEntitySet_InExactMatchMode_GeneratesARemoveAndAReplaceOperation(t *testing.T) {
	patch, err := CreatePatch([]byte(simpleObjEntitySet), []byte(simpleObjModifyEntitySetItem), entitySetTestCollections, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/t/0", change.Path, "they should be equal")
	change = patch[1]
	assert.Equal(t, "replace", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1/v", change.Path, "they should be equal")
	var expected float64 = 3
	assert.Equal(t, expected, change.Value, "they should be equal")
}

func TestCreatePatch_AddDuplicateItemToEntitySet_InEnsureExistsMode_GeneratesNoOperations(t *testing.T) {
	patch, err := CreatePatch([]byte(simpleObjEntitySet), []byte(simpleObjAddDuplicateEntitySetItem), entitySetTestCollections, nil, PatchStrategyEnsureExists)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(patch), "they should be equal")
}

func TestCreatePatch_AddDuplicateItemToEntitySet_InExactMatchMode_GeneratesARemoveOperation(t *testing.T) {
	patch, err := CreatePatch([]byte(simpleObjEntitySet), []byte(simpleObjAddDuplicateEntitySetItem), entitySetTestCollections, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/t/0", change.Path, "they should be equal")
}

func TestCreatePatch_ModifyItemInComplexNestedEntitySet_InEnsureExistsMode_GeneratesReplaceOperation(t *testing.T) {
	patch, err := CreatePatch([]byte(complexNextedEntitySet), []byte(complexNextedEntitySetModifyItem), entitySetTestCollections, nil, PatchStrategyEnsureExists)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "replace", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1/v/0/c", change.Path, "they should be equal")
	assert.Equal(t, "zz", change.Value, "they should be equal")
	change = patch[1]
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1/v/0/d/2", change.Path, "they should be equal")
	assert.Equal(t, float64(7), change.Value, "they should be equal")
	change = patch[2]
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1/v/0/d/3", change.Path, "they should be equal")
	assert.Equal(t, float64(8), change.Value, "they should be equal")
}

func TestCreatePatch_ModifyItemInComplexNestedEntitySet_InExactMatchMode_GeneratesReplaceOperation(t *testing.T) {
	patch, err := CreatePatch([]byte(complexNextedEntitySet), []byte(complexNextedEntitySetModifyItem), entitySetTestCollections, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 6, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/t/0", change.Path, "they should be equal")
	change = patch[1]
	assert.Equal(t, "replace", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1/v/0/c", change.Path, "they should be equal")
	assert.Equal(t, "zz", change.Value, "they should be equal")
	change = patch[2]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1/v/0/d/1", change.Path, "they should be equal")
	change = patch[3]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1/v/0/d/0", change.Path, "they should be equal")
	change = patch[4]
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1/v/0/d/0", change.Path, "they should be equal")
	assert.Equal(t, float64(7), change.Value, "they should be equal")
	change = patch[5]
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "/t/1/v/0/d/1", change.Path, "they should be equal")
	assert.Equal(t, float64(8), change.Value, "they should be equal")
}

func TestCreatePatch_AddMultipleDuplicateAndFailedItemsToEntitySet_InEnsureExistsMode_GeneratesNoOperations(t *testing.T) {
	patch, err := CreatePatch([]byte(simpleObjEntitySet), []byte(simpleObjAddMultipleDuplicateAndFailedItems), entitySetTestCollections, nil, PatchStrategyEnsureExists)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "/t/2", change.Path, "they should be equal")
	var expected = map[string]any{"k": float64(3), "v": float64(3)}
	assert.Equal(t, expected, change.Value, "they should be equal")
	change = patch[1]
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "/t/3", change.Path, "they should be equal")
	var expected2 = map[string]any{"k": float64(4), "v": float64(4)}
	assert.Equal(t, expected2, change.Value, "they should be equal")
}
