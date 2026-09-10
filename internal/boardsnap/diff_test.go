package boardsnap

import (
	"strings"
	"testing"
)

func TestCLILeadersCSV(t *testing.T) {
	rc, out, _ := runCLI(t, "leaders", "helio-text-v1", "--as-of", "2025-06-01", "--complete-only", "--format", "csv")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	want := "Snapshot: 2025-06-01\n" +
		"RANK,MODEL,SCORE,COVERED,TOTAL,COMPLETE\n" +
		"1,omnicrate/omni-embed-v2,0.6880,8,8,true\n" +
		"2,omnicrate/omni-embed-beta,0.6781,8,8,true\n" +
		"3,pelora/pl-text-large,0.6619,8,8,true\n" +
		"4,nuvixa/embro-7b,0.6473,8,8,true\n" +
		"5,pelora/pl-text-base,0.6238,7,8,false\n" +
		"6,zephira-labs/zpl-mini,0.5609,8,8,true\n"
	if out != want {
		t.Fatalf("out:\n%q\nwant:\n%q", out, want)
	}
}

func TestCLILeadersJSON(t *testing.T) {
	rc, out, _ := runCLI(t, "leaders", "vireo-code-v1", "--as-of", "2025-08-31", "--complete-only", "--format", "json")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.HasPrefix(out, "Snapshot: 2025-08-31\n[\n") || !strings.HasSuffix(out, "]\n") {
		t.Fatalf("bad json frame: %q", out[:60])
	}
	if !strings.Contains(out, `"model": "omnicrate/omni-forge-11b"`) {
		t.Fatalf("missing leader:\n%s", out)
	}
}

func TestCLILeadersBadFormat(t *testing.T) {
	rc, _, errOut := runCLI(t, "leaders", "helio-text-v1", "--format", "yaml")
	if rc != ExitUsage {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.Contains(errOut, "unknown format") {
		t.Fatalf("err: %q", errOut)
	}
}

func TestCLIDiff(t *testing.T) {
	rc, out, _ := runCLI(t, "diff", "helio-text-v1", "--from", "2025-06-01", "--complete-only")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	want := "from: 2025-08-31\n" +
		"to: 2025-06-01\n" +
		"model                          from_rank  to_rank  from_score  to_score\n" +
		"-----                          ---------  -------  ----------  --------\n" +
		"nuvixa/embro-ultra             -          1        -           0.7224\n" +
		"omnicrate/omni-embed-beta      2          2        0.6781      0.6781\n" +
		"pelora/pl-text-large           3          3        0.6619      0.6624\n" +
		"nuvixa/embro-7b                4          4        0.6473      0.6473\n" +
		"pelora/pl-text-base            5          5        0.6238      0.6238\n" +
		"zephira-labs/zpl-mini          6          6        0.5609      0.5609\n" +
		"omnicrate/omni-embed-v2        1          -        0.6880      -\n"
	if out != want {
		t.Fatalf("out:\n%q\nwant:\n%q", out, want)
	}
}

func TestCLIDiffUnknownBoard(t *testing.T) {
	rc, _, errOut := runCLI(t, "diff", "nope-v9")
	if rc != ExitNoBoard {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.Contains(errOut, "unknown board") {
		t.Fatalf("err: %q", errOut)
	}
}

func TestDiffLeaderboardsRemoval(t *testing.T) {
	// A model present only in the earlier snapshot sorts after to-present rows.
	from := []LeaderRow{{Rank: 1, Model: "a/x", Score: 0.5}, {Rank: 2, Model: "a/y", Score: 0.4}}
	to := []LeaderRow{{Rank: 1, Model: "a/y", Score: 0.45}}
	rows := DiffLeaderboards(from, to)
	if len(rows) != 2 {
		t.Fatalf("rows: %d", len(rows))
	}
	if rows[0].Model != "a/y" || rows[0].FromRank != 2 || rows[0].ToRank != 1 {
		t.Fatalf("row0: %+v", rows[0])
	}
	if rows[1].Model != "a/x" || rows[1].ToRank != 0 || rows[1].To != nil {
		t.Fatalf("row1: %+v", rows[1])
	}
}
