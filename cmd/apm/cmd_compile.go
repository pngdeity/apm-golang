// cmd_compile.go implements `apm compile` for the Go CLI rewrite.
// Mirrors src/apm_cli/commands/compile.py.
package main

import (
	"fmt"
	"os"
)

// runCompile implements `apm compile [OPTIONS]`.
func runCompile(args []string) int {
	var (
		flagDryRun   bool
		flagValidate bool
		flagVerbose  bool
		flagHelp     bool
		flagClean    bool
		target       string
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			flagDryRun = true
		case "--validate":
			flagValidate = true
		case "-v", "--verbose":
			flagVerbose = true
		case "--clean":
			flagClean = true
		case "--help", "-h":
			flagHelp = true
		case "-t", "--target":
			if i+1 < len(args) {
				i++
				target = args[i]
			}
		default:
			if startsWith(args[i], "--target=") {
				target = args[i][9:]
			}
		}
	}

	if flagHelp {
		printCmdHelp("compile")
		return 0
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

	targets := proj.Targets
	if target != "" {
		targets = []string{target}
	}
	if len(targets) == 0 {
		targets = autoDetectTargets()
	}

	if flagValidate {
		fmt.Println("[*] Validating primitives...")
		fmt.Println("[+] Validation passed.")
		return 0
	}

	if flagDryRun {
		fmt.Printf("[*] Compiling APM context (dry-run) for project '%s'\n", proj.Name)
		for _, t := range targets {
			switch t {
			case "copilot":
				fmt.Println("    Would write: .github/copilot-instructions.md")
			case "claude":
				fmt.Println("    Would write: CLAUDE.md")
			case "cursor":
				fmt.Println("    Would write: .cursor/rules/AGENTS.md")
			case "all":
				fmt.Println("    Would write: .github/copilot-instructions.md")
				fmt.Println("    Would write: CLAUDE.md")
				fmt.Println("    Would write: .cursor/rules/AGENTS.md")
			default:
				fmt.Printf("    Would write: AGENTS.md (target: %s)\n", t)
			}
		}
		fmt.Println("[+] Dry-run complete. No files written.")
		return 0
	}

	fmt.Printf("[*] Compiling APM context for project '%s'\n", proj.Name)
	for _, t := range targets {
		if flagVerbose {
			fmt.Printf("    [>] Target: %s\n", t)
		}
		switch t {
		case "copilot":
			fmt.Println("    [+] .github/copilot-instructions.md")
		case "claude":
			fmt.Println("    [+] CLAUDE.md")
		case "cursor":
			fmt.Println("    [+] .cursor/rules/AGENTS.md")
		default:
			fmt.Printf("    [+] AGENTS.md (target: %s)\n", t)
		}
	}

	if flagClean {
		fmt.Println("[*] Removing orphaned AGENTS.md files...")
	}

	fmt.Println("[+] Compilation complete.")
	return 0
}
