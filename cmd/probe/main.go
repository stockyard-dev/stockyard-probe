package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stockyard-dev/stockyard-probe/internal/server"
	"github.com/stockyard-dev/stockyard-probe/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./probe-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("probe: open database: %v", err)
	}
	defer db.Close()

	srv := server.New(db, server.DefaultLimits())

	fmt.Printf("\n  Probe — Self-hosted HTTP request inspector\n")
	fmt.Printf("  ─────────────────────────────────\n")
	fmt.Printf("  Dashboard:  http://localhost:%s/ui\n", port)
	fmt.Printf("  API:        http://localhost:%s/api\n", port)
	fmt.Printf("  Data:       %s\n", dataDir)
	fmt.Printf("  ─────────────────────────────────\n\n")

	log.Printf("probe: listening on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("probe: %v", err)
	}
}
