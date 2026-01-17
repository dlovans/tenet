// Package main provides a CLI tool for the Tenet VM.
// This is useful for testing and batch processing of JSON schemas.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"tenet/pkg/tenet"
)

func main() {
	// Define flags
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runDate := runCmd.String("date", "", "Effective date (ISO 8601 format, defaults to now)")
	runFile := runCmd.String("file", "", "Input JSON file (or use stdin)")

	verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)
	verifyNew := verifyCmd.String("new", "", "New JSON file to verify")
	verifyOld := verifyCmd.String("old", "", "Original JSON file")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd.Parse(os.Args[2:])
		handleRun(*runDate, *runFile)

	case "verify":
		verifyCmd.Parse(os.Args[2:])
		handleVerify(*verifyNew, *verifyOld)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Tenet VM - Declarative Logic Engine for JSON Schemas")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  tenet run [-date YYYY-MM-DD] [-file input.json]")
	fmt.Println("  tenet verify -new new.json -old old.json")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  tenet run -date 2025-06-15 -file schema.json")
	fmt.Println("  cat schema.json | tenet run -date 2025-06-15")
	fmt.Println("  tenet verify -new updated.json -old original.json")
}

func handleRun(dateStr, filePath string) {
	// Parse date
	effectiveDate := time.Now()
	if dateStr != "" {
		var err error
		effectiveDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			effectiveDate, err = time.Parse(time.RFC3339, dateStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Invalid date format '%s'\n", dateStr)
				os.Exit(1)
			}
		}
	}

	// Read input
	var input []byte
	var err error

	if filePath != "" {
		input, err = os.ReadFile(filePath)
	} else {
		input, err = io.ReadAll(os.Stdin)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	// Run the VM
	result, err := tenet.Run(string(input), effectiveDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result)
}

func handleVerify(newPath, oldPath string) {
	if newPath == "" || oldPath == "" {
		fmt.Fprintln(os.Stderr, "Error: Both -new and -old flags are required")
		os.Exit(1)
	}

	newJson, err := os.ReadFile(newPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading new file: %v\n", err)
		os.Exit(1)
	}

	oldJson, err := os.ReadFile(oldPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading old file: %v\n", err)
		os.Exit(1)
	}

	valid, err := tenet.Verify(string(newJson), string(oldJson))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Verification failed: %v\n", err)
		os.Exit(1)
	}

	if valid {
		fmt.Println("✓ Document verified: transformation is legal")
	} else {
		fmt.Println("✗ Document verification failed")
		os.Exit(1)
	}
}
