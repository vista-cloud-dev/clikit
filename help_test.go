package clikit

import (
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func bufCtx(color bool) (*strings.Builder, *Context) {
	var b strings.Builder
	return &b, newRenderContext(&b, color)
}

func TestEmitHelp_LandingIsCompact(t *testing.T) {
	var b strings.Builder
	if err := emitHelp(&b, testApp(t), nil, false); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	// Compact intro: category names + pointers, but NOT every command.
	for _, want := range []string{"Author", "Quality", "menu", "demo help"} {
		if !strings.Contains(out, want) {
			t.Errorf("landing missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "fmt") || strings.Contains(out, "lint") {
		t.Errorf("landing should NOT list individual commands:\n%s", out)
	}
}

func TestWriteRootHelp_PlainNoANSI(t *testing.T) {
	b, c := bufCtx(false)
	groups := []helpGroup{
		{title: "Author", entries: []helpEntry{{"fmt", "format code"}, {"lint", "lint code"}}},
		{title: "Verify", entries: []helpEntry{{"test", "run tests"}}},
	}
	writeRootHelp(c, "m", "the M toolchain", groups, nil, true)
	out := b.String()
	if strings.Contains(out, "\x1b") {
		t.Fatalf("plain mode leaked ANSI: %q", out)
	}
	for _, want := range []string{"Author", "Verify", "fmt", "format code", "test"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Index(out, "Author") > strings.Index(out, "Verify") {
		t.Error("group order not preserved")
	}
}

func TestWriteRootHelp_ColorHasANSI(t *testing.T) {
	// lipgloss strips color when stdout isn't a TTY (as under `go test`); force a
	// color profile so we can assert our color path emits ANSI.
	lipgloss.SetColorProfile(termenv.TrueColor)
	b, c := bufCtx(true)
	writeRootHelp(c, "m", "x", []helpGroup{{title: "G", entries: []helpEntry{{"a", "b"}}}}, nil, true)
	if !strings.Contains(b.String(), "\x1b") {
		t.Error("color mode produced no ANSI")
	}
}

func TestWriteRootHelp_LandingVsFull(t *testing.T) {
	groups := []helpGroup{{title: "G", entries: []helpEntry{{"a", "b"}}}}
	globals := []SchemaFlag{{Name: "output", Help: "format"}}

	bf, cf := bufCtx(false)
	writeRootHelp(cf, "m", "x", groups, globals, true)
	bl, cl := bufCtx(false)
	writeRootHelp(cl, "m", "x", groups, globals, false)

	if !strings.Contains(bf.String(), "output") {
		t.Error("full help should list global flags")
	}
	if strings.Contains(bl.String(), "output") {
		t.Error("landing should omit global flags")
	}
	if !strings.Contains(bl.String(), "--help") {
		t.Error("landing should hint at --help")
	}
}

func TestWriteCommandHelp_Plain(t *testing.T) {
	b, c := bufCtx(false)
	writeCommandHelp(c, "m fmt [flags] <file>", "Format M code.", "",
		[]SchemaArg{{Name: "file", Help: "file to format"}},
		[]SchemaFlag{{Name: "write", Help: "write in place"}}, "")
	out := b.String()
	for _, want := range []string{"Usage:", "m fmt", "Format M code.", "file", "write"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Example") {
		t.Error("no example given, but an Example section was rendered")
	}
	if strings.Contains(out, "\x1b") {
		t.Error("ANSI leak")
	}
}

func TestWriteCommandHelp_Example(t *testing.T) {
	b, c := bufCtx(false)
	writeCommandHelp(c, "v rpc-debug tail [flags]", "Stream RPCs.", "", nil, nil,
		"v rpc-debug tail --container vehu")
	out := b.String()
	for _, want := range []string{"Example", "v rpc-debug tail --container vehu"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// The example:"" tag survives onto the command node and emitHelp renders it.
func TestEmitHelp_ExampleFromTag(t *testing.T) {
	var cli struct {
		Tail struct {
			Container string `help:"container"`
		} `cmd:"" help:"stream" example:"demo tail --container vehu"`
	}
	k, err := kong.New(&cli, kong.Name("demo"))
	if err != nil {
		t.Fatal(err)
	}
	var node *kong.Node
	for _, n := range k.Model.Node.Children {
		if n.Name == "tail" {
			node = n
		}
	}
	if node == nil {
		t.Fatal("tail node not found")
	}
	var b strings.Builder
	if err := emitHelp(&b, k.Model, node, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "demo tail --container vehu") {
		t.Errorf("example not rendered:\n%s", b.String())
	}
}

func TestGroupsFrom_BucketsByTagAndOrder(t *testing.T) {
	var cli struct {
		Fmt  struct{} `cmd:"" group:"Author" help:"format"`
		Lint struct{} `cmd:"" group:"Author" help:"lint"`
		Test struct{} `cmd:"" group:"Verify" help:"test"`
		Misc struct{} `cmd:"" help:"untagged"`
	}
	k, err := kong.New(&cli)
	if err != nil {
		t.Fatal(err)
	}
	groups := groupsFrom(k.Model.Node)
	if len(groups) != 3 {
		t.Fatalf("want 3 groups, got %d: %+v", len(groups), groups)
	}
	if groups[0].title != "Author" || len(groups[0].entries) != 2 {
		t.Errorf("group0 = %+v", groups[0])
	}
	if groups[0].entries[0].name != "fmt" || groups[0].entries[1].name != "lint" {
		t.Errorf("Author entries/order wrong: %+v", groups[0].entries)
	}
	last := groups[len(groups)-1]
	if last.title != ungroupedTitle || len(last.entries) == 0 || last.entries[0].name != "misc" {
		t.Errorf("untagged bucket should be last: %+v", last)
	}
}
