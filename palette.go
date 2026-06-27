package clikit

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
)

// This file is the pure navigation state for the interactive `explore` palette
// (Phase 2). It has no TUI dependency, so the whole 2-D navigation model —
// categories, items, cursor movement across both, descend/ascend, filtering — is
// unit-testable. explore.go wraps it in a Bubbletea model. See
// docs/proposals/cli-discovery-ux.md §6.
//
// Layout is one row per category: a category NAME (cursor column 0) followed by
// its commands (columns 1..N). The cursor is a (row, col) pair: ←→ move within a
// row across [name, cmd1, cmd2, …]; ↑↓ move between category rows. Landing on a
// category name (col 0) shows what the category is; landing on a command shows
// what that command does.

// ungroupedTitle (the "Commands" bucket for untagged commands) is declared in
// help.go and shared here.

// paletteItem is one command cell.
type paletteItem struct {
	node     *kong.Node
	name     string
	summary  string
	parent   bool // has subcommands -> Enter descends
	needsArg bool // has a required positional -> not runnable as-is
}

// paletteCat is one category row: a name, a description (what the category is),
// and its command cells.
type paletteCat struct {
	name  string
	desc  string
	items []paletteItem
}

// catBlurb describes the editorial categories the toolchain ships, so a category
// name is meaningful on focus. A Kong group Description (via kong.ExplicitGroups)
// overrides it; an unknown category falls back to a command count.
var catBlurb = map[string]string{
	"Author":          "Write and check M source.",
	"Quality":         "Test, cover, and check the code.",
	"Engine":          "Reach a live engine.",
	"Sync":            "Move routines and builds between server and disk.",
	"Introspect":      "Inspect the tool itself.",
	"Domains":         "VistA subsystem tool domains.",
	"Scaffold":        "Generate new tool / domain skeletons.",
	"Inspect":         "Examine a build without changing it.",
	"Transform":       "Decompose, assemble, and canonicalize builds.",
	"Build & install": "Build and install on a live engine.",
	"Back-out":        "Snapshot and reverse an install.",
	ungroupedTitle:    "Other commands.",
}

func catDescription(name, kongDesc string, n int) string {
	switch {
	case kongDesc != "":
		return kongDesc
	case catBlurb[name] != "":
		return catBlurb[name]
	case n == 1:
		return "1 command"
	default:
		return fmt.Sprintf("%d commands", n)
	}
}

func hasRequiredArg(n *kong.Node) bool {
	for _, p := range n.Positional {
		if p.Required {
			return true
		}
	}
	return false
}

// paletteCats groups a node's command children into ordered categories (by group
// tag, first-seen order; untagged -> trailing "Commands"), each with a
// description.
func paletteCats(node *kong.Node) []paletteCat {
	idx := map[string]int{}
	var cats []paletteCat
	var ungrouped []paletteItem

	for _, ch := range node.Children {
		if ch.Type != kong.CommandNode || ch.Hidden {
			continue
		}
		it := paletteItem{
			node: ch, name: ch.Name, summary: ch.Help,
			parent: len(ch.Children) > 0, needsArg: hasRequiredArg(ch),
		}
		title, desc := "", ""
		if ch.Group != nil {
			if title = ch.Group.Title; title == "" {
				title = ch.Group.Key
			}
			desc = ch.Group.Description
		}
		if title == "" {
			ungrouped = append(ungrouped, it)
			continue
		}
		i, seen := idx[title]
		if !seen {
			idx[title] = len(cats)
			cats = append(cats, paletteCat{name: title, desc: desc})
			i = len(cats) - 1
		}
		cats[i].items = append(cats[i].items, it)
	}
	if len(ungrouped) > 0 {
		cats = append(cats, paletteCat{name: ungroupedTitle, items: ungrouped})
	}
	for i := range cats {
		cats[i].desc = catDescription(cats[i].name, cats[i].desc, len(cats[i].items))
	}
	return cats
}

// paletteState tracks a breadcrumb stack of nodes and a 2-D cursor (row over
// categories, col over [name, items…]) on the current node, plus a filter.
type paletteState struct {
	stack  []*kong.Node // stack[0] = root; last = current
	cats   []paletteCat
	row    int
	col    int // 0 = category name; 1..len(items) = item (col-1)
	filter string
}

func newPaletteState(root *kong.Node) *paletteState {
	ps := &paletteState{stack: []*kong.Node{root}}
	ps.rebuild()
	ps.gotoFirstItem()
	return ps
}

func (ps *paletteState) current() *kong.Node { return ps.stack[len(ps.stack)-1] }

// rebuild recomputes categories for the current node + filter, then clamps the
// cursor.
func (ps *paletteState) rebuild() {
	cats := paletteCats(ps.current())
	if ps.filter != "" {
		f := strings.ToLower(ps.filter)
		var kept []paletteCat
		for _, c := range cats {
			var items []paletteItem
			for _, it := range c.items {
				if strings.Contains(strings.ToLower(it.name), f) ||
					strings.Contains(strings.ToLower(it.summary), f) {
					items = append(items, it)
				}
			}
			if len(items) > 0 {
				c.items = items
				kept = append(kept, c)
			}
		}
		cats = kept
	}
	ps.cats = cats
	ps.clamp()
}

func (ps *paletteState) maxCol() int {
	if len(ps.cats) == 0 {
		return 0
	}
	return len(ps.cats[ps.row].items)
}

func (ps *paletteState) clamp() {
	if ps.row < 0 {
		ps.row = 0
	}
	if ps.row >= len(ps.cats) {
		ps.row = max(0, len(ps.cats)-1)
	}
	if ps.col < 0 {
		ps.col = 0
	}
	if mc := ps.maxCol(); ps.col > mc {
		ps.col = mc
	}
}

// gotoFirstItem parks the cursor on the first command (or the first category
// name if the first category is empty).
func (ps *paletteState) gotoFirstItem() {
	ps.row = 0
	if len(ps.cats) > 0 && len(ps.cats[0].items) > 0 {
		ps.col = 1
	} else {
		ps.col = 0
	}
}

func (ps *paletteState) moveUp() {
	if ps.row > 0 {
		ps.row--
		ps.clamp()
	}
}

func (ps *paletteState) moveDown() {
	if ps.row < len(ps.cats)-1 {
		ps.row++
		ps.clamp()
	}
}

func (ps *paletteState) moveLeft() {
	if ps.col > 0 {
		ps.col--
	}
}

func (ps *paletteState) moveRight() {
	if ps.col < ps.maxCol() {
		ps.col++
	}
}

// onCategory reports whether the cursor is on a category name (col 0).
func (ps *paletteState) onCategory() bool { return ps.col == 0 }

// selectedCat returns the category under the cursor (nil if the list is empty).
func (ps *paletteState) selectedCat() *paletteCat {
	if len(ps.cats) == 0 {
		return nil
	}
	return &ps.cats[ps.row]
}

// selectedItem returns the command under the cursor, or nil when on a category
// name or when the list is empty.
func (ps *paletteState) selectedItem() *paletteItem {
	if len(ps.cats) == 0 || ps.col == 0 {
		return nil
	}
	c := &ps.cats[ps.row]
	if ps.col-1 >= len(c.items) {
		return nil
	}
	return &c.items[ps.col-1]
}

// descend pushes into the selected command when it is a parent.
func (ps *paletteState) descend() bool {
	it := ps.selectedItem()
	if it == nil || !it.parent {
		return false
	}
	ps.stack = append(ps.stack, it.node)
	ps.filter = ""
	ps.rebuild()
	ps.gotoFirstItem()
	return true
}

// enter descends into a parent command (descended=true) or returns the selected
// leaf (descended=false). On a category name it is a no-op (nil, false).
func (ps *paletteState) enter() (*paletteItem, bool) {
	it := ps.selectedItem()
	if it == nil {
		return nil, false
	}
	if it.parent {
		ps.descend()
		return nil, true
	}
	return it, false
}

// back pops one level. Returns false at the root.
func (ps *paletteState) back() bool {
	if len(ps.stack) <= 1 {
		return false
	}
	ps.stack = ps.stack[:len(ps.stack)-1]
	ps.filter = ""
	ps.rebuild()
	ps.gotoFirstItem()
	return true
}

// setFilter replaces the filter text and recomputes the visible categories.
func (ps *paletteState) setFilter(s string) {
	ps.filter = s
	ps.rebuild()
	ps.gotoFirstItem()
}
