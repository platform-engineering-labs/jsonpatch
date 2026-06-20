package jsonpatch

import (
	"testing"
)

func TestAtomicField_DifferentValues_SingleReplace(t *testing.T) {
	a := `{"PolicyDocument": {"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": "s3:GetObject", "Resource": "*"}]}}`
	b := `{"PolicyDocument": {"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": "s3:PutObject", "Resource": "*"}]}}`

	collections := Collections{
		EntitySets: EntitySets{},
		Arrays:     []Path{},
		Atomics:    []Path{"$.PolicyDocument"},
	}

	patch, err := CreatePatch([]byte(a), []byte(b), collections, nil, PatchStrategyExactMatch)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 1 {
		t.Fatalf("expected 1 patch operation, got %d: %v", len(patch), patch)
	}
	if patch[0].Operation != "replace" {
		t.Errorf("expected replace operation, got %s", patch[0].Operation)
	}
	if patch[0].Path != "/PolicyDocument" {
		t.Errorf("expected path /PolicyDocument, got %s", patch[0].Path)
	}
}

func TestAtomicField_EqualValues_NoPatch(t *testing.T) {
	a := `{"PolicyDocument": {"Version": "2012-10-17", "Statement": [{"Effect": "Allow"}]}}`
	b := `{"PolicyDocument": {"Version": "2012-10-17", "Statement": [{"Effect": "Allow"}]}}`

	collections := Collections{
		Atomics: []Path{"$.PolicyDocument"},
	}

	patch, err := CreatePatch([]byte(a), []byte(b), collections, nil, PatchStrategyExactMatch)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 0 {
		t.Fatalf("expected 0 patch operations, got %d: %v", len(patch), patch)
	}
}

func TestAtomicField_NestedArrayDiffers_SingleReplace(t *testing.T) {
	// Even though the nested array has different elements, atomic should produce a single replace
	a := `{"Config": {"Items": [1, 2, 3]}}`
	b := `{"Config": {"Items": [4, 5, 6]}}`

	collections := Collections{
		Atomics: []Path{"$.Config"},
	}

	patch, err := CreatePatch([]byte(a), []byte(b), collections, nil, PatchStrategyExactMatch)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 1 {
		t.Fatalf("expected 1 patch operation, got %d: %v", len(patch), patch)
	}
	if patch[0].Operation != "replace" {
		t.Errorf("expected replace, got %s", patch[0].Operation)
	}
	if patch[0].Path != "/Config" {
		t.Errorf("expected path /Config, got %s", patch[0].Path)
	}
}

func TestAtomicField_NullToValue_Replace(t *testing.T) {
	// null→map type change produces replace (handled by diff before atomic check)
	a := `{"PolicyDocument": null}`
	b := `{"PolicyDocument": {"Version": "2012-10-17"}}`

	collections := Collections{
		Atomics: []Path{"$.PolicyDocument"},
	}

	patch, err := CreatePatch([]byte(a), []byte(b), collections, nil, PatchStrategyExactMatch)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 1 {
		t.Fatalf("expected 1 patch operation, got %d: %v", len(patch), patch)
	}
	// Type change from null to map is a single replace — correct atomic behavior
	if patch[0].Operation != "replace" {
		t.Errorf("expected replace operation, got %s", patch[0].Operation)
	}
	if patch[0].Path != "/PolicyDocument" {
		t.Errorf("expected path /PolicyDocument, got %s", patch[0].Path)
	}
}

func TestAtomicField_NonAtomicUnchanged(t *testing.T) {
	// Non-atomic fields should still recurse as before
	a := `{"PolicyDocument": {"Version": "old"}, "Name": "test"}`
	b := `{"PolicyDocument": {"Version": "new"}, "Name": "test"}`

	collections := Collections{} // no atomics

	patch, err := CreatePatch([]byte(a), []byte(b), collections, nil, PatchStrategyExactMatch)
	if err != nil {
		t.Fatal(err)
	}

	// Should recurse into PolicyDocument and produce a replace on /PolicyDocument/Version
	if len(patch) != 1 {
		t.Fatalf("expected 1 patch operation, got %d: %v", len(patch), patch)
	}
	if patch[0].Path != "/PolicyDocument/Version" {
		t.Errorf("expected recursive path /PolicyDocument/Version, got %s", patch[0].Path)
	}
}

func TestAtomicField_ExtraKeysInActual_SingleReplace(t *testing.T) {
	// Atomic field where actual has extra keys not in desired — should be a single replace with desired value
	a := `{"Policy": {"Version": "2012-10-17", "Id": "extra", "Statement": []}}`
	b := `{"Policy": {"Version": "2012-10-17", "Statement": [{"Effect": "Allow"}]}}`

	collections := Collections{
		Atomics: []Path{"$.Policy"},
	}

	patch, err := CreatePatch([]byte(a), []byte(b), collections, nil, PatchStrategyExactMatch)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 1 {
		t.Fatalf("expected 1 patch operation, got %d: %v", len(patch), patch)
	}
	if patch[0].Operation != "replace" {
		t.Errorf("expected replace, got %s", patch[0].Operation)
	}
	if patch[0].Path != "/Policy" {
		t.Errorf("expected path /Policy, got %s", patch[0].Path)
	}
}

func TestAtomicField_Array_SingleReplace(t *testing.T) {
	// An array field marked Atomic must produce a single whole-array replace,
	// not per-element remove+add. AWS Cloud Control does not reliably apply a
	// remove+add pair against a mutually-exclusive list (e.g. NetworkFirewall
	// FirewallPolicy.StatefulDefaultActions), leaving both old and new values.
	a := `{"L": ["aws:drop_strict"]}`
	b := `{"L": ["aws:drop_established"]}`

	collections := Collections{
		Atomics: []Path{"$.L"},
	}

	patch, err := CreatePatch([]byte(a), []byte(b), collections, nil, PatchStrategyExactMatch)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 1 {
		t.Fatalf("expected 1 patch operation, got %d: %v", len(patch), patch)
	}
	if patch[0].Operation != "replace" {
		t.Errorf("expected replace operation, got %s", patch[0].Operation)
	}
	if patch[0].Path != "/L" {
		t.Errorf("expected path /L, got %s", patch[0].Path)
	}
}

func TestAtomicField_NestedArray_SingleReplace(t *testing.T) {
	// The PLA-37 shape: an array nested inside an object, addressed by a dotted
	// hint key (FirewallPolicy.StatefulDefaultActions -> $.FirewallPolicy.StatefulDefaultActions).
	a := `{"FirewallPolicy": {"StatefulDefaultActions": ["aws:drop_strict"]}}`
	b := `{"FirewallPolicy": {"StatefulDefaultActions": ["aws:drop_established"]}}`

	collections := Collections{
		Atomics: []Path{"$.FirewallPolicy.StatefulDefaultActions"},
	}

	patch, err := CreatePatch([]byte(a), []byte(b), collections, nil, PatchStrategyExactMatch)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 1 {
		t.Fatalf("expected 1 patch operation, got %d: %v", len(patch), patch)
	}
	if patch[0].Operation != "replace" {
		t.Errorf("expected replace operation, got %s", patch[0].Operation)
	}
	if patch[0].Path != "/FirewallPolicy/StatefulDefaultActions" {
		t.Errorf("expected path /FirewallPolicy/StatefulDefaultActions, got %s", patch[0].Path)
	}
	if v, ok := patch[0].Value.([]any); !ok || len(v) != 1 || v[0] != "aws:drop_established" {
		t.Errorf("expected value [aws:drop_established], got %v", patch[0].Value)
	}
}

func TestAtomicField_Array_EqualContent_NoPatch(t *testing.T) {
	// Atomic array equal in content (set semantics) must produce no patch — even
	// when element order differs — to avoid a perpetual drift loop when the
	// provider returns the same values in a different order.
	a := `{"L": ["a", "b"]}`
	b := `{"L": ["b", "a"]}`

	collections := Collections{
		Atomics: []Path{"$.L"},
	}

	patch, err := CreatePatch([]byte(a), []byte(b), collections, nil, PatchStrategyExactMatch)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 0 {
		t.Fatalf("expected 0 patch operations, got %d: %v", len(patch), patch)
	}
}
