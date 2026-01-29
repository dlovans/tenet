//go:build js && wasm

// Package main provides WASM bindings for the Tenet VM.
// This allows the VM to run in browsers for reactive UI validation.
package main

import (
	"encoding/json"
	"syscall/js"
	"time"

	"github.com/dlovans/tenet/pkg/tenet"
)

func main() {
	// Export TenetRun function to JavaScript
	js.Global().Set("TenetRun", js.FuncOf(tenetRun))

	// Export TenetVerify function to JavaScript
	js.Global().Set("TenetVerify", js.FuncOf(tenetVerify))

	// Keep the Go runtime alive
	select {}
}

// tenetRun is the JS-callable wrapper for tenet.Run()
// Usage: TenetRun(jsonString, isoDateString) -> { result: string, error?: string }
func tenetRun(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return makeError("TenetRun requires 2 arguments: jsonText, dateString")
	}

	jsonText := args[0].String()
	dateStr := args[1].String()

	// Parse the date
	effectiveDate, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		// Try simpler date format
		effectiveDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return makeError("Invalid date format. Use ISO 8601 (YYYY-MM-DD or RFC3339)")
		}
	}

	// Execute the VM
	result, err := tenet.Run(jsonText, effectiveDate)
	if err != nil {
		return makeError(err.Error())
	}

	return makeResult(result)
}

// tenetVerify is the JS-callable wrapper for tenet.Verify()
// Usage: TenetVerify(newJsonString, oldJsonString) -> { valid: boolean, error?: string }
func tenetVerify(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return makeError("TenetVerify requires 2 arguments: newJson, oldJson")
	}

	newJson := args[0].String()
	oldJson := args[1].String()

	valid, err := tenet.Verify(newJson, oldJson)
	if err != nil {
		return map[string]any{
			"valid": false,
			"error": err.Error(),
		}
	}

	return map[string]any{
		"valid": valid,
	}
}

// makeError creates a JS-friendly error response
func makeError(msg string) map[string]any {
	return map[string]any{
		"error": msg,
	}
}

// makeResult creates a JS-friendly success response
func makeResult(jsonStr string) map[string]any {
	// Parse the result to return as a JS object instead of string
	var result any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// Fall back to string if parsing fails
		return map[string]any{
			"result": jsonStr,
		}
	}

	return map[string]any{
		"result": result,
	}
}
