package main

import (
	"fmt"
	"log"
	"os"

	"github.com/isurugi-k/oidc-demo/op/backend/config"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/database"
)

// 開発用シードデータ投入ツール
// Usage: go run cmd/seed/main.go
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.NewDB(cfg.DSN)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	seedSQL, err := os.ReadFile("db/seeds/seed.sql")
	if err != nil {
		log.Fatalf("failed to read seed file: %v", err)
	}

	if err := db.Exec(string(seedSQL)).Error; err != nil {
		log.Fatalf("failed to execute seed: %v", err)
	}

	fmt.Println("seed data inserted successfully")
}
