// Package main provides the CLI entrypoint for caster-generator.
//
// caster-generator is a semi-automated Go codegen tool that:
//   - Parses Go packages (AST + go/types) to understand structs and relationships
//   - Suggests field mappings best-effort
//   - Lets humans review + lock mappings via YAML
//   - Generates fast caster functions
package main

import (
	"fmt"
	"os"
)

func main() {
	// TODO: Implement CLI commands: analyze, suggest, gen, check
	fmt.Println("caster-generator - a semi-automated Go struct mapping codegen tool")
	fmt.Println("Commands: analyze | suggest | gen | check")
	fmt.Println("Run with -help for usage information")
	os.Exit(0)
}
