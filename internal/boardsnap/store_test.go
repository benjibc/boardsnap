package boardsnap

import (
	"os"
	"strings"
	"path/filepath"
	"runtime"
	"testing"
)

func testStore(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "snapshots")
}

func TestResolveLatestWithoutAsOf(t *testing.T) {
	got, err := ResolveSnapshotDate("", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	if got != "2025-08-31" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveBetweenSnapshots(t *testing.T) {
	store := testStore(t)
	for asOf, want := range map[string]string{
		"2025-06-01": "2025-03-15",
		"2025-07-19": "2025-06-01",
		"2025-03-16": "2025-03-15",
	} {
		got, err := ResolveSnapshotDate(asOf, store)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("as-of %s: got %q, want %q", asOf, got, want)
		}
	}
}

func TestResolveBeforeFirstSnapshotFails(t *testing.T) {
	if _, err := ResolveSnapshotDate("2024-12-31", testStore(t)); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveRejectsBadDate(t *testing.T) {
	if _, err := ResolveSnapshotDate("2025-6-1", testStore(t)); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadSnapshotBoards(t *testing.T) {
	snap, err := LoadSnapshot("2025-06-01", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Boards) != 2 {
		t.Fatalf("boards: got %d, want 2", len(snap.Boards))
	}
	helio := snap.Boards["helio-text-v1"]
	if len(helio.Tasks) != 8 {
		t.Fatalf("helio tasks: got %d, want 8", len(helio.Tasks))
	}
	if helio.TaskTypes["helio-legal-sts"] != "sts" {
		t.Fatalf("task type: got %q", helio.TaskTypes["helio-legal-sts"])
	}
}

func TestResolveExactFreezeDateInclusive(t *testing.T) {
	store := testStore(t)
	// An as-of date equal to a freeze date must resolve to that snapshot,
	// including the very first freeze date.
	for asOf, want := range map[string]string{
		"2025-03-16": "2025-03-15",
		"2025-06-01": "2025-03-15",
		"2025-09-01": "2025-08-31",
	} {
		got, err := ResolveSnapshotDate(asOf, store)
		if err != nil {
			t.Fatalf("as-of %s: %v", asOf, err)
		}
		if got != want {
			t.Fatalf("as-of %s: got %q, want %q", asOf, got, want)
		}
	}
}

func TestDefaultStoreWalksUpward(t *testing.T) {
	// A snapshots/ directory above the working directory must be discovered
	// by walking upward from the cwd.
	base := t.TempDir()
	nested := filepath.Join(base, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "snapshots", "2025-01-01"), 0o755); err != nil {
		t.Fatal(err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()
	got, err := DefaultStore()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "snapshots"); got != want {
		t.Fatalf("DefaultStore: got %q, want %q", got, want)
	}
}

func TestLoadSnapshotFiltersFutureFirstSeen(t *testing.T) {
	// Row-level as-of: a score file whose first_seen is later than the freeze
	// date (a retro-dated backfill) must not be ranked in that snapshot. The
	// 2025-06-01 store carries omnicrate/omni-embed-v2 with first_seen
	// 2025-07-15; it exists in the frozen files but must not load.
	snap, err := LoadSnapshot("2025-06-01", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, present := snap.Rows["omnicrate/omni-embed-v2"]; present {
		t.Fatal("retro-dated row (first_seen 2025-07-15) must be excluded from the 2025-06-01 snapshot")
	}
	// pelora/pl-text-base (first_seen 2025-01-10) is legitimately present.
	if _, present := snap.Rows["pelora/pl-text-base"]; !present {
		t.Fatal("pelora/pl-text-base should be present in the 2025-06-01 snapshot")
	}
}

func TestResolveInvalidDateMessageIncludesFormat(t *testing.T) {
	_, err := ResolveSnapshotDate("2025-6-1", testStore(t))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "(want YYYY-MM-DD)") {
		t.Fatalf("error must include the format hint: %v", err)
	}
}
