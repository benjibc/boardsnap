// boardsnap is a deterministic CLI for querying frozen, dated
// model-standings snapshots. It never touches the network: every ranking
// input is an immutable on-disk snapshot selected by freeze date.
package main

import (
	"os"

	"github.com/benjibc/boardsnap/internal/boardsnap"
)

func main() {
	os.Exit(boardsnap.Run(os.Args[1:], os.Stdout, os.Stderr))
}
