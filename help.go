package clikit

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"golang.org/x/term"
)

// This file is the styled, curated help renderer that replaces Kong's stock
// printer for the whole toolchain. It groups commands into editorial categories
// (from `group:""` tags), colors command names, dims summaries, and offers a
// two-tier surface: a compact landing page on a bare invocation, and the full
// grouped listing on `<tool> help` / `--help`. Rendering goes through the same
// *Context style primitives as command output, so it degrades to clean plain
// text off a TTY / under NO_COLOR automatically. See
// docs/proposals/cli-discovery-ux.md.

// ungroupedTitle is the bucket for commands carrying no `group:""` tag. It is
// always rendered last.
const ungroupedTitle = "Commands"

// helpEntry is one command row: its name (relative to its parent) and summary.
type helpEntry struct {
	name    string
	summary string
}

// helpGroup is a titled set of command rows.
type helpGroup struct {
	title   string
	entries []helpEntry
}

// newRenderContext builds a *Context for rendering help (or any standalone
// output) to an arbitrary writer, with color explicitly chosen by the caller.
// The glyph set follows the locale, matching NewContext.
func newRenderContext(w io.Writer, color bool) *Context {
	unicode := supportsUnicode()
	gl := glyphsUnicode
	if !unicode {
		gl = glyphsASCII
	}
	return &Context{
		Stdout:  w,
		Stderr:  w,
		Format:  FormatText,
		Color:   color,
		th:      newTheme(),
		gl:      gl,
		unicode: unicode,
	}
}

// helpColorEnabled reports whether help should be colored: stdout is a TTY and
// NO_COLOR is unset. (The help flag fires before global flags are bound, so we
// resolve from the environment rather than from Globals.)
func helpColorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// --- Kong glue ---------------------------------------------------------------

// helpPrinter is installed via kong.Help. It renders the selected node (or the
// root) through the styled renderer and pages the result.
func helpPrinter(_ kong.HelpOptions, kctx *kong.Context) error {
	return emitHelp(kctx.Stdout, kctx.Model, kctx.Selected(), true)
}

// emitHelp renders help for app/selected into a buffer (so color is decided by
// the real terminal, not by the immediate writer) and writes it to w, paging only
// when the output is taller than the screen.
//
//   - bare invocation (selected==nil, !full) → a COMPACT landing intro (never
//     paged), so a plain `m` lands the user back at the prompt.
//   - `help`/`--help` at the root or a group → the full grouped surface.
//   - a leaf command → usage + help + args + flags.
func emitHelp(w io.Writer, app *kong.Application, selected *kong.Node, full bool) error {
	var buf strings.Builder
	c := newRenderContext(&buf, helpColorEnabled())

	if selected == nil && !full {
		writeLanding(c, app)
		// The landing is deliberately short — never page it.
		_, err := io.WriteString(w, buf.String())
		return err
	}

	node := app.Node
	name := app.Name
	if selected != nil {
		node = selected
		name = app.Name + " " + selected.Path()
	}

	if len(node.Children) > 0 {
		// A group node (root, or e.g. `pkg`): grouped command listing.
		writeRootHelp(c, name, node.Help, groupsFrom(node), globalsOf(app), true)
	} else {
		// A leaf command: usage + help + arguments + flags.
		usage := app.Name + " " + selected.Summary()
		writeCommandHelp(c, usage, selected.Help, selected.Detail,
			cmdArgsOf(selected), cmdFlagsOf(selected, globalKeys(app)))
	}
	return pageThrough(w, buf.String(), pagerEnabled(false))
}

// writeLanding renders the compact intro shown for a bare invocation: the tool's
// tagline, a one-line overview of each command category (what each area is), and
// pointers to go deeper. It never lists every command — that's `<tool> help`.
func writeLanding(c *Context, app *kong.Application) {
	c.Subtitle(app.Name)
	if app.Help != "" {
		fmt.Fprintln(c.Stdout, app.Help)
	}
	if cats := paletteCats(app.Node); len(cats) > 0 {
		fmt.Fprintln(c.Stdout)
		pairs := make([][2]string, 0, len(cats))
		for _, cat := range cats {
			pairs = append(pairs, [2]string{cat.name, cat.desc})
		}
		c.KV(pairs...)
	}
	fmt.Fprintln(c.Stdout)
	fmt.Fprintln(c.Stdout, c.Faint(fmt.Sprintf(
		`Run "%s help" for all commands · "%s explore" to browse · "%s <command> --help" for one.`,
		app.Name, app.Name, app.Name)))
}

// groupsFrom buckets a node's command children into helpGroups by their
// `group:""` tag, preserving first-seen group order and collecting untagged
// commands into a trailing "Commands" group.
func groupsFrom(node *kong.Node) []helpGroup {
	idx := map[string]int{}
	groups := []helpGroup{}
	var ungrouped []helpEntry

	for _, child := range node.Children {
		if child.Type != kong.CommandNode || child.Hidden {
			continue
		}
		entry := helpEntry{name: child.Name, summary: child.Help}
		if child.Group == nil || child.Group.Key == "" {
			ungrouped = append(ungrouped, entry)
			continue
		}
		title := child.Group.Title
		if title == "" {
			title = child.Group.Key
		}
		i, seen := idx[title]
		if !seen {
			idx[title] = len(groups)
			groups = append(groups, helpGroup{title: title})
			i = len(groups) - 1
		}
		groups[i].entries = append(groups[i].entries, entry)
	}
	if len(ungrouped) > 0 {
		groups = append(groups, helpGroup{title: ungroupedTitle, entries: ungrouped})
	}
	return groups
}

// globalsOf returns the application-level flags (minus --help).
func globalsOf(app *kong.Application) []SchemaFlag {
	var out []SchemaFlag
	for _, f := range app.Flags {
		if f.Name == "help" {
			continue
		}
		out = append(out, flagSchema(f))
	}
	return out
}

// globalKeys is the set of application-level flag names, used to drop globals
// from a command's own flag list.
func globalKeys(app *kong.Application) map[string]bool {
	keys := map[string]bool{}
	for _, f := range app.Flags {
		keys[f.Name] = true
	}
	return keys
}

// cmdFlagsOf returns a command's own flags (minus --help and globals).
func cmdFlagsOf(cmd *kong.Node, globals map[string]bool) []SchemaFlag {
	var out []SchemaFlag
	for _, f := range cmd.Flags {
		if f.Name == "help" || globals[f.Name] {
			continue
		}
		out = append(out, flagSchema(f))
	}
	return out
}

// cmdArgsOf returns a command's positional arguments.
func cmdArgsOf(cmd *kong.Node) []SchemaArg {
	var out []SchemaArg
	for _, p := range cmd.Positional {
		out = append(out, SchemaArg{
			Name: p.Name, Type: valueType(p), Enum: splitEnum(p.Enum),
			Required: p.Required, Help: p.Help,
		})
	}
	return out
}

// --- pure renderers (testable with a buffer Context) -------------------------

// writeRootHelp renders a grouped command listing for name (the program or a
// group path), its summary, the command groups, and — when full — the global
// flags. A landing page (full=false) omits flags and points at --help.
func writeRootHelp(c *Context, name, summary string, groups []helpGroup, globals []SchemaFlag, full bool) {
	c.Subtitle(name)
	if summary != "" {
		fmt.Fprintln(c.Stdout, summary)
	}
	for _, g := range groups {
		fmt.Fprintln(c.Stdout)
		c.Title(g.title)
		writeEntries(c, g.entries)
	}
	if full && len(globals) > 0 {
		fmt.Fprintln(c.Stdout)
		c.Title("Global flags")
		writeFlags(c, globals)
	}
	fmt.Fprintln(c.Stdout)
	if full {
		fmt.Fprintln(c.Stdout, c.Faint(fmt.Sprintf("Run \"%s <command> --help\" for details on a command.", name)))
	} else {
		fmt.Fprintln(c.Stdout, c.Faint(fmt.Sprintf("Run \"%s --help\" for all commands and flags.", name)))
	}
}

// writeCommandHelp renders a leaf command: usage, help/detail prose, arguments,
// and flags.
func writeCommandHelp(c *Context, usage, help, detail string, args []SchemaArg, flags []SchemaFlag) {
	fmt.Fprintf(c.Stdout, "%s %s\n", c.Muted("Usage:"), usage)
	if help != "" {
		fmt.Fprintln(c.Stdout)
		fmt.Fprintln(c.Stdout, help)
	}
	if detail != "" {
		fmt.Fprintln(c.Stdout)
		fmt.Fprintln(c.Stdout, detail)
	}
	if len(args) > 0 {
		fmt.Fprintln(c.Stdout)
		c.Title("Arguments")
		rows := make([]helpEntry, 0, len(args))
		for _, a := range args {
			rows = append(rows, helpEntry{name: a.Name, summary: a.Help})
		}
		writeEntries(c, rows)
	}
	if len(flags) > 0 {
		fmt.Fprintln(c.Stdout)
		c.Title("Flags")
		writeFlags(c, flags)
	}
}

// writeEntries prints name→summary rows with the names aligned and accented and
// the summaries dimmed.
func writeEntries(c *Context, entries []helpEntry) {
	width := 0
	for _, e := range entries {
		if len(e.name) > width {
			width = len(e.name)
		}
	}
	for _, e := range entries {
		pad := strings.Repeat(" ", width-len(e.name))
		line := "  " + c.Accent(e.name) + pad
		if e.summary != "" {
			line += "  " + c.Faint(e.summary)
		}
		fmt.Fprintln(c.Stdout, line)
	}
}

// writeFlags prints flags as --name rows, reusing the entry formatter.
func writeFlags(c *Context, flags []SchemaFlag) {
	rows := make([]helpEntry, 0, len(flags))
	for _, f := range flags {
		name := "--" + f.Name
		if f.Short != "" {
			name = "-" + f.Short + ", " + name
		}
		rows = append(rows, helpEntry{name: name, summary: f.Help})
	}
	writeEntries(c, rows)
}
