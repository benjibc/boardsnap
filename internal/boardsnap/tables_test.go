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
		if r.Model != want[i] || !r.Complete {
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
		t.Fatalf("live leader: got %+v", liveRows[0])
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
