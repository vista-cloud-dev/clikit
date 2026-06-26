package clikit

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/willabides/kongplete"
)

// Run is the single entry point every CLI in the toolchain uses. It wires
// Kong (the command grammar), shell completion (kongplete), the resolved
// output Context, the deterministic error/exit-code ladder, and the styled
// help — so every tool behaves identically. It returns the process exit code.
//
//	func main() {
//	    cli := &CLI{}
//	    os.Exit(clikit.Run("hello", "…", cli, &cli.Globals))
//	}
//
// `g` must point at the Globals embedded in `cli` (populated by Parse).
func Run(name, description string, cli any, g *Globals, extra ...kong.Option) int {
	opts := []kong.Option{
		kong.Name(name),
		kong.Description(description),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true, FlagsLast: true}),
		kong.Help(helpPrinter),
	}
	opts = append(opts, extra...)

	parser, err := kong.New(cli, opts...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitUsage
	}

	// Handle shell-completion requests (no-op for normal invocations).
	kongplete.Complete(parser)

	// Two-tier discovery: a bare invocation shows a compact landing page; an
	// explicit `help` shows the full grouped surface. (`--help` is handled by
	// Kong via the custom printer above.)
	switch args := os.Args[1:]; {
	case len(args) == 0:
		return helpExit(emitHelp(os.Stdout, parser.Model, nil, false))
	case len(args) == 1 && args[0] == "help":
		return helpExit(emitHelp(os.Stdout, parser.Model, nil, true))
	}

	kctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		RenderError(NewContext(g, ""), Fail(ExitUsage, "USAGE", err.Error(), "run with --help for usage"))
		return ExitUsage
	}

	cc := NewContext(g, kctx.Command())
	// Bind the Context (and the parser, for `schema`) into command Run methods.
	if err := kctx.Run(cc, parser); err != nil {
		RenderError(cc, err)
		return exitOf(err)
	}
	return ExitOK
}

// helpExit maps a help-render error to an exit code (ExitOK on success).
func helpExit(err error) int {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitRuntime
	}
	return ExitOK
}
