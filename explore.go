package clikit

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Explore-palette colors (ANSI / xterm-256 indices so they map to the terminal's
// own palette): bold-white category names, white command names, and a bold
// light-blue cursor highlight (whatever the cursor is on). Detail summary is
// yellow; the [status] tag is green=runnable, gray=group, blue=needs-args.
var (
	expCat   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))  // bold white categories
	expCmd   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))             // white commands
	expSel   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")) // cursor: bold light blue
	expInfo  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))             // yellow detail
	expRun   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))             // runnable: green
	expGroup = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))              // group/info: gray
	expNeeds = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))             // not runnable: blue
)

// paint renders s with style only when color is on.
func paint(color bool, st lipgloss.Style, s string) string {
	if !color {
		return s
	}
	return st.Render(s)
}

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
		m.ps.moveUp()
	case tea.KeyDown:
		m.ps.moveDown()
	case tea.KeyLeft:
		m.ps.moveLeft()
	case tea.KeyRight:
		m.ps.moveRight()
	case tea.KeyBackspace:
		m.ps.back()
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
		case "k":
			m.ps.moveUp()
		case "j":
			m.ps.moveDown()
		case "h":
			m.ps.moveLeft()
		case "l":
			m.ps.moveRight()
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

	// One row per category: a bold-white name (cursor column 0), then its commands
	// (columns 1..N, white). The cursor — bold light blue with a ▸ pointer — can
	// land on the category name or any command; a trailing → marks a command that
	// descends. ←→ move within a row, ↑↓ between rows.
	labelW := 0
	for _, cat := range m.ps.cats {
		if len(cat.name) > labelW {
			labelW = len(cat.name)
		}
	}
	for r, cat := range m.ps.cats {
		prefix, nameStyle := "  ", expCat
		if r == m.ps.row && m.ps.col == 0 {
			prefix, nameStyle = paint(c.Color, expSel, c.gl.Title)+" ", expSel
		}
		pad := strings.Repeat(" ", labelW-len(cat.name))
		var cmds []string
		for i, it := range cat.items {
			name := it.name
			if it.parent {
				name += c.gl.Arrow
			}
			if r == m.ps.row && m.ps.col == i+1 {
				cmds = append(cmds, paint(c.Color, expSel, c.gl.Title+name))
			} else {
				cmds = append(cmds, paint(c.Color, expCmd, name))
			}
		}
		fmt.Fprintf(&b, "%s%s%s   %s\n", prefix, paint(c.Color, nameStyle, cat.name), pad, strings.Join(cmds, "  "))
	}
	if len(m.ps.cats) == 0 {
		fmt.Fprintln(&b, c.Faint("  (no matches)"))
	}

	// Bottom: a blank, optional filter line, the one-line detail (command path +
	// what it does, or — on a category name — what the category is), then the
	// keybar.
	fmt.Fprintln(&b)
	if m.filtering {
		fmt.Fprintf(&b, "  %s %s\n", c.Accent("filter:"), m.ps.filter+"_")
	}
	if it := m.ps.selectedItem(); it != nil {
		fmt.Fprintln(&b, "  "+m.detailLine(it))
	} else if cat := m.ps.selectedCat(); cat != nil {
		fmt.Fprintln(&b, "  "+m.catLine(cat))
	}
	fmt.Fprintln(&b, c.Faint(footerKeys))
	return b.String()
}

const footerKeys = "  ←↑↓→ move · ⏎ open/run · ⌫ back · / filter · q quit"

// catLine is the one-line status bar when the cursor is on a category name: the
// name, what the category is, and its command count.
func (m exploreModel) catLine(cat *paletteCat) string {
	c := m.c
	unit := "commands"
	if len(cat.items) == 1 {
		unit = "command"
	}
	return paint(c.Color, expCat, cat.name) + "  " +
		paint(c.Color, expInfo, cat.desc) + "  " +
		paint(c.Color, expGroup, fmt.Sprintf("[%d %s]", len(cat.items), unit))
}

// detailLine is the one-line status bar for the focused item: its full command
// path (green), a one-line summary of what it does (yellow), and a bracketed
// [status] tag (green runnable / blue needs-args / gray group) — all on one
// line, truncated to the terminal width so it never wraps.
func (m exploreModel) detailLine(it *paletteItem) string {
	c := m.c
	path := m.crumb() + " " + it.name
	label, st := badgeFor(it)

	// Fit the summary into what's left after the path, "[label]", and separators.
	budget := c.ruleWidth() - len(path) - (len(label) + 2) - 6
	summary := it.summary
	if r := []rune(summary); budget > 1 && len(r) > budget {
		summary = strings.TrimSpace(string(r[:budget-1])) + "…"
	}

	line := paint(c.Color, expCmd, path)
	if summary != "" {
		line += "  " + paint(c.Color, expInfo, summary)
	}
	return line + "  " + paint(c.Color, st, "["+label+"]")
}

// badgeFor returns the focused item's status label and color: runnable (green),
// needs-args (blue), or group (gray).
func badgeFor(it *paletteItem) (label string, st lipgloss.Style) {
	switch {
	case it.parent:
		return "group", expGroup
	case it.needsArg:
		return "needs args", expNeeds
	default:
		return "runnable", expRun
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
