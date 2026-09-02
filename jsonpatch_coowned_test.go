package jsonpatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func coOwnedCollections(path string, ids ...string) Collections {
	d := Drainable{}
	for _, id := range ids {
		d[id] = struct{}{}
	}
	return Collections{CoOwned: CoOwned{Path(path): d}}
}

// A surplus member of a co-owned set listing is tolerated unless drainable.
func TestCoOwnedSetToleratesSurplusMember(t *testing.T) {
	a := []byte(`{"members":["mine","theirs"]}`)
	b := []byte(`{"members":["mine"]}`)
	ops, err := CreatePatch(a, b, coOwnedCollections("$.members"), nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ops))
}

// A drainable member is removed, and its index addresses the ORIGINAL array.
func TestCoOwnedSetDrainsByOriginalIndex(t *testing.T) {
	a := []byte(`{"members":["theirs","mine"]}`)
	b := []byte(`{"members":[]}`)
	ops, err := CreatePatch(a, b, coOwnedCollections("$.members", `"mine"`), nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ops))
	assert.Equal(t, "remove", ops[0].Operation)
	assert.Equal(t, "/members/1", ops[0].Path)
}

// A non-co-owned listing behaves exactly as before.
func TestNonCoOwnedSetUnchanged(t *testing.T) {
	a := []byte(`{"members":["x","y"]}`)
	b := []byte(`{"members":["x"]}`)
	ops, err := CreatePatch(a, b, Collections{}, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ops))
	assert.Equal(t, "remove", ops[0].Operation)
}

func TestCoOwnedEntitySetDrainsOnlyDrainableKeys(t *testing.T) {
	a := []byte(`{"attrs":[{"Key":"theirs","Value":"1"},{"Key":"mine","Value":"2"}]}`)
	b := []byte(`{"attrs":[]}`)
	c := Collections{
		EntitySets: EntitySets{Path("$.attrs"): Key("Key")},
		CoOwned:    CoOwned{Path("$.attrs"): Drainable{`"mine"`: {}}},
	}
	ops, err := CreatePatch(a, b, c, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ops))
	assert.Equal(t, "remove", ops[0].Operation)
	assert.Equal(t, "/attrs/1", ops[0].Path)
}

func TestCoOwnedEntitySetStillUpdatesMatchedElements(t *testing.T) {
	a := []byte(`{"attrs":[{"Key":"theirs","Value":"1"},{"Key":"mine","Value":"old"}]}`)
	b := []byte(`{"attrs":[{"Key":"mine","Value":"new"}]}`)
	c := Collections{
		EntitySets: EntitySets{Path("$.attrs"): Key("Key")},
		CoOwned:    CoOwned{Path("$.attrs"): Drainable{}},
	}
	ops, err := CreatePatch(a, b, c, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ops))
	assert.Equal(t, "replace", ops[0].Operation)
}

func TestCoOwnedObjectDrainsOnlyDrainableKeys(t *testing.T) {
	a := []byte(`{"labels":{"mine":"1","theirs":"2"}}`)
	b := []byte(`{"labels":{}}`)
	c := Collections{CoOwned: CoOwned{Path("$.labels"): Drainable{"mine": {}}}}
	ops, err := CreatePatch(a, b, c, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ops))
	assert.Equal(t, "remove", ops[0].Operation)
	assert.Equal(t, "/labels/mine", ops[0].Path)
}

func TestPlainObjectKeysStillNeverRemoved(t *testing.T) {
	a := []byte(`{"labels":{"x":"1"}}`)
	b := []byte(`{"labels":{}}`)
	ops, err := CreatePatch(a, b, Collections{}, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ops))
}

func TestCoOwnedObjectPatchModeNeverDrains(t *testing.T) {
	a := []byte(`{"labels":{"mine":"1","theirs":"2"}}`)
	b := []byte(`{"labels":{}}`)
	c := Collections{CoOwned: CoOwned{Path("$.labels"): Drainable{"mine": {}}}}
	ops, err := CreatePatch(a, b, c, nil, PatchStrategyEnsureExists)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ops))
}

// An omitted co-owned entity-set field is whole-field tolerated (no ops),
// where a plain entity-set field absent from desired is removed whole today.
func TestOmittedCoOwnedEntitySetFieldTolerated(t *testing.T) {
	a := []byte(`{"attrs":[{"Key":"theirs","Value":"1"}]}`)
	b := []byte(`{}`)
	c := Collections{
		EntitySets: EntitySets{Path("$.attrs"): Key("Key")},
		CoOwned:    CoOwned{Path("$.attrs"): Drainable{`"anything"`: {}}},
	}
	ops, err := CreatePatch(a, b, c, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ops))
}

// A drainable key containing "/" is escaped per RFC6901 in the op path,
// matching makePath's own escaping behavior.
func TestCoOwnedObjectDrainEscapesSlashInKey(t *testing.T) {
	key := "app.kubernetes.io/name"
	a := []byte(`{"labels":{"` + key + `":"x"}}`)
	b := []byte(`{"labels":{}}`)
	c := Collections{CoOwned: CoOwned{Path("$.labels"): Drainable{key: {}}}}
	ops, err := CreatePatch(a, b, c, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ops))
	assert.Equal(t, "remove", ops[0].Operation)
	assert.Equal(t, makePath("/labels", key), ops[0].Path)
}

// A matched member's update path must address the element's position AFTER
// the emitted removals apply, not its position in the original document: a
// remove at a lower original index shifts every element above it down by
// one.
func TestEntitySetUpdatePathAccountsForEmittedRemoval(t *testing.T) {
	a := []byte(`{"attrs":[{"Key":"mine"},{"Key":"theirs","Value":"old"}]}`)
	b := []byte(`{"attrs":[{"Key":"theirs","Value":"new"}]}`)
	c := Collections{
		EntitySets: EntitySets{Path("$.attrs"): Key("Key")},
		CoOwned:    CoOwned{Path("$.attrs"): Drainable{`"mine"`: {}}},
	}
	ops, err := CreatePatch(a, b, c, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ops))
	assert.Equal(t, "remove", ops[0].Operation)
	assert.Equal(t, "/attrs/0", ops[0].Path)
	assert.Equal(t, "replace", ops[1].Operation)
	assert.Equal(t, "/attrs/0/Value", ops[1].Path)
	assert.Equal(t, "new", ops[1].Value)
}

// Two removals below the matched member and one above it: the update path
// must be shifted only by the removals at lower original indices.
func TestEntitySetUpdatePathAccountsForMultipleEmittedRemovals(t *testing.T) {
	a := []byte(`{"attrs":[{"Key":"a"},{"Key":"b"},{"Key":"target","Value":"old"},{"Key":"c"}]}`)
	b := []byte(`{"attrs":[{"Key":"target","Value":"new"}]}`)
	c := Collections{
		EntitySets: EntitySets{Path("$.attrs"): Key("Key")},
		CoOwned:    CoOwned{Path("$.attrs"): Drainable{`"a"`: {}, `"b"`: {}, `"c"`: {}}},
	}
	ops, err := CreatePatch(a, b, c, nil, PatchStrategyExactMatch)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(ops))
	assert.Equal(t, "remove", ops[0].Operation)
	assert.Equal(t, "/attrs/3", ops[0].Path)
	assert.Equal(t, "remove", ops[1].Operation)
	assert.Equal(t, "/attrs/1", ops[1].Path)
	assert.Equal(t, "remove", ops[2].Operation)
	assert.Equal(t, "/attrs/0", ops[2].Path)
	assert.Equal(t, "replace", ops[3].Operation)
	assert.Equal(t, "/attrs/0/Value", ops[3].Path)
	assert.Equal(t, "new", ops[3].Value)
}

// Under EnsureExists no removals are ever computed, so an update path must
// stay addressed at the member's original, unshifted index.
func TestEntitySetUpdatePathUnaffectedUnderEnsureExists(t *testing.T) {
	a := []byte(`{"attrs":[{"Key":"mine"},{"Key":"theirs","Value":"old"}]}`)
	b := []byte(`{"attrs":[{"Key":"theirs","Value":"new"}]}`)
	c := Collections{
		EntitySets: EntitySets{Path("$.attrs"): Key("Key")},
		CoOwned:    CoOwned{Path("$.attrs"): Drainable{`"mine"`: {}}},
	}
	ops, err := CreatePatch(a, b, c, nil, PatchStrategyEnsureExists)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ops))
	assert.Equal(t, "replace", ops[0].Operation)
	assert.Equal(t, "/attrs/1/Value", ops[0].Path)
}
