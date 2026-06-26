package clikit

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// This file is the optional pager: long human output (help, listings) is piped
// through $PAGER on an interactive terminal so it scrolls, and written straight
// through otherwise. Color is baked into the content before paging (the pager's
// destination is the real TTY even though our pipe is not), so styled output
// survives the pipe.

// pagerEnabled reports whether interactive paging should be used: stdout is a
// TTY and paging is not disabled by the caller, the CLIKIT_NO_PAGER override, or
// a PAGER set to the empty disabling values "cat"/"".
func pagerEnabled(noPager bool) bool {
	if noPager || os.Getenv("CLIKIT_NO_PAGER") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// resolvePager returns the pager command and its arguments: $PAGER when set
// (split on whitespace), otherwise `less -FRX` (quit if it fits on one screen,
// keep ANSI color, leave output on screen).
func resolvePager() (string, []string) {
	if p := strings.TrimSpace(os.Getenv("PAGER")); p != "" {
		fields := strings.Fields(p)
		return fields[0], fields[1:]
	}
	return "less", []string{"-FRX"}
}

// pageThrough writes content to w directly when paging is disabled, or pipes it
// through the resolved pager otherwise. If the pager can't be started it falls
// back to a direct write, so output is never lost.
func pageThrough(w io.Writer, content string, enabled bool) error {
	if !enabled {
		_, err := io.WriteString(w, content)
		return err
	}
	name, args := resolvePager()
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_, werr := io.WriteString(w, content)
		return werr
	}
	return nil
}
