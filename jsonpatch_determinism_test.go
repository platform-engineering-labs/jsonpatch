package jsonpatch

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// Patch order is observable when callers certify the exact planned operations.
// Equivalent object encodings must not produce a different certificate.
func TestCreatePatchDeterministicObjectTraversal(t *testing.T) {
	tests := []struct {
		name        string
		before      string
		after       string
		reordered   string
		collections Collections
		want        []JsonPatchOperation
		wantEnsure  []JsonPatchOperation
	}{
		{
			name:      "flat",
			before:    `{"alpha":"old","beta":"old","gamma":"old","name":"old"}`,
			after:     `{"alpha":"one","beta":"two","gamma":"three","name":"new"}`,
			reordered: `{"name":"new","gamma":"three","beta":"two","alpha":"one"}`,
			want: []JsonPatchOperation{
				NewPatch("replace", "/alpha", "one"), NewPatch("replace", "/beta", "two"),
				NewPatch("replace", "/gamma", "three"), NewPatch("replace", "/name", "new"),
			},
		},
		{
			name:      "nested",
			before:    `{"left":{"alpha":0,"beta":0},"right":{"gamma":0,"delta":0}}`,
			after:     `{"left":{"alpha":1,"beta":2},"right":{"gamma":3,"delta":4}}`,
			reordered: `{"right":{"delta":4,"gamma":3},"left":{"beta":2,"alpha":1}}`,
			want: []JsonPatchOperation{
				NewPatch("replace", "/left/alpha", float64(1)), NewPatch("replace", "/left/beta", float64(2)),
				NewPatch("replace", "/right/delta", float64(4)), NewPatch("replace", "/right/gamma", float64(3)),
			},
		},
		{
			name:      "escaped_keys",
			before:    `{"/":"old","~":"old","a/b":{"a~b":"old","a/b":"old"}}`,
			after:     `{"/":"slash","~":"tilde","a/b":{"a~b":"tilde","a/b":"slash"}}`,
			reordered: `{"a/b":{"a/b":"slash","a~b":"tilde"},"~":"tilde","/":"slash"}`,
			want: []JsonPatchOperation{
				NewPatch("replace", "/~1", "slash"),
				NewPatch("replace", "/a~1b/a~1b", "slash"), NewPatch("replace", "/a~1b/a~0b", "tilde"),
				NewPatch("replace", "/~0", "tilde"),
			},
		},
		{
			name:        "co_owned_object_removals",
			before:      `{"labels":{"alpha":"1","beta":"2","gamma":"3","foreign":"keep"}}`,
			after:       `{"labels":{}}`,
			reordered:   `{"labels":{}}`,
			collections: Collections{CoOwned: CoOwned{"$.labels": Drainable{"alpha": {}, "beta": {}, "gamma": {}}}},
			want: []JsonPatchOperation{
				NewPatch("remove", "/labels/alpha", nil), NewPatch("remove", "/labels/beta", nil), NewPatch("remove", "/labels/gamma", nil),
			},
			wantEnsure: []JsonPatchOperation{},
		},
		{
			name:        "omitted_entity_sets",
			before:      `{"alpha":[{"id":"a"}],"beta":[{"id":"b"}],"gamma":[{"id":"c"}]}`,
			after:       `{}`,
			reordered:   `{}`,
			collections: Collections{EntitySets: EntitySets{"$.alpha": "id", "$.beta": "id", "$.gamma": "id"}},
			want: []JsonPatchOperation{
				NewPatch("remove", "/alpha", nil), NewPatch("remove", "/beta", nil), NewPatch("remove", "/gamma", nil),
			},
			wantEnsure: []JsonPatchOperation{},
		},
		{
			name:        "updates_before_removals",
			before:      `{"alpha":[{"id":"a"}],"zeta":"old"}`,
			after:       `{"zeta":"new","beta":"added"}`,
			reordered:   `{"beta":"added","zeta":"new"}`,
			collections: Collections{EntitySets: EntitySets{"$.alpha": "id"}},
			want: []JsonPatchOperation{
				NewPatch("add", "/beta", "added"), NewPatch("replace", "/zeta", "new"), NewPatch("remove", "/alpha", nil),
			},
			wantEnsure: []JsonPatchOperation{NewPatch("add", "/beta", "added"), NewPatch("replace", "/zeta", "new")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, strategy := range []PatchStrategy{PatchStrategyExactMatch, PatchStrategyEnsureExists} {
				t.Run(string(strategy), func(t *testing.T) {
					want := tt.want
					if strategy == PatchStrategyEnsureExists && tt.wantEnsure != nil {
						want = tt.wantEnsure
					}
					wantJSON, err := json.Marshal(want)
					require.NoError(t, err)
					for i := 0; i < 100; i++ {
						after := tt.after
						if i%2 == 1 {
							after = tt.reordered
						}
						next, err := CreatePatch([]byte(tt.before), []byte(after), tt.collections, nil, strategy)
						require.NoError(t, err)
						got, err := json.Marshal(next)
						require.NoError(t, err)
						require.Equal(t, string(wantJSON), string(got), "iteration %d", i)
					}
				})
			}
		})
	}
}

// Sorting finished operations by path would violate the descending removal
// order needed to address array indices, and entity-set updates after removals.
func TestCreatePatchDeterminismPreservesArrayOperationOrder(t *testing.T) {
	for _, tt := range []struct {
		name        string
		before      string
		after       string
		collections Collections
		strategy    PatchStrategy
		want        []JsonPatchOperation
	}{
		{
			name:        "array",
			before:      `{"items":["a","b","c","d"]}`,
			after:       `{"items":["d"]}`,
			collections: Collections{Arrays: []Path{"$.items"}},
			want:        []JsonPatchOperation{NewPatch("remove", "/items/2", nil), NewPatch("remove", "/items/1", nil), NewPatch("remove", "/items/0", nil)},
		},
		{
			name:   "set",
			before: `{"items":["a","b","c","d"]}`,
			after:  `{"items":["d","e"]}`,
			want:   []JsonPatchOperation{NewPatch("remove", "/items/2", nil), NewPatch("remove", "/items/1", nil), NewPatch("remove", "/items/0", nil), NewPatch("add", "/items/1", "e")},
		},
		{
			name:        "entity_set_removals_before_update",
			before:      `{"items":[{"id":"a"},{"id":"b"},{"id":"c","value":"old"}]}`,
			after:       `{"items":[{"id":"c","value":"new"}]}`,
			collections: Collections{EntitySets: EntitySets{"$.items": "id"}},
			want:        []JsonPatchOperation{NewPatch("remove", "/items/1", nil), NewPatch("remove", "/items/0", nil), NewPatch("replace", "/items/0/value", "new")},
		},
		{
			name:        "array_multi_digit_indices",
			before:      `{"items":["a","b","c","d","e","f","g","h","i","j","k","l"]}`,
			after:       `{"items":["a","b","d","e","f","g","h","i","j","l","m"]}`,
			collections: Collections{Arrays: []Path{"$.items"}},
			want:        []JsonPatchOperation{NewPatch("remove", "/items/10", nil), NewPatch("remove", "/items/2", nil), NewPatch("add", "/items/10", "m")},
		},
		{
			name:   "set_multi_digit_indices",
			before: `{"items":["a","b","c","d","e","f","g","h","i","j","k","l"]}`,
			after:  `{"items":["a","b","d","e","f","g","h","i","j","l","m"]}`,
			want:   []JsonPatchOperation{NewPatch("remove", "/items/10", nil), NewPatch("remove", "/items/2", nil), NewPatch("add", "/items/10", "m")},
		},
		{
			name:        "entity_set_multi_digit_indices",
			before:      `{"items":[{"id":"a"},{"id":"b"},{"id":"c"},{"id":"d"},{"id":"e"},{"id":"f"},{"id":"g"},{"id":"h"},{"id":"i"},{"id":"j"},{"id":"k"},{"id":"l","value":"old"}]}`,
			after:       `{"items":[{"id":"a"},{"id":"b"},{"id":"d"},{"id":"e"},{"id":"f"},{"id":"g"},{"id":"h"},{"id":"i"},{"id":"j"},{"id":"l","value":"new"}]}`,
			collections: Collections{EntitySets: EntitySets{"$.items": "id"}},
			want:        []JsonPatchOperation{NewPatch("remove", "/items/10", nil), NewPatch("remove", "/items/2", nil), NewPatch("replace", "/items/9/value", "new")},
		},
		{
			name:        "ensure_exists_array_add_order",
			before:      `{"items":[]}`,
			after:       `{"items":["a","b","c","d","e","f","g","h","i","j","k","l"]}`,
			collections: Collections{Arrays: []Path{"$.items"}},
			strategy:    PatchStrategyEnsureExists,
			want: []JsonPatchOperation{
				NewPatch("add", "/items/0", "a"), NewPatch("add", "/items/1", "b"), NewPatch("add", "/items/2", "c"),
				NewPatch("add", "/items/3", "d"), NewPatch("add", "/items/4", "e"), NewPatch("add", "/items/5", "f"),
				NewPatch("add", "/items/6", "g"), NewPatch("add", "/items/7", "h"), NewPatch("add", "/items/8", "i"),
				NewPatch("add", "/items/9", "j"), NewPatch("add", "/items/10", "k"), NewPatch("add", "/items/11", "l"),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			strategy := tt.strategy
			if strategy == "" {
				strategy = PatchStrategyExactMatch
			}
			got, err := CreatePatch([]byte(tt.before), []byte(tt.after), tt.collections, nil, strategy)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
