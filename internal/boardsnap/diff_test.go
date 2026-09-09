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
	want := "snapshot: 2025-03-15\n" +
		"rank,model,score,covered,total,complete\n" +
		"1,pelora/pl-text-large,0.6403,8,8,true\n" +
		"2,nuvixa/embro-7b,0.6290,8,8,true\n" +
		"3,zephira-labs/zpl-mini,0.5396,8,8,true\n"
	if out != want {
		t.Fatalf("out:\n%q\nwant:\n%q", out, want)
	}
}

func TestCLILeadersJSON(t *testing.T) {
	rc, out, _ := runCLI(t, "leaders", "vireo-code-v1", "--as-of", "2025-08-31", "--complete-only", "--format", "json")
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.HasPrefix(out, "snapshot: 2025-06-01\n[\n") || !strings.HasSuffix(out, "]\n") {
		t.Fatalf("bad json frame: %q", out[:60])
	}
	if strings.Contains(out, "omni-forge-11b") {
		t.Fatalf("unexpected model:\n%s", out)
	}
	if !strings.Contains(out, `"model": "pelora/pl-code-mid", "score": 0.5267`) {
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
	want := "from: 2025-03-15\n" +
		"to: 2025-06-01\n" +
		"model                          from_rank  to_rank  from_score  to_score\n" +
		"-----                          ---------  -------  ----------  --------\n" +
		"omnicrate/omni-embed-beta      -          1        -           0.6572\n" +
		"pelora/pl-text-large           1          2        0.6403      0.6403\n" +
		"nuvixa/embro-7b                2          3        0.6290      0.6290\n" +
		"zephira-labs/zpl-mini          3          4        0.5396      0.5396\n"
	if out != want {
		t.Fatalf("out:\n%q\nwant:\n%q", out, want)
	}
}

func TestCLIDiffUnknownBoard(t *testing.T) {
	rc, _, errOut := runCLI(t, "diff", "nope-v9", "--from", "2025-06-01", "--to", "2025-09-01")
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
