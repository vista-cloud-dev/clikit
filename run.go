package clikit

import (
	"errors"
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
		cc := NewContext(g, "")
		// When the user names a command but leaves out its required arguments
		// (or its subcommand), answer with that verb's help — the terse "expected
		// <arg>" line alone doesn't show what the command actually wants. JSON
		// mode keeps the structured error so machine consumers are unaffected.
		if node := usageHelpNode(err, cc.Format); node != nil {
			RenderError(cc, Fail(ExitUsage, "USAGE", err.Error(), ""))
			_ = emitHelp(os.Stdout, parser.Model, node, true)
			return ExitUsage
		}
		RenderError(cc, Fail(ExitUsage, "USAGE", err.Error(), "run with --help for usage"))
		return ExitUsage
	}

	cc := NewContext(g, kctx.Command())
	// Bind the Context (and the parser, for `schema`) into command Run methods.
	if err := kctx.Run(cc, parser); err != nil {
		// A command can discover missing required input only while running (e.g.
		// an engine verb that needs a container). When it signals that with a
		// USAGE-coded error, give it the same help-on-usage-error treatment as a
		// parse error: the reason line plus the command's help, in human mode.
		if isUsageError(err) {
			if node := helpNodeForUsageError(kctx.Selected(), cc.Format); node != nil {
				RenderError(cc, err)
				_ = emitHelp(os.Stdout, parser.Model, node, true)
				return exitOf(err)
			}
		}
		RenderError(cc, err)
		return exitOf(err)
	}
	return ExitOK
}

// usageHelpNode decides whether a parse error should be answered with a
// command's help rather than the terse structured usage error. It returns that
// command node only when the error resolved to a specific named command — i.e.
// the user typed a verb that needs more input. Otherwise it returns nil and Run
// falls back to the structured error (unknown commands, root-level flag errors).
func usageHelpNode(err error, format OutputFormat) *kong.Node {
	var pe *kong.ParseError
	if !errors.As(err, &pe) || pe.Context == nil {
		return nil
	}
	return helpNodeForUsageError(pe.Context.Selected(), format)
}

// helpNodeForUsageError is the shared gate for both the parse-time and run-time
// usage paths: show a command's help only when the output is human-facing (not
// JSON) and a specific command node was resolved. nil → fall back to the terse
// structured error (preserving the machine-readable envelope for JSON consumers).
func helpNodeForUsageError(sel *kong.Node, format OutputFormat) *kong.Node {
	if format == FormatJSON || sel == nil || sel.Type != kong.CommandNode {
		return nil
	}
	return sel
}

// isUsageError reports whether a command's run-time error is a usage error
// (a clikit.Error carrying ExitUsage) — i.e. the command found missing or
// invalid required input only while running, and wants the help treatment.
func isUsageError(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Exit == ExitUsage
}

// helpExit maps a help-render error to an exit code (ExitOK on success).
func helpExit(err error) int {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitRuntime
	}
	return ExitOK
}
