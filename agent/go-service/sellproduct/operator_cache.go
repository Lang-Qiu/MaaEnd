package sellproduct

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	operatorCacheFileName      = "SellProductOwnedOperators.json"
	operatorCacheSchemaVersion = 2
)

var resolveOperatorCachePathFunc = defaultOperatorCachePath

type operatorCacheFile struct {
	SchemaVersion    int      `json:"schema_version"`
	UpdatedAt        string   `json:"updated_at"`
	Operators        []string `json:"operators"`
	ScannedOperators []string `json:"scanned_operators,omitempty"`
}

func defaultOperatorCachePath() string {
	return filepath.Join("debug", "record", operatorCacheFileName)
}

func readOperatorCache(path string) (operatorCacheFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return operatorCacheFile{}, nil
		}
		return operatorCacheFile{}, fmt.Errorf("read operator cache: %w", err)
	}
	if len(raw) == 0 {
		return operatorCacheFile{}, nil
	}

	var cache operatorCacheFile
	if err := json.Unmarshal(raw, &cache); err != nil {
		return operatorCacheFile{}, fmt.Errorf("parse operator cache: %w", err)
	}
	cache.Operators = uniqueNonEmptyStrings(cache.Operators)
	sort.Strings(cache.Operators)
	cache.ScannedOperators = uniqueNonEmptyStrings(cache.ScannedOperators)
	sort.Strings(cache.ScannedOperators)
	return cache, nil
}

func writeOperatorCache(path string, operators []string, scannedOperators []string, now time.Time) error {
	operators = uniqueNonEmptyStrings(operators)
	sort.Strings(operators)
	scannedOperators = uniqueNonEmptyStrings(scannedOperators)
	sort.Strings(scannedOperators)

	cache := operatorCacheFile{
		SchemaVersion:    operatorCacheSchemaVersion,
		UpdatedAt:        now.UTC().Format(time.RFC3339),
		Operators:        operators,
		ScannedOperators: scannedOperators,
	}
	return writeOperatorCacheFile(path, cache)
}

func writeOperatorCacheFile(path string, cache operatorCacheFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create operator cache dir: %w", err)
	}
	raw, err := json.MarshalIndent(cache, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal operator cache: %w", err)
	}
	raw = append(raw, '\n')
	if err := writeOperatorCacheAtomic(path, raw, 0644); err != nil {
		return fmt.Errorf("write operator cache: %w", err)
	}
	return nil
}

func mergeOperatorCache(cache operatorCacheFile, scanCandidates []operatorCandidate, owned []string, now time.Time) operatorCacheFile {
	operatorSet := operatorNameSet(cache.Operators)
	scannedSet := operatorNameSet(cache.ScannedOperators)
	scanSet := operatorCandidateNameSet(scanCandidates)

	for name := range scanSet {
		delete(operatorSet, name)
		scannedSet[name] = struct{}{}
	}
	for _, name := range owned {
		if _, ok := scanSet[name]; ok {
			operatorSet[name] = struct{}{}
		}
	}

	return operatorCacheFile{
		SchemaVersion:    operatorCacheSchemaVersion,
		UpdatedAt:        now.UTC().Format(time.RFC3339),
		Operators:        sortedSetValues(operatorSet),
		ScannedOperators: sortedSetValues(scannedSet),
	}
}

func operatorCacheCoversCandidates(cache operatorCacheFile, candidates []operatorCandidate) bool {
	if len(candidates) == 0 {
		return false
	}
	scanned := operatorNameSet(cache.ScannedOperators)
	if len(scanned) == 0 {
		return false
	}
	for _, candidate := range candidates {
		if _, ok := scanned[candidate.Name]; !ok {
			return false
		}
	}
	return true
}

func writeOperatorCacheAtomic(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func operatorNameSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return set
}

func operatorCandidateNameSet(candidates []operatorCandidate) map[string]struct{} {
	set := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate.Name == "" {
			continue
		}
		set[candidate.Name] = struct{}{}
	}
	return set
}

func sortedSetValues(set map[string]struct{}) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		if value == "" {
			continue
		}
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}
