package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var migrationsDir string
	flag.StringVar(&migrationsDir, "dir", "db/migrations", "Path to migrations directory")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]
	databaseURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		fmt.Fprintf(os.Stderr, "Error: DATABASE_URL environment variable is required\n")
		os.Exit(1)
	}

	absDir, err := filepath.Abs(migrationsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving migrations path: %v\n", err)
		os.Exit(1)
	}

	sourceURL := fmt.Sprintf("file://%s", filepath.ToSlash(absDir))

	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration initialization error: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close migration source: %v\n", sourceErr)
		}
		if dbErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close migration database: %v\n", dbErr)
		}
	}()

	switch command {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Fprintf(os.Stderr, "Migration up failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Migrations applied successfully (up).")

	case "down":
		steps := 1
		if len(args) > 1 {
			s, err := strconv.Atoi(args[1])
			if err == nil && s > 0 {
				steps = s
			}
		}
		if err := m.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Fprintf(os.Stderr, "Migration down failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Migrations rolled back successfully (%d steps down).\n", steps)

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				fmt.Println("No migrations applied yet (version: none).")
				return
			}
			fmt.Fprintf(os.Stderr, "Failed to get version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Current migration version: %d (dirty: %t)\n", version, dirty)

	case "force":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: version argument required for 'force'\n")
			os.Exit(1)
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid version number '%s': %v\n", args[1], err)
			os.Exit(1)
		}
		if err := m.Force(v); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to force version %d: %v\n", v, err)
			os.Exit(1)
		}
		fmt.Printf("Forced migration version to: %d\n", v)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run ./cmd/migrate [flags] <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up              Apply all available up migrations")
	fmt.Println("  down [steps]    Roll back 1 or N migrations")
	fmt.Println("  version         Display the current migration version and dirty state")
	fmt.Println("  force <version> Force set the dirty/version flag to <version>")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -dir <path>     Migrations folder (default: db/migrations)")
}
