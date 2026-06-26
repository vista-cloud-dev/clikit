package clikit

import (
	"strings"

	"github.com/alecthomas/kong"
)

// This file is the pure navigation state for the interactive `explore` palette
// (Phase 2 of the discovery UX). It has no TUI dependency, so the whole
// navigation model — grouping, cursor movement, descend/ascend, filtering — is
// unit-testable. explore.go wraps it in a Bubbletea model. See
// docs/proposals/cli-discovery-ux.md §6.

// paletteItem is one selectable row: the command node plus the display facts the
// palette needs (its editorial group, whether it descends, whether it needs an
// argument before it can run).
type paletteItem struct {
	node     *kong.Node
	group    string
	name     string
	summary  string
	parent   bool // has subcommands -> Enter/Right descends
	needsArg bool // has a required positional -> not runnable as-is
}

// orderedItems flattens a node's command children into rows, grouped by their
// `group:""` tag in first-seen order with untagged commands in a trailing
// "Commands" bucket — the same ordering the styled help uses.
func orderedItems(node *kong.Node) []paletteItem {
	idx := map[string]int{}
	buckets := [][]paletteItem{}
	var ungrouped []paletteItem

	for _, ch := range node.Children {
		if ch.Type != kong.CommandNode || ch.Hidden {
			continue
		}
		it := paletteItem{
			node: ch, name: ch.Name, summary: ch.Help,
			parent: len(ch.Children) > 0, needsArg: hasRequiredArg(ch),
		}
		title := ""
		if ch.Group != nil {
			if title = ch.Group.Title; title == "" {
				title = ch.Group.Key
			}
		}
		if title == "" {
			it.group = ungroupedTitle
			ungrouped = append(ungrouped, it)
			continue
		}
		it.group = title
		i, seen := idx[title]
		if !seen {
			idx[title] = len(buckets)
			buckets = append(buckets, nil)
			i = len(buckets) - 1
		}
		buckets[i] = append(buckets[i], it)
	}

	var out []paletteItem
	for _, b := range buckets {
		out = append(out, b...)
	}
	return append(out, ungrouped...)
}

func hasRequiredArg(n *kong.Node) bool {
	for _, p := range n.Positional {
		if p.Required {
			return true
		}
	}
	return false
}

// paletteState tracks a breadcrumb stack of nodes and a cursor over the current
// node's (optionally filtered) items.
type paletteState struct {
	stack  []*kong.Node // stack[0] = root; last element = current node
	cursor int
	filter string
	items  []paletteItem // current node's items after filtering
}

func newPaletteState(root *kong.Node) *paletteState {
	ps := &paletteState{stack: []*kong.Node{root}}
	ps.rebuild()
	return ps
}

func (ps *paletteState) current() *kong.Node { return ps.stack[len(ps.stack)-1] }

// rebuild recomputes the visible items for the current node + filter, clamping
// the cursor into range.
func (ps *paletteState) rebuild() {
	items := orderedItems(ps.current())
	if ps.filter != "" {
		f := strings.ToLower(ps.filter)
		kept := items[:0:0]
		for _, it := range items {
			if strings.Contains(strings.ToLower(it.name), f) ||
				strings.Contains(strings.ToLower(it.summary), f) {
				kept = append(kept, it)
			}
		}
		items = kept
	}
	ps.items = items
	if ps.cursor >= len(items) {
		ps.cursor = max(0, len(items)-1)
	}
	if ps.cursor < 0 {
		ps.cursor = 0
	}
}

// selected returns the item under the cursor, or nil when the list is empty.
func (ps *paletteState) selected() *paletteItem {
	if ps.cursor < 0 || ps.cursor >= len(ps.items) {
		return nil
	}
	return &ps.items[ps.cursor]
}

// move shifts the cursor by delta, clamped to the list bounds.
func (ps *paletteState) move(delta int) {
	ps.cursor += delta
	if ps.cursor < 0 {
		ps.cursor = 0
	}
	if ps.cursor >= len(ps.items) {
		ps.cursor = max(0, len(ps.items)-1)
	}
}

// descend pushes into the selected item when it is a parent. Returns false if
// the selection isn't a descendable command.
func (ps *paletteState) descend() bool {
	sel := ps.selected()
	if sel == nil || !sel.parent {
		return false
	}
	ps.stack = append(ps.stack, sel.node)
	ps.cursor, ps.filter = 0, ""
	ps.rebuild()
	return true
}

// enter descends into a parent (returning descended=true) or returns the
// selected leaf for the caller to act on (descended=false).
func (ps *paletteState) enter() (*paletteItem, bool) {
	sel := ps.selected()
	if sel == nil {
		return nil, false
	}
	if sel.parent {
		ps.descend()
		return nil, true
	}
	return sel, false
}

// back pops one level. Returns false at the root.
func (ps *paletteState) back() bool {
	if len(ps.stack) <= 1 {
		return false
	}
	ps.stack = ps.stack[:len(ps.stack)-1]
	ps.cursor, ps.filter = 0, ""
	ps.rebuild()
	return true
}

// setFilter replaces the filter text and recomputes the visible items.
func (ps *paletteState) setFilter(s string) {
	ps.filter = s
	ps.cursor = 0
	ps.rebuild()
}
