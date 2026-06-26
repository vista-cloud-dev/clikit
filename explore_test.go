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

func key(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }
func runes(s string) tea.KeyMsg    { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }
func upd(m tea.Model, k tea.KeyMsg) (exploreModel, tea.Cmd) {
	m2, cmd := m.Update(k)
	return m2.(exploreModel), cmd
}

func TestExplore_DownMovesCursor(t *testing.T) {
	m := newTestModel(t)
	if m.ps.cursor != 0 {
		t.Fatal("start cursor != 0")
	}
	m, _ = upd(m, key(tea.KeyDown))
	if m.ps.cursor != 1 {
		t.Errorf("after down, cursor = %d", m.ps.cursor)
	}
	m, _ = upd(m, runes("k"))
	if m.ps.cursor != 0 {
		t.Errorf("after k (up), cursor = %d", m.ps.cursor)
	}
}

func TestExplore_FilterFlow(t *testing.T) {
	m := newTestModel(t)
	m, _ = upd(m, runes("/"))
	if !m.filtering {
		t.Fatal("expected filtering mode after /")
	}
	for _, r := range "lint" {
		m, _ = upd(m, runes(string(r)))
	}
	if len(m.ps.items) != 1 || m.ps.items[0].name != "lint" {
		t.Fatalf("filtered items = %v", m.ps.items)
	}
	m, _ = upd(m, key(tea.KeyEnter)) // exit filter, keep results
	if m.filtering {
		t.Error("enter should exit filter mode")
	}
}

func TestExplore_EnterLeafChoosesAndQuits(t *testing.T) {
	m := newTestModel(t) // cursor on fmt (leaf)
	m, cmd := upd(m, key(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("selecting a leaf should return a quit cmd")
	}
	if m.chosen == nil || m.chosen.Name != "fmt" {
		t.Fatalf("chosen = %v", m.chosen)
	}
}

func TestExplore_RightDescendsIntoParent(t *testing.T) {
	m := newTestModel(t)
	for m.ps.selected() != nil && m.ps.selected().name != "pkg" {
		m, _ = upd(m, key(tea.KeyDown))
	}
	m, _ = upd(m, key(tea.KeyRight))
	if m.ps.current().Name != "pkg" {
		t.Fatalf("right did not descend; current = %q", m.ps.current().Name)
	}
}

func TestExplore_ViewShowsGroupsAndFooter(t *testing.T) {
	m := newTestModel(t)
	out := m.View()
	for _, want := range []string{"Author", "Quality", "fmt", "pkg", "filter", "quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("View missing %q in:\n%s", want, out)
		}
	}
}

func TestExplore_DetailLineIsOneLineWithPathAndSummary(t *testing.T) {
	m := newTestModel(t) // app "demo", cursor on fmt (help "format")
	var line string
	for _, ln := range strings.Split(m.View(), "\n") {
		if strings.Contains(ln, "demo fmt") {
			line = ln
			break
		}
	}
	if line == "" {
		t.Fatalf("no bottom detail line with the command path; View:\n%s", m.View())
	}
	// path AND summary must be on the SAME single line.
	if !strings.Contains(line, "format") {
		t.Errorf("detail line missing summary: %q", line)
	}
	if !strings.Contains(line, "runnable") {
		t.Errorf("detail line missing badge: %q", line)
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
