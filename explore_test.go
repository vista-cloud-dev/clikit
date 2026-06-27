package clikit

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestModel(t *testing.T) exploreModel {
	t.Helper()
	_, c := bufCtx(false)
	return newExploreModel(c, testApp(t))
}

func keyT(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }
func runes(s string) tea.KeyMsg     { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func upd(m tea.Model, k tea.KeyMsg) (exploreModel, tea.Cmd) {
	m2, cmd := m.Update(k)
	return m2.(exploreModel), cmd
}

func TestExplore_2DMoves(t *testing.T) {
	m := newTestModel(t) // (0,1) fmt
	m, _ = upd(m, keyT(tea.KeyRight))
	if m.ps.col != 2 {
		t.Errorf("right -> col %d", m.ps.col)
	}
	m, _ = upd(m, keyT(tea.KeyLeft))
	if m.ps.col != 1 {
		t.Errorf("left -> col %d", m.ps.col)
	}
	m, _ = upd(m, keyT(tea.KeyDown))
	if m.ps.row != 1 {
		t.Errorf("down -> row %d", m.ps.row)
	}
	m, _ = upd(m, runes("k")) // up
	if m.ps.row != 0 {
		t.Errorf("k(up) -> row %d", m.ps.row)
	}
}

func TestExplore_FilterFlow(t *testing.T) {
	m := newTestModel(t)
	m, _ = upd(m, runes("/"))
	if !m.filtering {
		t.Fatal("expected filtering after /")
	}
	for _, r := range "lint" {
		m, _ = upd(m, runes(string(r)))
	}
	if len(m.ps.cats) != 1 || m.ps.cats[0].items[0].name != "lint" {
		t.Fatalf("filtered cats = %+v", m.ps.cats)
	}
	m, _ = upd(m, keyT(tea.KeyEnter))
	if m.filtering {
		t.Error("enter should exit filter mode")
	}
}

func TestExplore_EnterLeafChoosesAndQuits(t *testing.T) {
	m := newTestModel(t) // on fmt (leaf)
	m, cmd := upd(m, keyT(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("selecting a leaf should return a quit cmd")
	}
	if m.chosen == nil || m.chosen.Name != "fmt" {
		t.Fatalf("chosen = %v", m.chosen)
	}
}

func TestExplore_EnterParentDescends(t *testing.T) {
	m := newTestModel(t)
	m, _ = upd(m, keyT(tea.KeyDown))
	m, _ = upd(m, keyT(tea.KeyDown)) // onto pkg
	m, cmd := upd(m, keyT(tea.KeyEnter))
	if cmd != nil || m.chosen != nil {
		t.Fatal("descending into a parent should not choose/quit")
	}
	if m.ps.current().Name != "pkg" {
		t.Fatalf("did not descend; current = %q", m.ps.current().Name)
	}
}

func TestExplore_ViewShowsCategoriesCommandsFooter(t *testing.T) {
	out := newTestModel(t).View()
	// Category names render UPPERCASE in the palette rows.
	for _, want := range []string{"AUTHOR", "QUALITY", "fmt", "pkg", "move", "quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("View missing %q in:\n%s", want, out)
		}
	}
}

func TestExplore_CommandDetailLine(t *testing.T) {
	m := newTestModel(t) // on fmt
	var line string
	for _, ln := range strings.Split(m.View(), "\n") {
		if strings.Contains(ln, "demo fmt") {
			line = ln
			break
		}
	}
	if line == "" || !strings.Contains(line, "format") || !strings.Contains(line, "runnable") {
		t.Errorf("command detail line wrong: %q", line)
	}
}

func TestExplore_CategoryInfoLine(t *testing.T) {
	m := newTestModel(t)
	m, _ = upd(m, keyT(tea.KeyLeft)) // fmt -> Author category name
	if !m.ps.onCategory() {
		t.Fatal("expected cursor on category")
	}
	out := m.View()
	if !strings.Contains(out, "Write and check M source") {
		t.Errorf("category info (description) not shown:\n%s", out)
	}
}

func TestExplore_QuitNoSelection(t *testing.T) {
	m := newTestModel(t)
	m, cmd := upd(m, runes("q"))
	if cmd == nil || !m.quit {
		t.Fatal("q should quit")
	}
	if m.chosen != nil {
		t.Error("quit should leave nothing chosen")
	}
}
