# boardsnap

`boardsnap` is a small, deterministic CLI for querying **frozen, dated
standings snapshots** of fictional model boards. Every ranking input is an
immutable on-disk snapshot; the CLI never fetches anything from the network.

The tool exists for people who must answer questions like

> "Which model led the `helio-text-v1` board on `mean-task`, considering only
> models with a score for every task, **as of 2025-08-01**?"

and get the same byte-exact answer every time.

## Build

```sh
go build -o boardsnap .
```

The binary looks for a `snapshots/` directory first beside the executable,
then at or above the working directory; `--store DIR` overrides.

## Data plane

Snapshots live under `snapshots/`:

```
snapshots/
  2025-03-15/
    boards.json            # board registry: id, name, task ids, task types
    scores/<org>__<model>.json
  2025-06-01/
    ...
  2025-08-31/
    ...
```

A score file records one model's results at that snapshot:

```json
{
  "model": "luminode/glow-7b",
  "first_seen": "2025-02-11",
  "results": {
    "quasar-news-retrieval": {"main_score": 0.4121, "split": "test"}
  }
}
```

Snapshot directories are named by their freeze date (`YYYY-MM-DD`) and are
never edited after creation. New models appear only in snapshots frozen on or
after their `first_seen` date.

## Commands

```
boardsnap snapshots                              # list snapshot freeze dates
boardsnap boards [--as-of DATE]                  # boards in the resolved snapshot
boardsnap tasks BOARD [--as-of DATE]             # a board's task ids
boardsnap leaders BOARD [--as-of DATE]
    [--aggregate mean-task|mean-type] [--complete-only] [--format table|csv|json]
boardsnap diff BOARD [--from DATE] [--to DATE] [--aggregate A] [--complete-only]
boardsnap model MODEL_ID [--as-of DATE]          # one model's scores
```

### As-of resolution

`--as-of DATE` resolves to the **latest snapshot whose freeze date is ≤ DATE**.
If no snapshot is old enough, the command exits with code 3 and prints
`error: no snapshot on or before <DATE>` to stderr. Without `--as-of`, the
latest snapshot is used. The resolved freeze date is printed as the first
output line so callers can verify which frozen table answered the query.

### Ranking rules (`leaders`)

- `mean-task`: arithmetic mean of `main_score` over the board tasks the model
  covers.
- `mean-type`: mean of per-type means (each task carries a `type`; type means
  are averaged first).
- `--complete-only`: drop any model missing a `main_score` for **at least one**
  board task. Without it, incomplete models are ranked by the mean of the
  scores they do have and their row is marked with a trailing ` (partial)`.
- Scores print with four decimals; ties break by ascending model id.
- Table columns: `rank`, `model`, `score`, `tasks_covered/tasks_total`.
- `--format csv` emits `rank,model,score,covered,total,complete`; `--format
  json` emits a stable-key JSON array.

### Rank movement (`diff`)

`diff` resolves two snapshots (`--from` / `--to`, defaulting to the first and
latest), ranks the same board in both, and prints each model's `from_rank`,
`to_rank`, and both scores. Models absent from one side print `-`.

## Exit codes

| code | meaning |
|---|---|
| 2 | usage error (bad flags, unknown aggregate/format/command) |
| 3 | no snapshot on or before the requested date |
| 4 | unknown board in the resolved snapshot |
| 5 | unknown model in the resolved snapshot |

## Development

```sh
go test ./...
```
