package boardsnap

import (
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Exit codes.
const (
	ExitUsage      = 2
	ExitNoSnapshot = 3
	ExitNoBoard    = 4
	ExitNoModel    = 5
)

// Run executes the CLI. Global flags must precede the subcommand:
//
//	boardsnap [--store DIR] <command> [flags]
//
// Commands: snapshots, boards, tasks, leaders, diff, model.
func Run(argv []string, stdout, stderr io.Writer) int {
	global := flag.NewFlagSet("boardsnap", flag.ContinueOnError)
	global.SetOutput(stderr)
	store := global.String("store", "", "snapshot store directory")
	if err := global.Parse(argv); err != nil {
		return ExitUsage
	}
	args := global.Args()
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: boardsnap [--store DIR] <snapshots|boards|tasks|leaders|diff|model> [flags]")
		return ExitUsage
	}
	storeDir := *store
	if storeDir == "" {
		var err error
		storeDir, err = DefaultStore()
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitNoSnapshot
		}
	}

	cmd, rest := args[0], args[1:]
	if cmd == "snapshots" {
		dates, err := ListSnapshotDates(storeDir)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitNoSnapshot
		}
		for _, d := range dates {
			fmt.Fprintln(stdout, d)
		}
		return 0
	}

	// Remaining commands resolve a snapshot first. Flags may be interleaved
	// with the positional argument (unlike flag.Parse's stop-at-first-arg).
	var pos []string
	var flagArgs []string
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		if a == "--complete-only" {
			flagArgs = append(flagArgs, a)
		} else if a == "--as-of" || a == "--aggregate" || a == "--format" ||
			a == "--from" || a == "--to" {
			if i+1 >= len(rest) {
				fmt.Fprintf(stderr, "error: flag %s needs a value\n", a)
				return ExitUsage
			}
			flagArgs = append(flagArgs, a, rest[i+1])
			i++
		} else if strings.HasPrefix(a, "--as-of=") || strings.HasPrefix(a, "--aggregate=") ||
			strings.HasPrefix(a, "--complete-only=") || strings.HasPrefix(a, "--format=") ||
			strings.HasPrefix(a, "--from=") || strings.HasPrefix(a, "--to=") {
			flagArgs = append(flagArgs, a)
		} else {
			pos = append(pos, a)
		}
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	asOf := fs.String("as-of", "", "resolve the latest snapshot on or before DATE (YYYY-MM-DD)")
	aggregate := fs.String("aggregate", "mean-task", "mean-task|mean-type (leaders/diff only)")
	completeOnly := fs.Bool("complete-only", false, "drop models missing any board task (leaders/diff only)")
	format := fs.String("format", "table", "table|csv|json (leaders only)")
	from := fs.String("from", "", "diff: earlier as-of date (default: first snapshot)")
	to := fs.String("to", "", "diff: later as-of date (default: latest snapshot)")
	if err := fs.Parse(flagArgs); err != nil {
		return ExitUsage
	}

	if cmd == "diff" {
		if len(pos) != 1 {
			fmt.Fprintln(stderr, "usage: boardsnap diff BOARD [--from DATE] [--to DATE] [--aggregate A] [--complete-only]")
			return ExitUsage
		}
		dates, err := ListSnapshotDates(storeDir)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitNoSnapshot
		}
		fromAsOf, toAsOf := *from, *to
		if fromAsOf == "" {
			if len(dates) == 0 {
				fmt.Fprintln(stderr, "error: snapshot store is empty")
				return ExitNoSnapshot
			}
			fromAsOf = dates[0]
		}
		if toAsOf == "" && len(dates) > 0 {
			toAsOf = dates[len(dates)-1]
		}
		fromSnap, err := LoadAsOf(fromAsOf, storeDir)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitNoSnapshot
		}
		toSnap, err := LoadAsOf(toAsOf, storeDir)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitNoSnapshot
		}
		if _, ok := fromSnap.Boards[pos[0]]; !ok {
			fmt.Fprintf(stderr, "error: unknown board %q in snapshot %s\n", pos[0], fromSnap.Date)
			return ExitNoBoard
		}
		if _, ok := toSnap.Boards[pos[0]]; !ok {
			fmt.Fprintf(stderr, "error: unknown board %q in snapshot %s\n", pos[0], toSnap.Date)
			return ExitNoBoard
		}
		fromRows, err := Leaderboard(fromSnap, pos[0], *aggregate, *completeOnly)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitUsage
		}
		toRows, err := Leaderboard(toSnap, pos[0], *aggregate, *completeOnly)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitUsage
		}
		fmt.Fprintf(stdout, "from: %s\nto: %s\n", toSnap.Date, fromSnap.Date)
		fmt.Fprint(stdout, FormatDiff(DiffLeaderboards(fromRows, toRows)))
		return 0
	}

	snap, err := LoadAsOf(*asOf, storeDir)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitNoSnapshot
	}
	fmt.Fprintf(stdout, "Snapshot: %s\n", snap.Date)

	switch cmd {
	case "boards":
		fmt.Fprint(stdout, FormatBoards(SortedBoards(snap.Boards)))
		return 0
	case "tasks", "leaders":
		if len(pos) != 1 {
			fmt.Fprintf(stderr, "usage: boardsnap %s BOARD [--as-of DATE]\n", cmd)
			return ExitUsage
		}
		if _, ok := snap.Boards[pos[0]]; !ok {
			fmt.Fprintf(stderr, "error: unknown board %q in snapshot %s\n", pos[0], snap.Date)
			return ExitNoBoard
		}
		if cmd == "tasks" {
			for _, t := range snap.Boards[pos[0]].Tasks {
				fmt.Fprintln(stdout, t)
			}
			return 0
		}
		rows, err := Leaderboard(snap, pos[0], *aggregate, *completeOnly)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitUsage
		}
		switch *format {
		case "table":
			fmt.Fprint(stdout, FormatLeaders(rows))
		case "csv":
			fmt.Fprint(stdout, FormatLeadersCSV(rows))
		case "json":
			fmt.Fprint(stdout, FormatLeadersJSON(rows))
		default:
			fmt.Fprintf(stderr, "error: unknown format %q\n", *format)
			return ExitUsage
		}
		return 0
	case "model":
		if len(pos) != 1 {
			fmt.Fprintln(stderr, "usage: boardsnap model MODEL_ID [--as-of DATE]")
			return ExitUsage
		}
		row, ok := snap.Rows[pos[0]]
		if !ok {
			fmt.Fprintf(stderr, "error: unknown model %q in snapshot %s\n", pos[0], snap.Date)
			return ExitNoModel
		}
		fmt.Fprintf(stdout, "model: %s\n", row.Model)
		fmt.Fprintf(stdout, "first_seen: %s\n", snap.Date)
		tasks := make([]string, 0, len(row.Scores))
		for t := range row.Scores {
			tasks = append(tasks, t)
		}
		sort.Strings(tasks)
		for _, t := range tasks {
			fmt.Fprintf(stdout, "  %s: %.4f\n", t, row.Scores[t])
		}
		return 0
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n", cmd)
		return ExitUsage
	}
}
