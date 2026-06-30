package clikit

import (
	"errors"
	"testing"

	"github.com/alecthomas/kong"
)

// parseErr builds a real kong.ParseError by parsing args against a grammar that
// is missing required input — the exact situation Run answers with verb help.
func parseErr(t *testing.T, grammar any, args ...string) error {
	t.Helper()
	k, err := kong.New(grammar, kong.Name("demo"))
	if err != nil {
		t.Fatal(err)
	}
	if _, perr := k.Parse(args); perr != nil {
		return perr
	}
	t.Fatalf("expected a parse error for args %v, got nil", args)
	return nil
}

// A bare verb that needs a positional argument resolves to that verb's node in
// human (text) mode, so Run can print its help instead of a terse error — but
// stays nil in JSON mode, preserving the machine-readable error envelope.
func TestUsageHelpNode_MissingPositional(t *testing.T) {
	var cli struct {
		Greet struct {
			Name string `arg:"" required:"" help:"who to greet"`
		} `cmd:"" help:"greet someone"`
	}
	err := parseErr(t, &cli, "greet") // missing required <name>

	node := usageHelpNode(err, FormatText)
	if node == nil {
		t.Fatal("text mode: expected the targeted command node, got nil")
	}
	if node.Name != "greet" {
		t.Errorf("node = %q, want \"greet\"", node.Name)
	}
	if usageHelpNode(err, FormatJSON) != nil {
		t.Error("json mode: want nil (terse structured error preserved), got a node")
	}
}

// Naming a command that requires a subcommand (none given) resolves to that
// parent node — emitHelp then lists its subcommands.
func TestUsageHelpNode_MissingSubcommand(t *testing.T) {
	var cli struct {
		Pkg struct {
			Parse struct{} `cmd:"" help:"parse"`
			Build struct{} `cmd:"" help:"build"`
		} `cmd:"" help:"pkg domain"`
	}
	err := parseErr(t, &cli, "pkg")

	node := usageHelpNode(err, FormatText)
	if node == nil || node.Name != "pkg" {
		t.Fatalf("want \"pkg\" node, got %v", node)
	}
	if node.Type != kong.CommandNode {
		t.Errorf("node.Type = %v, want CommandNode", node.Type)
	}
}

// Errors that don't name a specific command (or aren't parse errors) fall back
// to the terse structured error — usageHelpNode returns nil.
func TestUsageHelpNode_FallbackCases(t *testing.T) {
	if got := usageHelpNode(nil, FormatText); got != nil {
		t.Errorf("nil error: want nil, got %v", got)
	}
	if got := usageHelpNode(errors.New("boom"), FormatText); got != nil {
		t.Errorf("non-ParseError: want nil, got %v", got)
	}
}
