package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver

	"frontend-backend/internal/config"
	"frontend-backend/internal/server"
	"frontend-backend/internal/storage"
)

func main() {
	configPath := flag.String("c", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		fmt.Println("Usage: go run cmd/main.go [-c <config_file_path>]\nExample: go run cmd/main.go -c config.yaml")
		os.Exit(1)
	}

	dbinfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)

	db, err := sql.Open("postgres", dbinfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully connected to database!")

	store := storage.NewPostgresStorage(db)
	server := server.NewServer(store)

	log.Fatal(http.ListenAndServe(":8080", server))
}
