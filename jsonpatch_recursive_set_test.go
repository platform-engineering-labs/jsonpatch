package jsonpatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMatchesValue_RecursiveSetEquality covers the core invariant of the
// default list-as-set semantics: two lists must compare equal whenever
// their elements form the same multiset, *including* when individual
// elements contain nested collections whose order happens to differ.
//
// Prior to the recursive fix, matchesValue serialised each element via
// json.Marshal and compared the multiset of resulting byte strings.
// json.Marshal sorts map keys (good) but preserves array order (bad for
// us), so a nested list-ordering difference propagated out as a false
// inequality on the outer element comparison.
func TestMatchesValue_NestedArrayOrderingInsideElement(t *testing.T) {
	a := []any{
		map[string]any{
			"Name":    "grafana",
			"Ports":   []any{float64(3000), float64(4318)},
			"Env":     []any{map[string]any{"Name": "X"}, map[string]any{"Name": "Y"}},
		},
		map[string]any{"Name": "mimir", "Ports": []any{float64(9009)}},
	}
	b := []any{
		map[string]any{"Name": "mimir", "Ports": []any{float64(9009)}},
		map[string]any{
			"Name":    "grafana",
			"Ports":   []any{float64(4318), float64(3000)},
			"Env":     []any{map[string]any{"Name": "Y"}, map[string]any{"Name": "X"}},
		},
	}

	assert.True(t, matchesValue(a, b, true),
		"lists should compare equal: same multiset of elements, nested collections only differ by order")
}

// TestMatchesValue_NestedArrayContentDifferenceStillDetected guards the
// negative direction: if a nested list in an element has a real content
// difference (not just reordering), matchesValue must return false.
func TestMatchesValue_NestedArrayContentDifferenceStillDetected(t *testing.T) {
	a := []any{
		map[string]any{"Name": "grafana", "Ports": []any{float64(3000), float64(4318)}},
	}
	b := []any{
		map[string]any{"Name": "grafana", "Ports": []any{float64(3000), float64(9999)}},
	}

	assert.False(t, matchesValue(a, b, true),
		"different nested content must still be detected as inequality")
}

// TestMatchesValue_NestedMapValueDifferenceStillDetected guards the
// negative direction for nested maps as well.
func TestMatchesValue_NestedMapValueDifferenceStillDetected(t *testing.T) {
	a := []any{
		map[string]any{"Name": "grafana", "Meta": map[string]any{"Role": "admin"}},
	}
	b := []any{
		map[string]any{"Name": "grafana", "Meta": map[string]any{"Role": "viewer"}},
	}

	assert.False(t, matchesValue(a, b, true),
		"different nested map content must still be detected as inequality")
}

// TestMatchesValue_DifferentLengthsAreUnequal keeps the length short-circuit
// working.
func TestMatchesValue_DifferentLengthsAreUnequal(t *testing.T) {
	a := []any{map[string]any{"Name": "grafana"}}
	b := []any{map[string]any{"Name": "grafana"}, map[string]any{"Name": "mimir"}}

	assert.False(t, matchesValue(a, b, true),
		"lists of different lengths are not equal")
}

// TestMatchesValue_DuplicatesAreMultisetSensitive ensures duplicate
// elements are counted as a multiset, not a set — [A, A] != [A, B].
func TestMatchesValue_DuplicatesAreMultisetSensitive(t *testing.T) {
	a := []any{
		map[string]any{"Name": "grafana"},
		map[string]any{"Name": "grafana"},
	}
	b := []any{
		map[string]any{"Name": "grafana"},
		map[string]any{"Name": "mimir"},
	}

	assert.False(t, matchesValue(a, b, true),
		"multiset semantics: duplicated elements should not match against distinct elements")
}

// TestMatchesValue_DeepNestingIsSymmetric ensures matching is symmetric
// — matchesValue(a, b) == matchesValue(b, a) — even across arbitrary depth.
func TestMatchesValue_DeepNestingIsSymmetric(t *testing.T) {
	a := []any{
		map[string]any{
			"Outer": []any{
				map[string]any{
					"Inner": []any{float64(1), float64(2), float64(3)},
				},
			},
		},
	}
	b := []any{
		map[string]any{
			"Outer": []any{
				map[string]any{
					"Inner": []any{float64(3), float64(2), float64(1)},
				},
			},
		},
	}

	assert.True(t, matchesValue(a, b, true), "a vs b")
	assert.True(t, matchesValue(b, a, true), "b vs a — symmetric")
}
