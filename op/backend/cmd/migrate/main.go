package main

import (
	"fmt"
	"log"
	"os"

	"github.com/isurugi-k/oidc-demo/op/backend/config"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/database"
)

// マイグレーションツール
// Usage:
//
//	go run cmd/migrate/main.go          — マイグレーション実行
//	go run cmd/migrate/main.go fresh    — 全テーブル削除 → 再作成 → seed
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	subcommand := ""
	if len(os.Args) > 1 {
		subcommand = os.Args[1]
	}

	switch subcommand {
	case "fresh":
		fmt.Println("dropping all tables and re-running migrations...")
		if err := database.FreshMigrations(cfg.DSN, "db/migrations"); err != nil {
			log.Fatalf("fresh migration failed: %v", err)
		}
		fmt.Println("migrations completed successfully")

		// seed も自動実行
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

	default:
		if err := database.RunMigrations(cfg.DSN, "db/migrations"); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		fmt.Println("migrations completed successfully")
	}
}
