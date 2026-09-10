package boardsnap

import (
	"fmt"
	"sort"
	"strings"
)

// LeadersHeader and LeadersSep head every leaders table.
const (
	LeadersHeader = "rank  model                          score   tasks"
	LeadersSep    = "----  -----                          -----   -----"
)

// FormatLeaders renders the deterministic leaders table (byte-exact).
func FormatLeaders(rows []LeaderRow) string {
	var b strings.Builder
	b.WriteString(LeadersHeader + "\n")
	b.WriteString(LeadersSep + "\n")
	for _, r := range rows {
		partial := ""
		if !r.Complete {
			partial = " (partial)"
		}
		fmt.Fprintf(&b, "%-4d  %-30s %.4f  %d/%d%s\n", r.Rank, r.Model, r.Score, r.Covered, r.Total, partial)
	}
	return b.String()
}

// FormatBoards renders the boards table (byte-exact).
func FormatBoards(boards []Board) string {
	var b strings.Builder
	b.WriteString("board                          name                       tasks\n")
	b.WriteString("-----                          ----                       ----\n")
	for _, bd := range boards {
		fmt.Fprintf(&b, "%-30s %-26s %d\n", bd.ID, bd.Name, len(bd.Tasks))
	}
	return b.String()
}

// SortedBoards returns boards ordered by id for deterministic output.
func SortedBoards(m map[string]Board) []Board {
	out := make([]Board, 0, len(m))
	for _, b := range m {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// FormatLeadersCSV renders rows as CSV: rank,model,score,covered,total,complete.
func FormatLeadersCSV(rows []LeaderRow) string {
	var b strings.Builder
	b.WriteString("rank,model,score,covered,total,complete\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "%d,%s,%.4f,%d,%d,%t\n", r.Rank, r.Model, r.Score, r.Covered, r.Total, r.Complete)
	}
	return b.String()
}

// FormatLeadersJSON renders rows as a JSON array (stable key order, 4-decimal scores).
func FormatLeadersJSON(rows []LeaderRow) string {
	var b strings.Builder
	b.WriteString("[\n")
	for i, r := range rows {
		comma := ","
		if i == len(rows)-1 {
			comma = ""
		}
		fmt.Fprintf(&b, "  {\"rank\": %d, \"model\": %q, \"score\": %.4f, \"tasks_covered\": %d, \"tasks_total\": %d, \"complete\": %t}%s\n",
			r.Rank, r.Model, r.Score, r.Covered, r.Total, r.Complete, comma)
	}
	b.WriteString("]\n")
	return b.String()
}

// DiffRow is one model's rank movement between two snapshots.
type DiffRow struct {
	Model    string
	FromRank int // 0 when absent from the "from" table
	ToRank   int // 0 when absent from the "to" table
	From     *float64
	To       *float64
}

// FormatDiff renders rank movement rows (byte-exact). Absent sides print "-".
func FormatDiff(rows []DiffRow) string {
	var b strings.Builder
	b.WriteString("model                          from_rank  to_rank  from_score  to_score\n")
	b.WriteString("-----                          ---------  -------  ----------  --------\n")
	for _, r := range rows {
		fromRank, toRank := "-", "-"
		fromScore, toScore := "-", "-"
		if r.FromRank > 0 {
			fromRank = fmt.Sprintf("%d", r.FromRank)
		}
		if r.ToRank > 0 {
			toRank = fmt.Sprintf("%d", r.ToRank)
		}
		if r.From != nil {
			fromScore = fmt.Sprintf("%.4f", *r.From)
		}
		if r.To != nil {
			toScore = fmt.Sprintf("%.4f", *r.To)
		}
		fmt.Fprintf(&b, "%-30s %-10s %-8s %-11s %s\n", r.Model, fromRank, toRank, fromScore, toScore)
	}
	return b.String()
}

// DiffLeaderboards joins two leaderboards into rank-movement rows, ordered by
// to-rank (present rows first), then from-rank, then model id.
func DiffLeaderboards(from, to []LeaderRow) []DiffRow {
	byModel := map[string]*DiffRow{}
	for _, r := range from {
		s := r.Score
		byModel[r.Model] = &DiffRow{Model: r.Model, FromRank: r.Rank, From: &s}
	}
	for _, r := range to {
		s := r.Score
		if d, ok := byModel[r.Model]; ok {
			d.ToRank = r.Rank
			d.To = &s
		} else {
			byModel[r.Model] = &DiffRow{Model: r.Model, ToRank: r.Rank, To: &s}
		}
	}
	out := make([]DiffRow, 0, len(byModel))
	for _, d := range byModel {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.ToRank == 0) != (b.ToRank == 0) {
			return b.ToRank == 0 // present-in-"to" first
		}
		if a.ToRank != b.ToRank {
			return a.ToRank < b.ToRank
		}
		if (a.FromRank == 0) != (b.FromRank == 0) {
			return b.FromRank == 0
		}
		if a.FromRank != b.FromRank {
			return a.FromRank < b.FromRank
		}
		return a.Model < b.Model
	})
	return out
}
