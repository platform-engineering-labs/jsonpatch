package jsonpatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// This test reproduces the bug reported by a user where adding FormaeResourceLabel
// and FormaeStackLabel tags to an existing resource with a Name tag produces
// incorrect array indices in the patch document.
//
// Scenario:
// - Existing resource has Tags: [{"Key": "Name", "Value": "ecg-core"}]
// - New resource has Tags: [{"Key": "Name", ...}, {"Key": "FormaeResourceLabel", ...}, {"Key": "FormaeStackLabel", ...}]
// - Expected: add operations at /Tags/1 and /Tags/2
// - Actual (bug): add operations at /Tags/2 and /Tags/3
//
// The bug occurs because processSet passes the target array index to applyOp,
// and the calling code adds len(source) as an offset, resulting in indices
// that are off by the number of elements that are retained from the source.

var (
	tagsSourceOneItem = `{
		"Tags": [{"Key": "Name", "Value": "ecg-core"}],
		"CidrBlock": "10.192.0.0/16"
	}`

	tagsTargetThreeItems = `{
		"Tags": [
			{"Key": "Name", "Value": "ecg-core"},
			{"Key": "FormaeResourceLabel", "Value": "ecg-core-1"},
			{"Key": "FormaeStackLabel", "Value": "network-stack"}
		],
		"CidrBlock": "10.192.0.0/16"
	}`

	// No collections defined - Tags are treated as a set (default behavior)
	tagsTestCollections = Collections{
		EntitySets: EntitySets{},
		Arrays:     []Path{},
	}
)

func TestCreatePatch_AddTagsWhileRetainingExisting_InEnsureExistsMode_GeneratesCorrectIndices(t *testing.T) {
	// This test reproduces the exact scenario from the user's bug report:
	// - Source has 1 tag (Name)
	// - Target has 3 tags (Name, FormaeResourceLabel, FormaeStackLabel)
	// - Name tag is retained, two new tags are added
	// - Expected patch: add at /Tags/1 and /Tags/2
	// - Bug produces: add at /Tags/2 and /Tags/3

	patch, err := CreatePatch(
		[]byte(tagsSourceOneItem),
		[]byte(tagsTargetThreeItems),
		tagsTestCollections,
		nil,
		PatchStrategyEnsureExists,
	)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(patch), "Expected 2 add operations for the two new tags")

	// First add operation should be at /Tags/1
	change := patch[0]
	assert.Equal(t, "add", change.Operation)
	assert.Equal(t, "/Tags/1", change.Path, "First new tag should be added at index 1")
	expectedValue := map[string]any{"Key": "FormaeResourceLabel", "Value": "ecg-core-1"}
	assert.Equal(t, expectedValue, change.Value)

	// Second add operation should be at /Tags/2
	change = patch[1]
	assert.Equal(t, "add", change.Operation)
	assert.Equal(t, "/Tags/2", change.Path, "Second new tag should be added at index 2")
	expectedValue = map[string]any{"Key": "FormaeStackLabel", "Value": "network-stack"}
	assert.Equal(t, expectedValue, change.Value)
}

func TestCreatePatch_AddMultipleTagsWhileRetainingMultiple_InEnsureExistsMode_GeneratesCorrectIndices(t *testing.T) {
	// Similar test with 3 existing tags and 2 new tags
	// This tests the pattern mentioned in the handoff: "indices are off by exactly
	// the number of existing tags"

	source := `{
		"Tags": [
			{"Key": "Name", "Value": "my-resource"},
			{"Key": "Environment", "Value": "prod"},
			{"Key": "Team", "Value": "platform"}
		]
	}`

	target := `{
		"Tags": [
			{"Key": "Name", "Value": "my-resource"},
			{"Key": "Environment", "Value": "prod"},
			{"Key": "Team", "Value": "platform"},
			{"Key": "FormaeResourceLabel", "Value": "my-label"},
			{"Key": "FormaeStackLabel", "Value": "my-stack"}
		]
	}`

	patch, err := CreatePatch(
		[]byte(source),
		[]byte(target),
		tagsTestCollections,
		nil,
		PatchStrategyEnsureExists,
	)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(patch), "Expected 2 add operations for the two new tags")

	// First add operation should be at /Tags/3
	change := patch[0]
	assert.Equal(t, "add", change.Operation)
	assert.Equal(t, "/Tags/3", change.Path, "First new tag should be added at index 3")

	// Second add operation should be at /Tags/4
	change = patch[1]
	assert.Equal(t, "add", change.Operation)
	assert.Equal(t, "/Tags/4", change.Path, "Second new tag should be added at index 4")
}
