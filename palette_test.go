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

func TestPaletteCats_OrderItemsAndDesc(t *testing.T) {
	cats := paletteCats(paletteRoot(t))
	if len(cats) != 3 {
		t.Fatalf("want 3 cats, got %d: %+v", len(cats), cats)
	}
	if cats[0].name != "Author" || cats[1].name != "Quality" || cats[2].name != ungroupedTitle {
		t.Fatalf("cat order = %q,%q,%q", cats[0].name, cats[1].name, cats[2].name)
	}
	if len(cats[0].items) != 2 || cats[0].items[0].name != "fmt" || cats[0].items[1].name != "lint" {
		t.Errorf("Author items = %+v", cats[0].items)
	}
	if !cats[2].items[0].parent {
		t.Error("pkg should be a parent")
	}
	if !cats[1].items[0].needsArg {
		t.Error("test should need args")
	}
	if cats[0].desc == "" || cats[2].desc == "" {
		t.Error("categories should have descriptions")
	}
}

func TestPalette_InitialCursorOnFirstCommand(t *testing.T) {
	ps := newPaletteState(paletteRoot(t))
	if ps.row != 0 || ps.col != 1 {
		t.Fatalf("initial cursor = (%d,%d), want (0,1)", ps.row, ps.col)
	}
	if it := ps.selectedItem(); it == nil || it.name != "fmt" {
		t.Fatalf("selectedItem = %v", it)
	}
}

func TestPalette_2DMovementAndClamp(t *testing.T) {
	ps := newPaletteState(paletteRoot(t)) // (0,1) fmt
	ps.moveRight()                        // (0,2) lint
	if it := ps.selectedItem(); it == nil || it.name != "lint" {
		t.Fatalf("after right: %v", ps.selectedItem())
	}
	ps.moveRight() // clamp at maxCol=2
	if ps.col != 2 {
		t.Errorf("right past end col=%d", ps.col)
	}
	ps.moveLeft() // fmt
	ps.moveLeft() // category name (col 0)
	if !ps.onCategory() || ps.selectedItem() != nil {
		t.Fatalf("expected on category; col=%d item=%v", ps.col, ps.selectedItem())
	}
	if cat := ps.selectedCat(); cat == nil || cat.name != "Author" || cat.desc == "" {
		t.Fatalf("selectedCat = %+v", ps.selectedCat())
	}
	ps.moveLeft() // clamp at 0
	if ps.col != 0 {
		t.Errorf("left past start col=%d", ps.col)
	}
	// from (0,2) lint, moving down clamps col into Quality (maxCol 1)
	ps.row, ps.col = 0, 2
	ps.moveDown()
	if ps.row != 1 || ps.col != 1 {
		t.Errorf("down+clamp = (%d,%d), want (1,1)", ps.row, ps.col)
	}
	for i := 0; i < 5; i++ {
		ps.moveDown()
	}
	if ps.row != len(ps.cats)-1 {
		t.Errorf("down past end row=%d", ps.row)
	}
}

func TestPalette_DescendAndBack(t *testing.T) {
	ps := newPaletteState(paletteRoot(t))
	// move onto pkg (last category, first command)
	ps.moveDown()
	ps.moveDown()
	if it := ps.selectedItem(); it == nil || it.name != "pkg" {
		t.Fatalf("expected pkg, got %v", ps.selectedItem())
	}
	if _, descended := ps.enter(); !descended {
		t.Fatal("enter on pkg should descend")
	}
	if ps.current().Name != "pkg" {
		t.Fatalf("current = %q", ps.current().Name)
	}
	if it := ps.selectedItem(); it == nil || it.name != "parse" {
		t.Fatalf("after descend selectedItem = %v", ps.selectedItem())
	}
	if !ps.back() {
		t.Fatal("back should succeed")
	}
	if ps.current().Name == "pkg" {
		t.Fatal("back did not pop")
	}
	if ps.back() {
		t.Error("back at root should be false")
	}
}

func TestPalette_EnterLeafReturnsIt(t *testing.T) {
	ps := newPaletteState(paletteRoot(t)) // on fmt (leaf)
	it, descended := ps.enter()
	if descended || it == nil || it.name != "fmt" {
		t.Fatalf("enter leaf: it=%v descended=%v", it, descended)
	}
}

func TestPalette_EnterOnCategoryIsNoop(t *testing.T) {
	ps := newPaletteState(paletteRoot(t))
	ps.col = 0 // on the Author category name
	it, descended := ps.enter()
	if it != nil || descended {
		t.Errorf("enter on category should be a no-op; it=%v descended=%v", it, descended)
	}
}

func TestPalette_Filter(t *testing.T) {
	ps := newPaletteState(paletteRoot(t))
	ps.setFilter("lint")
	if len(ps.cats) != 1 || len(ps.cats[0].items) != 1 || ps.cats[0].items[0].name != "lint" {
		t.Fatalf("filtered cats = %+v", ps.cats)
	}
	ps.setFilter("")
	if len(ps.cats) != 3 {
		t.Fatalf("cleared filter -> %d cats", len(ps.cats))
	}
}
