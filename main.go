package main

import (
	"fmt"
	"os"

	"github.com/garder500/holydb/cmd/holydb"
)

func main() {
	if err := holydb.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
