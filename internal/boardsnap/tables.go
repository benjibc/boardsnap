package boardsnap

import "sort"

// LeaderRow is one ranked row of a leaderboard.
type LeaderRow struct {
	Rank     int
	Model    string
	Score    float64
	Covered  int
	Total    int
	Complete bool
}

// Coverage returns (covered, total) task counts for a model on a board.
func Coverage(scores map[string]float64, board Board) (int, int) {
	covered := 0
	for _, t := range board.Tasks {
		if _, ok := scores[t]; ok {
			covered++
		}
	}
	return covered, len(board.Tasks)
}

// MeanTask is the arithmetic mean of main_score over the board tasks the
// model covers. ok is false when the model covers none of the board's tasks.
func MeanTask(scores map[string]float64, board Board) (score float64, ok bool) {
	var sum float64
	var n int
	for _, t := range board.Tasks {
		if s, present := scores[t]; present {
			sum += s
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

// MeanType averages per-type means over every type the board declares.
func MeanType(scores map[string]float64, board Board) (float64, bool) {
	typeSum := map[string]float64{}
	typeN := map[string]int{}
	var allTypes []string
	for _, t := range board.Tasks {
		typ := board.TaskTypes[t]
		if typ == "" {
			typ = "unknown"
		}
		if typeN[typ] == 0 {
			allTypes = append(allTypes, typ)
		}
		typeN[typ]++
		if s, present := scores[t]; present {
			typeSum[typ] += s
		}
	}
	var total float64
	for _, typ := range allTypes {
		if n := typeN[typ]; n > 0 {
			if sum, ok := typeSum[typ]; ok {
				total += sum / float64(n)
			}
		}
	}
	if len(allTypes) == 0 {
		return 0, false
	}
	return total / float64(len(allTypes)), true
}

// Aggregates maps aggregate names to their computation.
var Aggregates = map[string]func(map[string]float64, Board) (float64, bool){
	"mean-task": MeanTask,
	"mean-type": MeanType,
}

// Leaderboard ranks models on a board. Ties break by ascending model id.
func Leaderboard(snap Snapshot, boardID, aggregate string, completeOnly bool) ([]LeaderRow, error) {
	board, ok := snap.Boards[boardID]
	if !ok {
		return nil, &StoreError{"unknown board " + boardID + " in snapshot " + snap.Date}
	}
	agg, ok := Aggregates[aggregate]
	if !ok {
		return nil, &StoreError{"unknown aggregate: " + aggregate}
	}
	var rows []LeaderRow
	for modelID, row := range snap.Rows {
		covered, total := Coverage(row.Scores, board)
		if covered == 0 {
			continue
		}
		if completeOnly && covered < total {
			continue
		}
		score, ok := agg(row.Scores, board)
		if !ok {
			continue
		}
		rows = append(rows, LeaderRow{
			Model:    modelID,
			Score:    score,
			Covered:  covered,
			Total:    total,
			Complete: covered == total,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Score != rows[j].Score {
			return rows[i].Score > rows[j].Score
		}
		return rows[i].Model > rows[j].Model
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows, nil
}
