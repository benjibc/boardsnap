package boardsnap

import (
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
		"2025-06-01": "2025-06-01",
		"2025-07-19": "2025-06-01",
		"2025-03-15": "2025-03-15",
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
		"2025-03-15": "2025-03-15",
		"2025-06-01": "2025-06-01",
		"2025-08-31": "2025-08-31",
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
