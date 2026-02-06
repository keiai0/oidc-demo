package main

import (
	"fmt"
	"log"

	"github.com/isurugi-k/oidc-demo/op/backend/config"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/infrastructure/database"
)

// マイグレーション手動実行ツール
// Usage: go run cmd/migrate/main.go
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := database.RunMigrations(cfg.DSN, "db/migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	fmt.Println("migrations completed successfully")
}
