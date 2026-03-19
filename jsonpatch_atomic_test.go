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
