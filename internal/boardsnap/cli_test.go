package boardsnap

import (
	"bytes"
	"sort"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	full := append([]string{"--store", testStore(t)}, args...)
	rc := Run(full, &out, &errOut)
	return rc, out.String(), errOut.String()
}

func TestCLISnapshots(t *testing.T) {
	rc, out, _ := runCLI(t, "snapshots")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if out != "2025-03-15\n2025-06-01\n2025-08-31\n" {
		t.Fatalf("out: %q", out)
	}
}

func TestCLILeadersByteExact(t *testing.T) {
	rc, out, _ := runCLI(t, "leaders", "helio-text-v1", "--as-of", "2025-06-01", "--complete-only")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	want := "Snapshot: 2025-06-01\n" +
		"rank  model                          score   tasks\n" +
		"----  -----                          -----   -----\n" +
		"1     omnicrate/omni-embed-v2        0.6880  8/8\n" +
		"2     omnicrate/omni-embed-beta      0.6781  8/8\n" +
		"3     pelora/pl-text-large           0.6619  8/8\n" +
		"4     nuvixa/embro-7b                0.6473  8/8\n" +
		"5     pelora/pl-text-base            0.6238  7/8(partial)\n" +
		"6     zephira-labs/zpl-mini          0.5609  8/8\n"
	if out != want {
		t.Fatalf("out:\n%q\nwant:\n%q", out, want)
	}
}

func TestCLILeadersPartialMarked(t *testing.T) {
	rc, out, _ := runCLI(t, "leaders", "helio-text-v1", "--as-of", "2025-03-15")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.Contains(out, "2     kantrel/kds-retro-base         0.6506  5/8(partial)\n") {
		t.Fatalf("missing partial marker:\n%s", out)
	}
}

func TestCLINoSnapshotExitCode(t *testing.T) {
	rc, _, errOut := runCLI(t, "leaders", "helio-text-v1", "--as-of", "2024-12-31")
	if rc != ExitNoSnapshot {
		t.Fatalf("rc=%d, want %d", rc, ExitNoSnapshot)
	}
	if !strings.Contains(errOut, "no snapshot on or before 2024-12-31") {
		t.Fatalf("err: %q", errOut)
	}
}

func TestCLIUnknownBoard(t *testing.T) {
	rc, _, errOut := runCLI(t, "leaders", "nope-v9", "--as-of", "2025-06-01")
	if rc != ExitNoBoard {
		t.Fatalf("rc=%d, want %d", rc, ExitNoBoard)
	}
	if !strings.Contains(errOut, "unknown board") {
		t.Fatalf("err: %q", errOut)
	}
}

func TestCLIUnknownModel(t *testing.T) {
	// embro-ultra exists in the live snapshot but not at 2025-06-01.
	rc, _, _ := runCLI(t, "model", "nuvixa/embro-ultra", "--as-of", "2025-06-01")
	if rc != ExitNoModel {
		t.Fatalf("rc=%d, want %d", rc, ExitNoModel)
	}
}

func TestCLIModel(t *testing.T) {
	rc, out, _ := runCLI(t, "model", "pelora/pl-text-large", "--as-of", "2025-08-31")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.HasPrefix(out, "Snapshot: 2025-08-31\nmodel: pelora/pl-text-large\nfirst_seen: 2025-08-31\n") {
		t.Fatalf("out head: %q", out[:120])
	}
	if !strings.Contains(out, "  helio-argu-retrieval: 0.6023\n") {
		t.Fatalf("missing task line:\n%s", out)
	}
}

func TestCLITasks(t *testing.T) {
	rc, out, _ := runCLI(t, "tasks", "vireo-code-v1", "--as-of", "2025-03-20")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.HasPrefix(out, "Snapshot: 2025-03-15\n") {
		t.Fatalf("out head: %q", out[:40])
	}
	if !strings.Contains(out, "vireo-api-sts\n") {
		t.Fatalf("missing task:\n%s", out)
	}
}

func TestCLIBoards(t *testing.T) {
	rc, out, _ := runCLI(t, "boards", "--as-of", "2025-08-31")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.Contains(out, "helio-text-v1") || !strings.Contains(out, "vireo-code-v1") {
		t.Fatalf("out:\n%s", out)
	}
}

func TestCLILeadersAsOfOnFreezeDate(t *testing.T) {
	// --as-of equal to a freeze date must answer from that exact snapshot:
	// the 2025-06-01 table contains omnicrate, the 2025-03-15 table does not.
	rc, out, _ := runCLI(t, "leaders", "helio-text-v1", "--as-of", "2025-06-01", "--complete-only")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.HasPrefix(out, "Snapshot: 2025-06-01\n") || !strings.Contains(out, "omnicrate/omni-embed-v2") {
		t.Fatalf("as-of on freeze date answered from the wrong snapshot:\n%s", out)
	}
	rc, out, _ = runCLI(t, "leaders", "helio-text-v1", "--as-of", "2025-03-15", "--complete-only")
	if rc != 0 {
		t.Fatalf("first freeze date: rc=%d", rc)
	}
	if !strings.HasPrefix(out, "Snapshot: 2025-03-15\n") || !strings.Contains(out, "pelora/pl-text-base") {
		t.Fatalf("first freeze date answered from the wrong snapshot:\n%s", out)
	}
}

func TestCLIBoardsOrderAndSnapshotsChronological(t *testing.T) {
	// boards lists ids in ascending order; snapshots lists freeze dates
	// chronologically.
	rc, out, _ := runCLI(t, "snapshots")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if out != "2025-03-15\n2025-06-01\n2025-08-31\n" {
		t.Fatalf("snapshots not chronological:\n%s", out)
	}
	rc, out, _ = runCLI(t, "boards", "--as-of", "2025-08-31")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	helio := strings.Index(out, "helio-text-v1")
	vireo := strings.Index(out, "vireo-code-v1")
	if helio < 0 || vireo < 0 || !(helio < vireo) {
		t.Fatalf("boards not in ascending id order:\n%s", out)
	}
}

func TestCLIModelTasksSortedById(t *testing.T) {
	// The model view lists tasks alphabetically, not by score.
	rc, out, _ := runCLI(t, "model", "nuvixa/embro-7b", "--as-of", "2025-08-31")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var taskLines []string
	for _, l := range lines {
		if strings.HasPrefix(l, "  ") {
			taskLines = append(taskLines, strings.TrimSpace(l))
		}
	}
	sorted := append([]string(nil), taskLines...)
	sort.Strings(sorted)
	for i := range taskLines {
		if taskLines[i] != sorted[i] {
			t.Fatalf("tasks not sorted by id:\n%s", strings.Join(taskLines, "\n"))
		}
	}
}

func TestCLILeadersJSONBooleanComplete(t *testing.T) {
	// JSON `complete` is a boolean, not a quoted string.
	rc, out, _ := runCLI(t, "leaders", "helio-text-v1", "--as-of", "2025-06-01", "--complete-only", "--format", "json")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.Contains(out, `"complete": true`) {
		t.Fatalf("missing boolean complete:\n%s", out)
	}
}
