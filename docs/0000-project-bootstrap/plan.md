# Plan — project & repository bootstrap

## Intensity & terrain (declared before work)

- **Intensity: spec-anchored / static.** This story is the repository foundation every later
  story consumes (module tiers, the dependency wall, the license/provenance boundary, the asset
  guard, the build/test gates); it is durable and hard to reverse. Matches `PROFILE.md`
  ("Cross-cutting engine contracts … spec-anchored / static"). **static**, not living: no watcher
  tool exists in this repo, so the spec is kept true by discipline.
- **Terrain: greenfield.** No Go code exists yet; this story creates the skeleton. No behaviour to
  preserve, so the brownfield overlay does not apply.

## Approach

Stand up a compilable, dependency-free Go skeleton whose structure *is* the architecture: the
`againrom` module, one buildable package per tier under `pkg/`, entry points under `cmd/`, and a
stdlib-only import-graph check that makes the dependency DAG (FR-2) executable rather than
advisory. Around the code sit the distribution-boundary artifacts — the GPL licence and an
honest (currently empty) third-party notice set, a POSIX asset-guard that scans both the working
tree and full git history, a `.gitignore` boundary, and the architecture/provenance docs. The
asset root is reached only through a flag/env helper so no install path is ever compiled in. Every
package builds clean and every test is synthetic, so `go build/vet/test ./...` and `gofmt -l` all
pass on a machine with no game present.

## Facts verified during planning (baseline)

- Go **1.26.1** is installed (`go version go1.26.1 windows/amd64`); `go mod init againrom` emits a
  `go 1.26.1` directive and accepts it.
- The repository has an initial commit carrying `AGENTS.md`, `CLAUDE.MD`, `.gitignore`, and a
  workflow-artifact commit carrying `docs/0000-project-bootstrap/spec.md`. No Go module, `pkg/`,
  `cmd/`, `LICENSE`, `LICENSES/`, `THIRD_PARTY_NOTICES.md`, `scripts/`, `docs/ARCHITECTURE.md`, or
  `docs/PROVENANCE.md` exists yet.
- `.gitignore` currently excludes `.idea/` and `SDD/` (the SDD playbook stays out of the tracked
  repository); it carries no game-asset boundary yet.
- A POSIX shell is available (Git Bash 5.2) to author and run the guard.
- The canonical GPL-3.0 text is retrievable verbatim from the FSF (35149 bytes), so `LICENSE` need
  not be hand-transcribed.

## Files to touch

Module & tiers (FR-1):
- `go.mod` — **ADD** — `module againrom`, `go 1.26.1`.
- `pkg/formats/res/doc.go`, `pkg/formats/reg/doc.go`, `pkg/formats/alm/doc.go` — **ADD** — tier
  package docs; stdlib(+`x/text`)-only tier.
- `pkg/vfs/doc.go`, `pkg/data/doc.go`, `pkg/sim/doc.go`, `pkg/mapload/doc.go`, `pkg/render/doc.go`,
  `pkg/ui/doc.go`, `pkg/game/doc.go` — **ADD** — one buildable package per tier.
- `cmd/againrom/main.go`, `cmd/restool/main.go`, `cmd/regtool/main.go`, `cmd/almtool/main.go` —
  **ADD** — the game entry point and per-format tool entry points.

Dependency direction (FR-2):
- `internal/archtest/dag.go` — **ADD** — stdlib-only import-graph checker: a pure evaluator plus a
  module-tree loader (whole-module allow-map, fail-closed on unknown packages; `sim` stdlib-only).
- `internal/archtest/dag_test.go` — **ADD** — (a) loads the live tree and asserts it is clean, and
  (b) a table-driven test that feeds the pure evaluator synthetic violating edges — including
  `sim → formats/res` — and asserts it reports a violation **naming that edge** (AC-2's negative
  path, proven by a committed test rather than a transient manual injection).

Notices consistency (FR-3, AC-3 unit level):
- `internal/notices/notices_test.go` — **ADD** — stdlib-only unit test that parses `go.mod`'s
  `require` set and asserts `THIRD_PARTY_NOTICES.md` lists exactly those third-party modules (today:
  the empty set), failing on any stale or missing entry.

Config-driven asset root (FR-7):
- `pkg/game/assetroot.go` — **ADD** — `ResolveAssetRoot(flagValue, envValue string) string`
  precedence helper; no path literal.
- `pkg/game/assetroot_test.go` — **ADD** — synthetic precedence test (flag > env > empty).
- (`cmd/againrom/main.go`, above) wires an `-assets` flag and `AGAINROM_ASSETS` env into it.

Licence & notices (FR-3):
- `LICENSE` — **ADD** — GPL-3.0-or-later, canonical FSF text.
- `LICENSES/README.md` — **ADD** — convention for per-dependency licence texts (none yet).
- `THIRD_PARTY_NOTICES.md` — **ADD** — inventory (currently: no third-party compiled modules) +
  regeneration procedure.

Guard & boundary (FR-4, FR-5):
- `scripts/check-no-game-assets.sh` — **ADD** — POSIX sh; default scans tracked tree, `--history`
  scans the full object graph; non-zero exit naming any tracked game-asset path.
- `.gitignore` — **MODIFY** — add the game install, `examples/samples/`, and the curated game-asset
  extensions; retain `.idea/` and `SDD/`.

Docs (FR-8):
- `docs/ARCHITECTURE.md` — **ADD** — module tiers, the dependency DAG (matching the FR-2 check),
  the determinism wall, and the synthetic-test convention.
- `docs/PROVENANCE.md` — **ADD** — distribution boundary, game-first evidence hierarchy,
  clean-implementation procedure, own-data-only rebuild policy, the review-enforced no-third-party-
  source stance, and the separate research repo/submodule as the source of format facts.

## Risks (product)

- **R-1 — a hollow DAG check.** If the import test misclassifies module-internal imports as stdlib
  (the module path `againrom` has no dot, so a naive "no dot ⇒ stdlib" rule would skip every
  `againrom/…` edge), mis-maps directories to import paths, omits `cmd/`, or fails open on a package
  it does not recognise, a determinism-wall breach (e.g. `sim` importing a `formats` package) could
  ship green (threatens FR-2/P-3). *Mitigation:* the check classifies an import as intra-module by
  the `againrom/` path prefix (not by dots), covers `pkg/` and `cmd/`, is **fail-closed** on any
  unrecognised package, and is proven by injecting a deliberate violating import that must fail
  naming the edge (AC-2).
- **R-2 — notice drift.** As compiled dependencies are added by later stories, an unmaintained
  `THIRD_PARTY_NOTICES.md` would misstate the licence inventory (threatens FR-3). *Mitigation:* the
  zero-dependency baseline is exactly accurate now; an automated unit test cross-checks the notices
  against `go.mod`'s `require` set on every `go test` run, and the file documents the `go list -m
  all` regeneration step for when the module set changes.
- **R-3 — an asset type the guard misses.** A game-asset extension outside the blocklist could let
  game data into history (threatens P-1). *Mitigation:* the blocklist covers the known RoM/Allods
  asset extensions; `--history` scans the whole object graph; the guard is the backstop even when
  `.gitignore` is bypassed with `git add -f`. (Third-party source has no matchable signature and is
  handled by the provenance-review policy, not this guard.)
- **R-4 — a compiled-in install path.** A hardcoded GOG directory in shipped source would break
  config-driven asset location (threatens FR-7). *Mitigation:* the sole asset-root entry point is
  the flag/env helper, and a grep over shipped `pkg/` and `cmd/` for install-path fragments is part
  of the acceptance evidence.

## Success criteria

- **SC-1 (FR-1 → AC-1).** `go build ./...` and `go vet ./...` exit 0 on the skeleton, and
  `gofmt -l .` prints nothing (the project formatting gate). *Automated.*
- **SC-2 (FR-2 → AC-2, P-3).** `go test ./...` runs the `internal/archtest` tests green: the live
  tree is clean, and a committed table-driven test drives the evaluator with synthetic violating
  edges (e.g. `sim` → `formats/res`) and asserts it reports a violation naming the offending edge.
  Corroborated once by injecting a real violating import and observing `go test` go red. *Automated.*
- **SC-3 (FR-3 → AC-3).** `LICENSE`, `LICENSES/`, and `THIRD_PARTY_NOTICES.md` exist; the
  `internal/notices` unit test asserts the notices list exactly `go.mod`'s third-party `require` set
  (currently empty) with no stale or missing entries, corroborated by `go list -m all`. *Automated
  (unit) + review.*
- **SC-4 (FR-4 → AC-4, P-1).** `scripts/check-no-game-assets.sh` exits 0 on the clean tree; a
  force-staged fake `graphics.res` makes it exit non-zero naming the file; `--history` scans past
  commits. *Automated.*
- **SC-5 (FR-5 → AC-5).** With a game-install directory and extracted sample bytes on disk,
  `git status` reports none of them tracked. *Automated.*
- **SC-6 (FR-6 → AC-6, P-2).** `go test ./...` is green with no game install present; no test reads
  a game path. *Automated.*
- **SC-7 (FR-7 → AC-7).** A grep over shipped `pkg/` and `cmd/` finds no hardcoded install path; the
  asset root resolves from the `-assets` flag / `AGAINROM_ASSETS` env. *Review + automated.*
- **SC-8 (FR-8 → AC-8).** `docs/ARCHITECTURE.md` and `docs/PROVENANCE.md` exist; ARCHITECTURE's DAG
  matches the FR-2 allow-map; PROVENANCE states the distribution boundary, evidence hierarchy, and
  clean-implementation procedure. *Review.*

## Design decisions

- **D-1 — the import check is a stdlib `go/parser` AST walk in `internal/archtest`, split into a
  pure evaluator and a loader so its negative path is testable.** The **evaluator** is a pure
  function `Check(pkgImports map[string][]string, allow allowMap) []Violation` — given package →
  import-list data it returns the violating edges, each naming importer and imported package. The
  **loader** walks the module tree, maps each directory with `.go` files to its `againrom/…` import
  path, parses imports, and feeds the evaluator. **Import classification** is by path so the check
  cannot go hollow: an import is **intra-module** iff it equals `againrom` or has the prefix
  `againrom/` (the module path — *not* judged by dots, since `againrom` itself has no dot); otherwise
  **stdlib** iff its first path segment contains no dot, else **external**. The per-package allow-map
  lists the intra-module packages each may import (`vfs→{formats/res}`, `data→{vfs,formats/reg}`,
  `mapload→{formats/alm,data,sim}`, `render→{sim,vfs}`, `ui→{render}`, `game→any pkg/*`;
  `formats/*→{}`; `sim→{}`); external imports are rejected everywhere **except** `golang.org/x/text`
  under `formats/*`. Command edges are explicit: `cmd/againrom→any pkg/*`, `cmd/restool→{formats/res,
  vfs}`, `cmd/regtool→{formats/reg}`, `cmd/almtool→{formats/alm}`. **Fail-closed and whole-module:**
  the loader polices **every** `againrom/…` package **except** the `internal/` test helpers; any
  policed package absent from the allow-map is itself an error — so a package added later *anywhere*
  (not only under `pkg/`/`cmd/`) cannot silently escape the DAG. Acyclicity is guaranteed by
  construction (the allow-map is a strict layering), so no separate live-graph toposort is needed.
  For `pkg/sim` the stdlib-only assertion additionally covers its `_test.go` files, so the
  determinism package stays pure even in tests; other tiers' `_test.go` imports are intentionally
  not policed (test code commonly imports across tiers). *Known limitation (recorded, not deferred
  scope):* files carrying `//go:build` constraints are parsed regardless of tags; the skeleton has
  none, revisited when platform-tagged files first appear. *Rejected:* `golang.org/x/tools/go/
  packages` — pulls a dependency the skeleton otherwise does not use and would bloat the notices;
  *rejected:* shelling out to `go list -json` — depends on the go binary at test time and is less
  hermetic than parsing source. Placing it under `internal/` keeps it a non-tier helper the DAG
  rules do not police as an engine package.
- **D-2 — zero third-party dependencies in the skeleton (pure stdlib).** *Rejected:* pre-adding
  `golang.org/x/text` or Ebitengine — the spec and project rule say add only dependencies actually
  used; both are unused until the format and rendering stories, and adding them now would make the
  notices claim a dependency the build does not compile.
- **D-3 — each package is made buildable by a `doc.go` (libraries) or `main.go` (commands).**
  *Rejected:* empty directories or `.keep` files — Go cannot build an empty package, `go build
  ./...` would skip the tiers, and the DAG check would have no packages to anchor on.
- **D-4 — the determinism wall is set up as an import boundary only; behavioural determinism is
  deferred per the spec.** `pkg/sim` is a stdlib-only package with no `Step` implementation yet. The
  arch check enforces the **structural** half of the wall now — `pkg/sim` imports only stdlib and no
  other tier (exactly P-3) — which is all FR-2/P-3 require. It does **not** claim to enforce the
  **behavioural** half (no IO, no clock, no floats): those hazards can live in stdlib (`time`,
  `os`, `math/rand`) or need no import at all (float builtins), and the spec explicitly marks full
  determinism "intent, enforced later", proven on one entity in the walking-skeleton story.
  ARCHITECTURE states this split honestly rather than overclaiming. *Rejected:* adding a
  `Step(state, commands) → state` stub now — it edges into engine logic (out of scope) and an empty
  body invites a lint/vet smell. *Rejected:* a curated nondeterministic-stdlib denylist for `sim`
  now — it would enforce more than FR-2/P-3 ask (silent scope expansion) and, being a denylist,
  could not be complete anyway; the behavioural wall is the later story's job.
- **D-5 — asset root via `game.ResolveAssetRoot(flag, env)` with precedence flag > env > empty,
  consumed by an `-assets` flag and `AGAINROM_ASSETS` env in `cmd/againrom`.** *Rejected:* a
  hardcoded default install path — violates FR-7; *rejected:* a config-file-only source — heavier
  than needed and the spec names flag/config as the mechanism. Keeping the resolver in `pkg/game`
  makes it unit-testable synthetically and gives `cmd/againrom` a legal `→ pkg/game` edge for the
  DAG check to exercise.
- **D-6 — the guard is POSIX sh matching paths, not contents.** Tree mode scans `git ls-files`;
  `--history` scans `git rev-list --all --objects` (ref-reachable history — sufficient for P-1 on
  the tracked repo). Because `rev-list --objects` emits `<sha> <path>` (and bare `<sha>` for
  commit/tag objects), history mode first **extracts the path field** (`cut -d' ' -f2-`, which keeps
  paths containing spaces and yields empty for the bare-sha lines) *before* matching — otherwise the
  end-anchored extension pattern would test against the SHA and never fire. The resulting path list is
  **piped into `grep -iE`**, never iterated with a `for f in $(…)` loop, so paths with spaces do not
  word-split (avoiding a non-POSIX `read -d ''`).
  Matching is **case-insensitive and end-anchored** on the path — the enumerated game-asset-extension
  set is `\.(res|reg|alm|16a|256|spr|pal|fnt|fon|wav|snd|ogg|mp3|smk|bik|mpg|avi|exe|dll)$`
  (archives, registries, maps, sprites/palettes, game fonts, audio, video, executables). The guard
  keys **only** on game-asset paths: third-party *source* has no distinctive extension, so matching
  it would catch nothing or over-match — the "no third-party source" stance is a provenance policy
  enforced by review (`docs/PROVENANCE.md`), never claimed here as a mechanism. Every offender is
  printed before a single non-zero exit; a clean scan exits 0. Because matching is on tracked paths,
  `docs/PROVENANCE.md` listing these extensions in prose is never flagged, and an untracked local
  checkout — a game install, extracted bytes, or the `research/` submodule working tree — never trips
  the guard. *Rejected:* magic-byte/content sniffing —
  heavier and false-positive prone when path/extension already identifies assets, and it would flag
  no additional real case here; *rejected:* scanning untracked on-disk files — P-1 is about the
  tracked repo and history, and untracked local assets are the developer's, covered by `.gitignore`.
  *Residual (accepted):* a game asset renamed to a non-listed/absent extension is not caught by
  path matching; the blocklist is extended when such a case is identified.
- **D-7 — `THIRD_PARTY_NOTICES.md` states the honest empty set; `LICENSES/` holds a convention
  README; an `internal/notices` unit test keeps it honest.** The test (stdlib only) parses the
  `require` directives of `go.mod` and asserts the notices file lists exactly that third-party set
  (today: empty), failing on drift — satisfying AC-3's **unit** level rather than relying on manual
  review alone. *Rejected:* a `go-licenses`/build-graph generator — it needs an external tool and a
  network fetch of licence texts for a set that is currently empty, which is disproportionate; the
  documented `go list -m all` procedure regenerates the human-facing entries when a real dependency
  lands, and the unit test guards the invariant meanwhile.
- **D-8 — `go.mod` pins `go 1.26.1`.** *Rejected:* a floating `go 1.26` minor — the project's
  toolchain-parity rule pins the toolchain, and the patch-level pin is the honest reading of
  "pinned".
- **D-9 — `LICENSE` is the verbatim canonical FSF GPL-3.0 text.** *Rejected:* hand-transcribing or
  a markdown-converted copy — a licence must be byte-accurate.
