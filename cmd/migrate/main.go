package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"colossa-pm/migrations"

	"github.com/joho/godotenv"
)

const usage = `
Usage: migrate <command> [args]

Commands:
  up              Run all pending migrations
  down            Roll back all migrations
  steps <n>       Run n migrations (negative to roll back, e.g. steps -1)
  version         Print current migration version
  force <version> Force-set version to fix a dirty state
`

func main() {
	// Load .env before reading any env vars
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env vars")
	}

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Print(usage)
		os.Exit(1)
	}

	switch args[0] {
	case "up":
		if err := migrations.Up(); err != nil {
			log.Fatalf("up: %v", err)
		}
		fmt.Println("migrations applied successfully")

	case "down":
		if err := migrations.Down(); err != nil {
			log.Fatalf("down: %v", err)
		}
		fmt.Println("migrations rolled back successfully")

	case "steps":
		if len(args) < 2 {
			log.Fatal("steps requires a number argument, e.g: migrate steps -1")
		}
		n, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatalf("invalid steps argument %q: %v", args[1], err)
		}
		if err := migrations.Steps(n); err != nil {
			log.Fatalf("steps: %v", err)
		}
		fmt.Printf("%d migration step(s) applied\n", n)

	case "version":
		version, dirty, err := migrations.Version()
		if err != nil {
			log.Fatalf("version: %v", err)
		}
		fmt.Printf("version: %d  dirty: %v\n", version, dirty)

	case "force":
		if len(args) < 2 {
			log.Fatal("force requires a version number, e.g: migrate force 3")
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatalf("invalid version argument %q: %v", args[1], err)
		}
		if err := migrations.Force(v); err != nil {
			log.Fatalf("force: %v", err)
		}
		fmt.Printf("version forced to %d\n", v)

	default:
		fmt.Printf("unknown command: %q\n", args[0])
		fmt.Print(usage)
		os.Exit(1)
	}
}
