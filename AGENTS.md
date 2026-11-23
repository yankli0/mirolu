# AGENTS.md — Mirolu Engineering & Agent Guidelines

## 1. Overview & Four Core Concepts

Mirolu is a single-binary personal infrastructure automation framework written in **Go 1.27.4** for a three-node Fedora 44/45 Linux fleet (`brisingamen`, `glaipnir`, `geko`).

Everything compiles declaratively from **four core concepts** in `internal/core`:
1. **Resource**: Atomic idempotent state unit (`Check -> Diff -> Apply`) executed by one of five drivers (`file`, `package`, `service`, `secret_target`, `command`), ordered via `Requires` and `Notifies` DAG edges.
2. **Module**: Purely declarative capability (`Name()`, `Build(ctx, Node, SecretLookup) ([]Resource, error)`) emitting `Resource` values.
3. **Node**: Target host (`Name`, `Role`, `User`, `Transport`, `Tags`, `Modules`, `Facts`).
4. **Secret**: `SecretRef` URI decoupled from backend storage, resolved into a redacted, zeroed `SecretValue`.

Do not invent top-level concepts (`Playbook`, `Role`, `Handler`, `Inventory`, custom DSLs). See [docs/plan/README.md](docs/plan/README.md) and [docs/plan/ARCHITECTURE.md](docs/plan/ARCHITECTURE.md) for the full design.

## 2. Package Boundaries & Forbidden Patterns

Import boundaries are enforced in `.golangci.yml` via `depguard` and `forbidigo`:
- `internal/core`: `$gostd` only.
- `internal/modules`: `internal/core` only; **no** `os/exec` or `database/sql`.
- `internal/ledger`, `internal/transport`, `internal/secret`, `internal/ui`: `internal/core` only.
- `internal/slots`: `internal/core` and `internal/modules` only.
- `internal/fleet`: `internal/core`, `internal/modules`, `internal/slots` only.
- `internal/driver`: `internal/core`, `internal/transport`, `internal/secret` only.
- `internal/engine`: `internal/core`, `internal/transport`, `internal/driver`, `internal/ledger` only.
- `cmd/mirolu`: `internal/engine`, `internal/fleet`, `internal/ui`, `internal/secret`, `internal/ledger` only.
- **Forbidden libraries/calls**: `gopkg.in/yaml.v3` (use `go.yaml.in/yaml/v3`), direct `github.com/getsops/sops` (wrap `sops` CLI), `charmbracelet/lipgloss` v1 (use `charm.land/lipgloss/v2`), CGO SQLite (`mattn/go-sqlite3`, `ncruces/go-sqlite3`; use `modernc.org/sqlite`), `io/ioutil`, `crypto/md5`, `http.DefaultClient`, `ssh.InsecureIgnoreHostKey`, raw `x/crypto/ssh/knownhosts` (use `github.com/skeema/knownhosts`), and `fmt.Print*` outside `cmd/mirolu` and `internal/ui`.

## 3. Working Cadence & Plan Discipline

- **Minimum Proof-of-Concept first**: Build only the minimal, essential functionality needed for the current sub-step. Do **not** over-engineer, add speculative fields/helpers/enums, or implement full-blown functionality prematurely.
- **Interactive pace — never rush a whole stage**: Work strictly **one small atomic sub-step at a time** following [docs/plan/STAGES.md](docs/plan/STAGES.md), and **pause for user confirmation/feedback** before advancing to the next sub-step.
- **Surface misalignments & taste decisions early**: Before implementing a sub-step, check if anything in the plan/docs looks obsolete, redundant, or misaligned, and pause to ask the user—alongside every taste checkpoint (`T-01`–`T-09`) or open decision slot (`DEC-01`–`DEC-05`) in [docs/plan/DECISIONS.md](docs/plan/DECISIONS.md). Never guess user preferences.
- Never advance from Stage N to Stage N+1 until every Definition of Done check for Stage N exits `0`.
- Defer all legacy config migration from `../dot_uchi/archive_v2` and `../dot_uchi/archive` strictly until **Stage 10**.
- **Branch per stage & keep `docs/` local for now**: develop each stage on a dedicated branch off `main` named `stages/<N>-<slug>` (e.g. `stages/0-scaffold`, `stages/1-core-dag`); never commit stage work directly to `main`. Keep `docs/` excluded from git commits for now (locally ignored via `.git/info/exclude`).

## 4. Toolchain & Verification Commands

Tool versions are pinned in `mise.toml` (`go 1.27.4`, `golangci-lint` v2, `gitleaks`).
- `make build`: static `CGO_ENABLED=0` binary build (`bin/mirolu`)
- `make build-hardened`: build with `GOEXPERIMENT=runtimesecret` and tag `runtimesecret`
- `make test`: `go test -race -count=1 ./...`
- `make test-integration`: Fedora 45 rootless Podman container tests (`-tags=integration`, Stage 4+)
- `make lint`: `golangci-lint run ./...`
- `make secret-scan`: `gitleaks` directory and git history scan
- `make verify`: runs `lint`, `test`, `build`, and `secret-scan`

## 5. Security & Fedora Guardrails

- **Never read, edit, print, or commit** `.env*`, `*.key`, `secrets/*.dec.*`, or `~/.ssh/**`.
- **Zero plaintext secret persistence**: never write secret values to SQLite (`ledger.db`), unified diffs, logs (`log/slog`), error strings, or systemd `Environment=`.
- **SELinux atomic writes**: write temp files in the target directory (never `mv` from `/tmp`) and run `restorecon -F`.
- **Fedora 44/45 rules**: check `/run/ostree-booted` before DNF, use strictly **DNF5** syntax, enforce Tailscale `>= 1.104.4`, use `loginctl enable-linger` + `XDG_RUNTIME_DIR`/`DBUS_SESSION_BUS_ADDRESS` for rootless user services (never `sudo systemctl --user`), and attach Quadlet containers to `proxy.network` with `:z`/`:Z` volume labels.
