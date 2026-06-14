package sellproduct

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestOperatorCacheReadWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "SellProductOwnedOperators.json")
	now := time.Date(2026, 6, 14, 1, 2, 3, 0, time.UTC)

	if err := writeOperatorCache(
		path,
		[]string{"Wulfgard", "Ardelia", "Wulfgard", ""},
		[]string{"Wulfgard", "Ardelia", "Avywenna", ""},
		now,
	); err != nil {
		t.Fatalf("writeOperatorCache: %v", err)
	}
	cache, err := readOperatorCache(path)
	if err != nil {
		t.Fatalf("readOperatorCache: %v", err)
	}
	if cache.SchemaVersion != operatorCacheSchemaVersion {
		t.Fatalf("schema version = %d, want %d", cache.SchemaVersion, operatorCacheSchemaVersion)
	}
	if cache.UpdatedAt != "2026-06-14T01:02:03Z" {
		t.Fatalf("updated_at = %q", cache.UpdatedAt)
	}
	want := []string{"Ardelia", "Wulfgard"}
	if !reflect.DeepEqual(cache.Operators, want) {
		t.Fatalf("operators = %#v, want %#v", cache.Operators, want)
	}
	wantScanned := []string{"Ardelia", "Avywenna", "Wulfgard"}
	if !reflect.DeepEqual(cache.ScannedOperators, wantScanned) {
		t.Fatalf("scanned operators = %#v, want %#v", cache.ScannedOperators, wantScanned)
	}
}

func TestOperatorCacheMissingAndEmpty(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	cache, err := readOperatorCache(missing)
	if err != nil {
		t.Fatalf("missing cache should not error: %v", err)
	}
	if len(cache.Operators) != 0 {
		t.Fatalf("missing cache operators = %#v", cache.Operators)
	}

	empty := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(empty, nil, 0644); err != nil {
		t.Fatal(err)
	}
	cache, err = readOperatorCache(empty)
	if err != nil {
		t.Fatalf("empty cache should not error: %v", err)
	}
	if len(cache.Operators) != 0 {
		t.Fatalf("empty cache operators = %#v", cache.Operators)
	}
}

func TestNormalizeOperatorCandidates(t *testing.T) {
	got := normalizeOperatorCandidates([]operatorCandidate{
		{Name: "Beta", Expected: []string{"贝塔"}, Priority: 2},
		{Name: "", Expected: []string{"忽略"}, Priority: 0},
		{Name: "Alpha", Expected: []string{"阿尔法", "阿尔法", ""}, Priority: 1},
		{Name: "Beta", Expected: []string{"重复"}, Priority: 0},
	})
	want := []operatorCandidate{
		{Name: "Alpha", Expected: []string{"阿尔法"}, Priority: 1},
		{Name: "Beta", Expected: []string{"贝塔"}, Priority: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeOperatorCandidates = %#v, want %#v", got, want)
	}
}

func TestFilterOwnedCandidates(t *testing.T) {
	candidates := []operatorCandidate{
		{Name: "Both", Priority: 0},
		{Name: "Money", Priority: 1},
		{Name: "Exp", Priority: 2},
	}
	owned := operatorNameSet([]string{"Exp", "Both"})
	got := filterOwnedCandidates(candidates, owned)
	want := []operatorCandidate{
		{Name: "Both", Priority: 0},
		{Name: "Exp", Priority: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterOwnedCandidates = %#v, want %#v", got, want)
	}
}

func TestOperatorCacheCoversCandidates(t *testing.T) {
	cache := operatorCacheFile{
		ScannedOperators: []string{"Alpha", "Beta"},
	}
	if !operatorCacheCoversCandidates(cache, []operatorCandidate{{Name: "Alpha"}, {Name: "Beta"}}) {
		t.Fatal("cache should cover all candidates")
	}
	if operatorCacheCoversCandidates(cache, []operatorCandidate{{Name: "Alpha"}, {Name: "Gamma"}}) {
		t.Fatal("cache should not cover missing candidates")
	}
	if operatorCacheCoversCandidates(cache, nil) {
		t.Fatal("empty candidates should not be covered")
	}
}

func TestMergeOperatorCacheUpdatesScannedScope(t *testing.T) {
	now := time.Date(2026, 6, 14, 1, 2, 3, 0, time.UTC)
	cache := operatorCacheFile{
		Operators:        []string{"Old", "Keep"},
		ScannedOperators: []string{"Old", "Keep"},
	}
	got := mergeOperatorCache(
		cache,
		[]operatorCandidate{{Name: "Old"}, {Name: "New"}},
		[]string{"New"},
		now,
	)
	if want := []string{"Keep", "New"}; !reflect.DeepEqual(got.Operators, want) {
		t.Fatalf("operators = %#v, want %#v", got.Operators, want)
	}
	if want := []string{"Keep", "New", "Old"}; !reflect.DeepEqual(got.ScannedOperators, want) {
		t.Fatalf("scanned operators = %#v, want %#v", got.ScannedOperators, want)
	}
}
