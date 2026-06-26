# clikit

The shared CLI convention layer for the **vista-cloud-dev** Go toolchain — a small,
standalone, importable module every `m-*` and `v-*` command-line tool builds on so
they all share:

- one command grammar (Kong) and `Run(name, desc, cli, *Globals)` entry point;
- one `--output text|json|auto` contract with a versioned JSON envelope;
- one deterministic error object + the exit-code ladder
  (`0` ok · `1` runtime · `2` usage · `3` check/drift · `4` refused);
- `schema` (agent-discovery) and `version` reflection;
- a TTY-gated styling layer (lipgloss) and shell completions (kongplete).

## Why a standalone module

clikit originated as a vendored, byte-identical package inside `go-cli-template`
(and copied into m-cli, v-pkg, m-dev-tools-mcp, …). The `v` umbrella CLI composes
multiple domains **in one process**, and Go requires those domains to share the
**same** `clikit.Context` type — impossible while each repo vendors its own copy.
Extracting clikit into this module (2026-06-25) gives every consumer one type, and
is the prerequisite for mounting a second `v` domain. See
[`v-cli/docs/v-cli-platform.md`](https://github.com/vista-cloud-dev/v-cli/blob/main/docs/v-cli-platform.md) §6.

The v-family (`v-cli`, `v-pkg`) consumes this module first; the m-family tools
migrate off their vendored copies as they are next touched.

## Use

```go
import "github.com/vista-cloud-dev/clikit"

func main() {
	cli := &CLI{}
	os.Exit(clikit.Run("mytool", "one-line description", cli, &cli.Globals))
}
```

## Versioning

Tagged releases (SemVer). Consumers pin a version in `go.mod` (no `replace`
directives) — the same serialize-the-contract discipline the rest of the org uses.
