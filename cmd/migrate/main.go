package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	dir := flag.String("dir", "migrations", "directory with migration files")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: migrate [-dir <dir>] <command> [args]")
		fmt.Println("Commands: up, down, status, create <name>")
		os.Exit(1)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("set dialect: %v", err)
	}

	command := args[0]
	commandArgs := args[1:]

	if err := goose.RunContext(context.Background(), command, db, *dir, commandArgs...); err != nil {
		log.Fatalf("goose %s: %v", command, err)
	}
}
