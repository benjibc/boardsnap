package boardsnap

import "testing"

func almost(got, want float64) bool {
	d := got - want
	if d < 0 {
		d = -d
	}
	return d < 5e-5
}

func TestCompleteOnlyDropsPartialModels(t *testing.T) {
	snap, err := LoadAsOf("2025-03-16", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := Leaderboard(snap, "helio-text-v1", "mean-task", true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"pelora/pl-text-large", "nuvixa/embro-7b", "zephira-labs/zpl-mini"}
	if len(rows) != len(want) {
		t.Fatalf("rows: got %d, want %d", len(rows), len(want))
	}
	for i, r := range rows {
		if r.Model != want[i] {
			t.Fatalf("row %d: got %+v", i, r)
		}
	}
}

func TestPartialModelsRankByCoveredMean(t *testing.T) {
	snap, err := LoadAsOf("2025-03-16", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := Leaderboard(snap, "helio-text-v1", "mean-task", false)
	if err != nil {
		t.Fatal(err)
	}
	top := rows[0]
	if top.Model != "kantrel/kds-retro-base" || top.Complete {
		t.Fatalf("top: got %+v", top)
	}
	if top.Covered != 5 || top.Total != 8 {
		t.Fatalf("coverage: got %d/%d", top.Covered, top.Total)
	}
	if !almost(top.Score, 0.6706) {
		t.Fatalf("score: got %.6f", top.Score)
	}
}

func TestMeanTaskValues(t *testing.T) {
	snap, err := LoadAsOf("2025-03-16", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := Leaderboard(snap, "helio-text-v1", "mean-task", false)
	byModel := map[string]LeaderRow{}
	for _, r := range rows {
		byModel[r.Model] = r
	}
	if !almost(byModel["pelora/pl-text-large"].Score, 0.6403) {
		t.Fatalf("pelora: got %.6f", byModel["pelora/pl-text-large"].Score)
	}
	if !almost(byModel["nuvixa/embro-7b"].Score, 0.6290) {
		t.Fatalf("embro-7b: got %.6f", byModel["nuvixa/embro-7b"].Score)
	}
}

func TestMeanTypeReweightsTypes(t *testing.T) {
	snap, err := LoadAsOf("2025-03-16", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := Leaderboard(snap, "helio-text-v1", "mean-type", false)
	byModel := map[string]LeaderRow{}
	for _, r := range rows {
		byModel[r.Model] = r
	}
	if !almost(byModel["pelora/pl-text-large"].Score, 0.6453) {
		t.Fatalf("pelora mean-type: got %.6f", byModel["pelora/pl-text-large"].Score)
	}
	if !almost(byModel["kantrel/kds-retro-base"].Score, 0.3503) {
		t.Fatalf("kantrel mean-type: got %.6f", byModel["kantrel/kds-retro-base"].Score)
	}
}

func TestAsOfExcludesLaterModels(t *testing.T) {
	snap, err := LoadAsOf("2025-06-01", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := Leaderboard(snap, "helio-text-v1", "mean-task", true)
	for _, r := range rows {
		if r.Model == "nuvixa/embro-ultra" {
			t.Fatal("embro-ultra must not appear in the 2025-06-01 snapshot")
		}
	}
	if rows[0].Model != "pelora/pl-text-large" {
		t.Fatalf("leader at 2025-06-01: got %q", rows[0].Model)
	}

	live, err := LoadAsOf("", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	liveRows, _ := Leaderboard(live, "helio-text-v1", "mean-task", true)
	if liveRows[0].Model != "nuvixa/embro-ultra" || !almost(liveRows[0].Score, 0.7010) {
		t.Fatalf("latest leader: got %+v", liveRows[0])
	}
}

func TestSecondBoardIsIndependent(t *testing.T) {
	snap, err := LoadAsOf("2025-08-31", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := Leaderboard(snap, "vireo-code-v1", "mean-task", true)
	want := []string{"pelora/pl-code-mid", "kantrel/kds-forge-3b"}
	if len(rows) != len(want) {
		t.Fatalf("rows: got %d, want %d", len(rows), len(want))
	}
	for i, r := range rows {
		if r.Model != want[i] {
			t.Fatalf("row %d: got %q, want %q", i, r.Model, want[i])
		}
	}
}

func TestUnknownBoardAndAggregate(t *testing.T) {
	snap, err := LoadAsOf("2025-06-01", testStore(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Leaderboard(snap, "nope-v9", "mean-task", false); err == nil {
		t.Fatal("expected unknown board error")
	}
	if _, err := Leaderboard(snap, "helio-text-v1", "median", false); err == nil {
		t.Fatal("expected unknown aggregate error")
	}
}

func TestMeanTypeUncoveredTypesExcluded(t *testing.T) {
	// A board with four types where the model covers only retrieval and sts:
	// the uncovered types must not drag the mean toward zero, and each
	// covered type's mean is over that type's covered tasks only.
	board := Board{
		ID:    "synthetic-v1",
		Tasks: []string{"t1", "t2", "t3", "t4", "t5"},
		TaskTypes: map[string]string{
			"t1": "retrieval", "t2": "retrieval",
			"t3": "sts", "t4": "classification", "t5": "clustering",
		},
	}
	scores := map[string]float64{"t1": 0.60, "t2": 0.40, "t3": 0.80}
	got, ok := MeanType(scores, board)
	if !ok {
		t.Fatal("expected a score")
	}
	want := (0.50 + 0.80 + 0.0 + 0.0) / 4 // uncovered types count as zero (current behavior)
	if !almost(got, want) {
		t.Fatalf("MeanType: got %.6f, want %.6f", got, want)
	}
	// mean-task over covered tasks for the same model:
	mt, _ := MeanTask(scores, board)
	if !almost(mt, (0.60+0.40+0.80)/3) {
		t.Fatalf("MeanTask: got %.6f", mt)
	}
}

func TestLeaderboardTieBreakAscendingModel(t *testing.T) {
	// Equal scores must tie-break by ascending model id (README: "ties break
	// by ascending model id").
	snap := Snapshot{
		Date:   "2099-01-01",
		Boards: map[string]Board{},
		Rows:   map[string]ModelRow{},
	}
	snap.Boards["b-v1"] = Board{ID: "b-v1", Tasks: []string{"t1", "t2"}, TaskTypes: map[string]string{"t1": "retrieval", "t2": "sts"}}
	for _, m := range []string{"zeta/zzz-1", "alpha/aaa-1", "mid/mmm-1"} {
		snap.Rows[m] = ModelRow{Model: m, Scores: map[string]float64{"t1": 0.5000, "t2": 0.5000}}
	}
	rows, err := Leaderboard(snap, "b-v1", "mean-task", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"zeta/zzz-1", "mid/mmm-1", "alpha/aaa-1"}
	for i, r := range rows {
		if r.Model != want[i] {
			t.Fatalf("row %d: got %q, want %q", i, r.Model, want[i])
		}
	}
}

func TestCompleteOnlyDropsAnyMissingTask(t *testing.T) {
	// A model covering 7 of 8 board tasks must be dropped by --complete-only:
	// complete coverage means every board task, not all-but-one.
	snap := Snapshot{
		Date:   "2099-01-01",
		Boards: map[string]Board{"b-v1": {ID: "b-v1", Tasks: []string{"t1", "t2", "t3"}}},
		Rows:   map[string]ModelRow{},
	}
	snap.Rows["full/m1"] = ModelRow{Model: "full/m1", Scores: map[string]float64{"t1": 0.5, "t2": 0.5, "t3": 0.5}}
	snap.Rows["seven/m2"] = ModelRow{Model: "seven/m2", Scores: map[string]float64{"t1": 0.9, "t2": 0.9}}
	rows, err := Leaderboard(snap, "b-v1", "mean-task", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Model != "full/m1" {
		t.Fatalf("--complete-only must drop any model missing a board task: %+v", rows)
	}
}
