package clikit

import (
	"strings"
	"testing"
)

func TestPageThrough_DisabledWritesDirect(t *testing.T) {
	var b strings.Builder
	if err := pageThrough(&b, "hello world", false); err != nil {
		t.Fatal(err)
	}
	if b.String() != "hello world" {
		t.Errorf("got %q", b.String())
	}
}

func TestResolvePager_Default(t *testing.T) {
	t.Setenv("PAGER", "")
	name, args := resolvePager()
	if name != "less" {
		t.Errorf("name=%q", name)
	}
	if strings.Join(args, " ") != "-FRX" {
		t.Errorf("args=%v", args)
	}
}

func TestResolvePager_EnvOverride(t *testing.T) {
	t.Setenv("PAGER", "more -x")
	name, args := resolvePager()
	if name != "more" || len(args) != 1 || args[0] != "-x" {
		t.Errorf("name=%q args=%v", name, args)
	}
}
