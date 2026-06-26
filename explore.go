package clikit

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// This file is the interactive `explore` palette (Phase 2): a Bubbletea TUI over
// the CLI's own command tree. You arrow through editorial groups and commands,
// read a detail strip on focus (with a runnable/needs-args badge), descend into
// sub-domains, fuzzy-filter, and select a command. It is mounted as the reusable
// ExploreCmd, so every tool in the toolchain gets `explore` for free. The
// navigation logic lives in palette.go (pure, testable); this file is rendering +
// key handling. See docs/proposals/cli-discovery-ux.md §6.

// exploreModel is the Bubbletea model. The palette state is a pointer so cursor
// movement persists across Bubbletea's value-copied model.
type exploreModel struct {
	c         *Context
	appName   string
	ps        *paletteState
	chosen    *kong.Node // the leaf the user selected (nil if they quit)
	quit      bool
	filtering bool
}

func newExploreModel(c *Context, app *kong.Application) exploreModel {
	return exploreModel{c: c, appName: app.Name, ps: newPaletteState(app.Node)}
}

func (m exploreModel) Init() tea.Cmd { return nil }

func (m exploreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.filtering {
		switch k.Type {
		case tea.KeyEnter, tea.KeyEsc:
			m.filtering = false
		case tea.KeyBackspace:
			if n := len(m.ps.filter); n > 0 {
				m.ps.setFilter(m.ps.filter[:n-1])
			}
		case tea.KeySpace:
			m.ps.setFilter(m.ps.filter + " ")
		case tea.KeyRunes:
			m.ps.setFilter(m.ps.filter + string(k.Runes))
		}
		return m, nil
	}

	switch k.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.quit = true
		return m, tea.Quit
	case tea.KeyUp:
		m.ps.move(-1)
	case tea.KeyDown:
		m.ps.move(1)
	case tea.KeyLeft, tea.KeyBackspace:
		m.ps.back()
	case tea.KeyRight:
		m.ps.descend()
	case tea.KeyEnter:
		if sel, descended := m.ps.enter(); !descended && sel != nil {
			m.chosen = sel.node
			return m, tea.Quit
		}
	case tea.KeyRunes:
		switch string(k.Runes) {
		case "q":
			m.quit = true
			return m, tea.Quit
		case "j":
			m.ps.move(1)
		case "k":
			m.ps.move(-1)
		case "h":
			m.ps.back()
		case "l":
			m.ps.descend()
		case "/":
			m.filtering = true
		}
	}
	return m, nil
}

func (m exploreModel) View() string {
	c := m.c
	var b strings.Builder

	// Breadcrumb header.
	fmt.Fprintln(&b, c.th.title.render(c.Color, m.crumb()))

	// Grouped list with a cursor pointer; headers print when the group changes.
	prevGroup := ""
	for i, it := range m.ps.items {
		if it.group != prevGroup {
			fmt.Fprintln(&b, "  "+c.th.subtitle.render(c.Color, it.group))
			prevGroup = it.group
		}
		marker := "  "
		name := it.name
		if i == m.ps.cursor {
			marker = c.Accent(m.c.gl.Arrow) + " "
			name = c.th.title.render(c.Color, name)
		} else {
			name = c.Accent(name)
		}
		suffix := ""
		if it.parent {
			suffix = " " + c.Faint(m.c.gl.Arrow)
		}
		fmt.Fprintf(&b, "    %s%s%s\n", marker, name, suffix)
	}
	if len(m.ps.items) == 0 {
		fmt.Fprintln(&b, c.Faint("    (no matches)"))
	}

	// Filter line + footer keybar.
	fmt.Fprintln(&b)
	if m.filtering {
		fmt.Fprintf(&b, "  %s %s\n", c.Accent("filter:"), m.ps.filter+"_")
	}
	fmt.Fprintln(&b, c.Faint(footerKeys))

	// Bottom status bar — one line: what the focused command is + what it does.
	if sel := m.ps.selected(); sel != nil {
		fmt.Fprintln(&b, "  "+m.detailLine(sel))
	}
	return b.String()
}

const footerKeys = "  ↑↓ move · → open · ← back · / filter · ⏎ select · q quit"

// detailLine is the one-line status bar for the focused item: its full command
// path, a one-line summary of what it does, and a runnable/needs-args/group
// badge — truncated to the terminal width so it never wraps.
func (m exploreModel) detailLine(it *paletteItem) string {
	c := m.c
	path := m.crumb() + " " + it.name
	kind, label := badgeKind(it)

	// Fit the summary into what's left after the path, badge, and separators.
	budget := c.ruleWidth() - len(path) - (len(label) + 2) - 6
	summary := it.summary
	if r := []rune(summary); budget > 1 && len(r) > budget {
		summary = strings.TrimSpace(string(r[:budget-1])) + "…"
	}

	line := c.Accent(path)
	if summary != "" {
		line += "  " + c.Faint(summary)
	}
	return line + "  " + c.Badge(kind, label)
}

// badgeKind classifies the focused item for its status pill.
func badgeKind(it *paletteItem) (kind, label string) {
	switch {
	case it.parent:
		return "info", "group"
	case it.needsArg:
		return "warn", "needs args"
	default:
		return "ok", "runnable"
	}
}

// crumb is the breadcrumb path of the current navigation depth.
func (m exploreModel) crumb() string {
	parts := make([]string, len(m.ps.stack))
	for i, n := range m.ps.stack {
		if i == 0 {
			parts[i] = m.appName
		} else {
			parts[i] = n.Name
		}
	}
	return strings.Join(parts, " ")
}

// ExploreCmd is the reusable `explore` subcommand: an interactive palette over
// the tool's own command tree. Embed it in any CLI:
//
//	Explore clikit.ExploreCmd `cmd:"" help:"Browse commands interactively."`
//
// On a non-interactive stdout it falls back to the full styled help.
type ExploreCmd struct{}

// Run launches the palette. clikit.Run binds the *kong.Kong.
func (ExploreCmd) Run(c *Context, k *kong.Kong) error {
	if !interactiveTTY() {
		return emitHelp(os.Stdout, k.Model, nil, true)
	}
	final, err := tea.NewProgram(newExploreModel(c, k.Model), tea.WithAltScreen()).Run()
	if err != nil {
		return Fail(ExitRuntime, "EXPLORE_FAILED", err.Error(), "")
	}
	if fm, ok := final.(exploreModel); ok && fm.chosen != nil {
		return emitHelp(os.Stdout, k.Model, fm.chosen, true)
	}
	return nil
}

// interactiveTTY reports whether both stdin and stdout are terminals (Bubbletea
// needs a real TTY on both).
func interactiveTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
