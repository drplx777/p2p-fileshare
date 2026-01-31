package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s [up|down]\n", os.Args[0])
		_, _ = fmt.Fprintf(os.Stderr, "Env: DATABASE_URL\n")
		_, _ = fmt.Fprintf(os.Stderr, "Migrations dir: ./migrations\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		_, _ = fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(2)
	}

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	switch flag.Arg(0) {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	default:
		flag.Usage()
		os.Exit(2)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

