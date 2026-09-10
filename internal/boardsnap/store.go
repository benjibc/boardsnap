// Package boardsnap implements snapshot discovery, as-of resolution, ranking,
// and deterministic output for the boardsnap CLI.
package boardsnap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// DateRe matches snapshot directory names and --as-of values.
var DateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// StoreError is any store-level failure (bad layout, missing snapshot).
type StoreError struct{ msg string }

func (e *StoreError) Error() string { return e.msg }

// Board is one standings board in a snapshot.
type Board struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Tasks      []string          `json:"tasks"`
	TaskTypes  map[string]string `json:"task_types"`
	Aggregates []string          `json:"aggregates"`
}

// ModelRow is one model's scores inside a snapshot.
type ModelRow struct {
	Model     string
	FirstSeen string
	Scores    map[string]float64 // task id -> main_score
}

// Snapshot is a frozen standings table at one freeze date.
type Snapshot struct {
	Date   string
	Boards map[string]Board
	Rows   map[string]ModelRow
}

type boardsDoc struct {
	FreezeDate string  `json:"freeze_date"`
	Boards     []Board `json:"boards"`
}

type scoreDoc struct {
	Model     string `json:"model"`
	FirstSeen string `json:"first_seen"`
	Results   map[string]struct {
		MainScore *float64 `json:"main_score"`
		Split     string   `json:"split"`
	} `json:"results"`
}

// DefaultStore locates the snapshots/ directory shipped beside the source.
func DefaultStore() (string, error) {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "snapshots")
		if st, statErr := os.Stat(candidate); statErr == nil && st.IsDir() {
			return candidate, nil
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", &StoreError{"cannot locate working directory"}
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "snapshots")
		if st, statErr := os.Stat(candidate); statErr == nil && st.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", &StoreError{"snapshot store not found (no snapshots/ directory at or above the working directory)"}
		}
	}
}

// ListSnapshotDates returns the sorted freeze dates available in the store.
func ListSnapshotDates(store string) ([]string, error) {
	entries, err := os.ReadDir(store)
	if err != nil {
		return nil, &StoreError{fmt.Sprintf("snapshot store not found: %s", store)}
	}
	var dates []string
	for _, e := range entries {
		if e.IsDir() && DateRe.MatchString(e.Name()) {
			dates = append(dates, e.Name())
		}
	}
	sort.Strings(dates)
	return dates, nil
}

// ResolveSnapshotDate maps an as-of date ("" means latest) to a freeze date.
func ResolveSnapshotDate(asOf, store string) (string, error) {
	dates, err := ListSnapshotDates(store)
	if err != nil {
		return "", err
	}
	if len(dates) == 0 {
		return "", &StoreError{"snapshot store is empty"}
	}
	if asOf == "" {
		return dates[len(dates)-1], nil
	}
	if !DateRe.MatchString(asOf) {
		return "", &StoreError{fmt.Sprintf("invalid as-of date: %q", asOf)}
	}
	resolved := ""
	for _, d := range dates {
		if d <= asOf {
			resolved = d
		}
	}
	if resolved == "" {
		return "", &StoreError{fmt.Sprintf("no snapshot on or before %s", asOf)}
	}
	return resolved, nil
}

// LoadSnapshot loads the frozen snapshot at one freeze date.
func LoadSnapshot(date, store string) (Snapshot, error) {
	snap := Snapshot{Date: date, Boards: map[string]Board{}, Rows: map[string]ModelRow{}}
	dir := filepath.Join(store, date)
	raw, err := os.ReadFile(filepath.Join(dir, "boards.json"))
	if err != nil {
		return snap, &StoreError{fmt.Sprintf("snapshot %s is missing boards.json", date)}
	}
	var doc boardsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return snap, &StoreError{fmt.Sprintf("snapshot %s has malformed boards.json: %v", date, err)}
	}
	for _, b := range doc.Boards {
		if len(b.Aggregates) == 0 {
			b.Aggregates = []string{"mean-task"}
		}
		snap.Boards[b.ID] = b
	}
	scoresDir := filepath.Join(dir, "scores")
	entries, err := os.ReadDir(scoresDir)
	if err != nil {
		return snap, nil // no scores directory: empty but valid
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(scoresDir, e.Name()))
		if err != nil {
			return snap, &StoreError{fmt.Sprintf("snapshot %s: %v", date, err)}
		}
		var sd scoreDoc
		if err := json.Unmarshal(payload, &sd); err != nil {
			return snap, &StoreError{fmt.Sprintf("snapshot %s: malformed %s: %v", date, e.Name(), err)}
		}
		row := ModelRow{Model: sd.Model, FirstSeen: sd.FirstSeen, Scores: map[string]float64{}}
		if row.FirstSeen == "" {
			row.FirstSeen = date
		}
		_ = row
		for task, res := range sd.Results {
			if res.MainScore != nil {
				row.Scores[task] = *res.MainScore
			}
		}
		snap.Rows[row.Model] = row
	}
	return snap, nil
}

// LoadAsOf resolves asOf and loads the corresponding frozen snapshot.
func LoadAsOf(asOf, store string) (Snapshot, error) {
	date, err := ResolveSnapshotDate(asOf, store)
	if err != nil {
		return Snapshot{}, err
	}
	return LoadSnapshot(date, store)
}
