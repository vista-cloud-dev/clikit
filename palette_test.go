package clikit

import (
	"testing"

	"github.com/alecthomas/kong"
)

// testApp builds a small kong model: grouped leaves, a leaf that needs an arg,
// and an untagged parent command with its own children.
func testApp(t *testing.T) *kong.Application {
	t.Helper()
	var cli struct {
		Fmt  struct{} `cmd:"" group:"Author" help:"format"`
		Lint struct{} `cmd:"" group:"Author" help:"lint"`
		Test struct {
			Path string `arg:"" help:"path"`
		} `cmd:"" group:"Quality" help:"test"`
		Pkg struct {
			Parse struct{} `cmd:"" help:"parse"`
			Build struct{} `cmd:"" help:"build"`
		} `cmd:"" help:"pkg domain"`
	}
	k, err := kong.New(&cli, kong.Name("demo"))
	if err != nil {
		t.Fatal(err)
	}
	return k.Model
}

func paletteRoot(t *testing.T) *kong.Node { return testApp(t).Node }

func TestOrderedItems_GroupsOrderAndFlags(t *testing.T) {
	items := orderedItems(paletteRoot(t))
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.name
	}
	// Author(fmt,lint), Quality(test), then untagged Pkg -> "Commands"
	want := []string{"fmt", "lint", "test", "pkg"}
	if len(names) != len(want) {
		t.Fatalf("items = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("order = %v, want %v", names, want)
		}
	}
	byName := map[string]paletteItem{}
	for _, it := range items {
		byName[it.name] = it
	}
	if !byName["pkg"].parent {
		t.Error("pkg should be a parent (has children)")
	}
	if byName["fmt"].parent {
		t.Error("fmt should not be a parent")
	}
	if !byName["test"].needsArg {
		t.Error("test should be flagged needsArg (required positional)")
	}
	if byName["fmt"].needsArg {
		t.Error("fmt should not need args")
	}
	if byName["pkg"].group != ungroupedTitle {
		t.Errorf("pkg group = %q, want %q", byName["pkg"].group, ungroupedTitle)
	}
}

func TestPalette_MoveClamps(t *testing.T) {
	ps := newPaletteState(paletteRoot(t))
	if ps.cursor != 0 {
		t.Fatalf("cursor0 = %d", ps.cursor)
	}
	ps.move(-1)
	if ps.cursor != 0 {
		t.Error("up at top should clamp to 0")
	}
	ps.move(1)
	if ps.cursor != 1 {
		t.Errorf("down -> %d", ps.cursor)
	}
	for i := 0; i < 10; i++ {
		ps.move(1)
	}
	if ps.cursor != len(ps.items)-1 {
		t.Errorf("down past end -> %d, want %d", ps.cursor, len(ps.items)-1)
	}
}

func TestPalette_DescendAndBack(t *testing.T) {
	ps := newPaletteState(paletteRoot(t))
	// move cursor onto "pkg" (last item)
	for ps.selected() != nil && ps.selected().name != "pkg" {
		ps.move(1)
	}
	sel, descended := ps.enter()
	if !descended || sel != nil {
		t.Fatalf("enter on parent: descended=%v sel=%v", descended, sel)
	}
	if ps.current().Name != "pkg" {
		t.Fatalf("current = %q, want pkg", ps.current().Name)
	}
	got := []string{ps.items[0].name, ps.items[1].name}
	if got[0] != "parse" || got[1] != "build" {
		t.Fatalf("pkg children = %v", got)
	}
	if !ps.back() {
		t.Fatal("back from pkg should succeed")
	}
	if ps.current().Name == "pkg" {
		t.Fatal("back did not pop")
	}
	if ps.back() {
		t.Error("back at root should return false")
	}
}

func TestPalette_EnterLeafSelects(t *testing.T) {
	ps := newPaletteState(paletteRoot(t)) // cursor on fmt (leaf)
	sel, descended := ps.enter()
	if descended {
		t.Fatal("leaf enter should not descend")
	}
	if sel == nil || sel.name != "fmt" {
		t.Fatalf("selected = %v", sel)
	}
}

func TestPalette_Filter(t *testing.T) {
	ps := newPaletteState(paletteRoot(t))
	ps.setFilter("lint")
	if len(ps.items) != 1 || ps.items[0].name != "lint" {
		t.Fatalf("filtered = %v", ps.items)
	}
	ps.setFilter("")
	if len(ps.items) != 4 {
		t.Fatalf("cleared filter -> %d items", len(ps.items))
	}
}
