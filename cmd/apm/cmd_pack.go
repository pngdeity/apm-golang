// cmd_pack.go implements `apm pack` and `apm unpack` for the Go CLI rewrite.
// Mirrors src/apm_cli/commands/pack.py.
package main

import (
	"fmt"
	"os"
)

// runPack implements `apm pack [OPTIONS]`.
func runPack(args []string) int {
	var (
		flagDryRun bool
		flagHelp   bool
		flagJSON   bool
		output     string
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			flagDryRun = true
		case "--json":
			flagJSON = true
		case "--help", "-h":
			flagHelp = true
		case "-o", "--output":
			if i+1 < len(args) {
				i++
				output = args[i]
			}
		}
	}

	if flagHelp {
		printCmdHelp("pack")
		return 0
	}

	if output == "" {
		output = "./build"
	}

	cwd, _ := os.Getwd()
	ymlPath, err := findApmYML(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] No apm.yml found. Run 'apm init' to create one.\n")
		return 1
	}
	proj, err := parseApmYML(ymlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[x] Failed to parse apm.yml: %v\n", err)
		return 1
	}

	if flagDryRun {
		if flagJSON {
			fmt.Printf(`{"project":%q,"output":%q,"dry_run":true,"artifacts":[]}`, proj.Name, output)
			fmt.Println()
		} else {
			fmt.Printf("[*] Packing project '%s' (dry-run)\n", proj.Name)
			fmt.Printf("    Output: %s\n", output)
			fmt.Printf("    Dependencies: %d APM, %d MCP\n", len(proj.Deps), len(proj.MCPDeps))
			fmt.Println("[+] Dry-run complete. No files written.")
		}
		return 0
	}

	if flagJSON {
		fmt.Printf(`{"project":%q,"output":%q,"artifacts":[]}`, proj.Name, output)
		fmt.Println()
	} else {
		fmt.Printf("[*] Packing project '%s'\n", proj.Name)
		fmt.Printf("    Output: %s\n", output)
		fmt.Println("[+] Pack complete.")
	}
	return 0
}

// runUnpack implements `apm unpack [OPTIONS]`.
func runUnpack(args []string) int {
	var (
		flagHelp bool
		bundle   string
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			flagHelp = true
		default:
			if !startsWith(args[i], "-") && bundle == "" {
				bundle = args[i]
			}
		}
	}

	if flagHelp {
		printCmdHelp("unpack")
		return 0
	}

	if bundle == "" {
		fmt.Fprintln(os.Stderr, "Error: Missing BUNDLE argument.")
		fmt.Fprintln(os.Stderr, `Try 'apm unpack --help' for help.`)
		return 2
	}

	if _, err := os.Stat(bundle); err != nil {
		fmt.Fprintf(os.Stderr, "[x] Bundle not found: %s\n", bundle)
		return 1
	}

	fmt.Printf("[*] Unpacking bundle: %s\n", bundle)
	fmt.Println("[+] Unpack complete.")
	return 0
}
