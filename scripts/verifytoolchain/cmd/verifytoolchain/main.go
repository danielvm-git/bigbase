// Command verifytoolchain is the declared-toolchain contract verifier (issue #179).
//
// It reads toolchain.toml (default: ./toolchain.toml) and checks every declared
// tool is on PATH and meets its minimum version. Exits 0 if the contract is
// satisfied, 1 otherwise, printing a clear per-tool report.
//
// Usage:
//
//	go run ./scripts/verifytoolchain [-contract toolchain.toml] [-json]
//
// The scripts/verify-toolchain.sh wrapper is the canonical entry point used by
// CI (so the job stays a portable shell step).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/danielvm/bigbase/scripts/verifytoolchain"
)

func main() {
	contractPath := flag.String("contract", "toolchain.toml", "path to the toolchain contract")
	asJSON := flag.Bool("json", false, "emit results as JSON instead of a human-readable table")
	flag.Parse()

	c, err := verifytoolchain.LoadContract(*contractPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "TOOLCHAIN_CONTRACT_INVALID: %v\n", err)
		os.Exit(2)
	}

	results, verr := verifytoolchain.Verify(c, nil)

	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"contract_version": c.Meta.Version,
			"ok":               verr == nil,
			"results":          results,
		})
	} else {
		printReport(c, results)
	}

	if verr != nil {
		fmt.Fprintf(os.Stderr, "\nFAIL: %v\n", verr)
		fmt.Fprintln(os.Stderr, "\nThe CI toolchain-parity job mirrors scripts/setup-vps.sh.")
		fmt.Fprintln(os.Stderr, "A required tool missing here would be missing on the VPS at deploy time.")
		os.Exit(1)
	}
	fmt.Printf("\nPASS: declared toolchain contract satisfied (%d required, %d optional checked).\n",
		len(c.Tools.Required), len(c.Tools.Optional))
}

func printReport(c *verifytoolchain.Contract, results []verifytoolchain.Result) {
	fmt.Printf("Declared toolchain contract v%d: %s\n", c.Meta.Version, c.Meta.Description)
	fmt.Println(strings.Repeat("-", 78))
	fmt.Printf("%-12s %-9s %-14s %-14s %s\n", "TOOL", "STATUS", "FOUND", "MIN", "PATH")
	fmt.Println(strings.Repeat("-", 78))
	for _, r := range results {
		status := statusLabel(r)
		found := "-"
		if r.Found != nil {
			found = parsedStringDisplay(*r.Found)
		} else if r.FoundStr != "" {
			found = r.FoundStr
		}
		min := r.FloorStr
		if min == "" {
			min = "(any)"
		}
		fmt.Printf("%-12s %-9s %-14s %-14s %s\n", r.Tool, status, found, min, orDash(r.Path))
		if r.Diagnostic != "" {
			fmt.Printf("             -> %s\n", r.Diagnostic)
		}
		if r.Skip != "" {
			fmt.Printf("             -> skipped: %s\n", r.Skip)
		}
	}
}

func statusLabel(r verifytoolchain.Result) string {
	switch {
	case r.Skip != "":
		return "skip"
	case !r.Present && !r.Required:
		return "absent"
	case r.OK:
		return "ok"
	case !r.Present:
		return "MISSING"
	default:
		return "LOW"
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func parsedStringDisplay(v verifytoolchain.Version) string {
	// Reuse the same formatting the package uses internally.
	return fmt.Sprintf("%d.%d%s", v.Major, v.Minor, patchSuffix(v))
}

func patchSuffix(v verifytoolchain.Version) string {
	if v.HasPatch {
		return fmt.Sprintf(".%d", v.Patch)
	}
	return ""
}
