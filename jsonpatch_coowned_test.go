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
