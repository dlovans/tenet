// Package main provides a CLI tool for the Tenet VM.
// This is useful for testing and batch processing of JSON schemas.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dlovans/tenet/pkg/lint"
	"github.com/dlovans/tenet/pkg/tenet"
)

func main() {
	// Define flags
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runDate := runCmd.String("date", "", "Effective date (ISO 8601 format, defaults to now)")
	runFile := runCmd.String("file", "", "Input JSON file (or use stdin)")

	verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)
	verifyNew := verifyCmd.String("new", "", "Completed document to verify")
	verifyBase := verifyCmd.String("base", "", "Original base schema")

	lintCmd := flag.NewFlagSet("lint", flag.ExitOnError)
	lintFile := lintCmd.String("file", "", "JSON schema file to lint")

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
		handleVerify(*verifyNew, *verifyBase)

	case "lint":
		lintCmd.Parse(os.Args[2:])
		handleLint(*lintFile)

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
	fmt.Println("  tenet verify -new completed.json -base schema.json")
	fmt.Println("  tenet lint -file schema.json")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  tenet run -date 2025-06-15 -file schema.json")
	fmt.Println("  cat schema.json | tenet run -date 2025-06-15")
	fmt.Println("  tenet lint -file schema.json")
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

func handleVerify(newPath, basePath string) {
	if newPath == "" || basePath == "" {
		fmt.Fprintln(os.Stderr, "Error: Both -new and -base flags are required")
		os.Exit(1)
	}

	newJson, err := os.ReadFile(newPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading new file: %v\n", err)
		os.Exit(1)
	}

	baseJson, err := os.ReadFile(basePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading base schema: %v\n", err)
		os.Exit(1)
	}

	valid, err := tenet.Verify(string(newJson), string(baseJson))
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

func handleLint(filePath string) {
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

	result, err := lint.Run(string(input))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Lint error: %v\n", err)
		os.Exit(1)
	}

	if len(result.Issues) == 0 {
		fmt.Println("✓ No issues found")
		return
	}

	// Print issues
	for _, issue := range result.Issues {
		icon := "⚠"
		if issue.Severity == "error" {
			icon = "✗"
		}
		location := ""
		if issue.Field != "" {
			location = fmt.Sprintf(" [field: %s]", issue.Field)
		}
		if issue.Rule != "" {
			location += fmt.Sprintf(" [rule: %s]", issue.Rule)
		}
		fmt.Printf("%s %s%s: %s\n", icon, issue.Severity, location, issue.Message)
	}

	if !result.Valid {
		os.Exit(1)
	}
}
