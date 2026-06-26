// Package clikit is the shared CLI convention layer for the vista-cloud-dev Go
// toolchain — the standalone, importable module both the m-* and v-* tools build
// on (extracted from go-cli-template 2026-06-25 so the `v` umbrella can mount
// multiple domains that share one clikit.Context type; see v-cli-platform.md §6).
//
// Every Go CLI in the toolchain (m-cli, irissync, the `v` umbrella + its v-*
// domains such as v-pkg, m-dev-tools-mcp, …) is built on this package so they
// share one command grammar, one --output/JSON contract, one error/exit-code
// ladder, and one TTY-gated styling layer.
package clikit

// OutputFormat is the resolved render mode for a single invocation.
type OutputFormat string

const (
	// FormatText renders human-facing output (styled when stdout is a TTY).
	FormatText OutputFormat = "text"
	// FormatJSON renders the machine-readable envelope (§5.5).
	FormatJSON OutputFormat = "json"
)

// Globals are the flags every CLI in the toolchain shares. Embed it in the
// root command struct, e.g.:
//
//	type CLI struct {
//	    clikit.Globals
//	    Fmt FmtCmd `cmd:"" help:"…"`
//	}
type Globals struct {
	Output  string `short:"o" enum:"text,json,auto" default:"auto" help:"Output: text (styled on a TTY), json (machine-readable), or auto."`
	NoColor bool   `name:"no-color" env:"NO_COLOR" help:"Disable ANSI styling even on a TTY."`
	Verbose bool   `short:"v" help:"Verbose diagnostics to stderr."`
}
