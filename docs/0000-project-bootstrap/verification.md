# Verification — project & repository bootstrap

Reproducible evidence for the acceptance criteria and derived properties of
`spec.md`, realized through the plan's success criteria (`SC-1…SC-8`).

## Environment

- Toolchain: `go1.26.1 windows/amd64` (the version pinned in `go.mod`).
- Shell for the guard/git checks: Git Bash (GNU bash 5.2, POSIX `sh`).
- Observation window: 2026-07-22. No game installation was present on the
  machine during any run.

## Results by acceptance criterion

### AC-1 — build & vet clean (SC-1) · Unit/build
`go build ./...` → exit 0; `go vet ./...` → exit 0; `gofmt -l .` → empty output.
All 14 tier/command packages compile.

### AC-2 — DAG violation fails naming the edge (SC-2, P-3) · Unit
- Committed synthetic test: `internal/archtest` `TestCheckNamesOffendingEdge`
  drives the pure evaluator with violating graphs; the `sim → formats/res` case
  yields `pkg/sim -> pkg/formats/res: import violates the dependency DAG`, and the
  fail-closed / external / allowed cases behave as specified. `go test` green.
- Live corroboration: temporarily adding `import _ "againrom/pkg/formats/res"` to
  a `pkg/sim` file made `go test ./internal/archtest/` fail with
  `unexpected DAG violation: pkg/sim -> pkg/formats/res …`; removing it returned
  the package to `ok`.

### AC-3 — notices accurate, no stale/missing (SC-3) · Unit + review
`go list -m all` prints only `againrom` (the main module) — no third-party
compiled modules — matching `THIRD_PARTY_NOTICES.md` ("None"). The
`internal/notices` unit test asserts the notices table equals the `go.mod`
`require` set; it is green now, and injecting a phantom module row made it fail
with `… listed in THIRD_PARTY_NOTICES.md but not required in go.mod (stale)`,
confirming it is not hollow. `LICENSE` is the verbatim canonical GPL-3.0 text
(35149 bytes); `LICENSES/README.md` documents the per-dependency-text convention.

### AC-4 — no-game-assets guard (SC-4, P-1) · Tooling
`scripts/check-no-game-assets.sh` run under `sh`:
- Clean working tree → `clean (tree scan)`, exit 0.
- A force-staged `graphics.res` → exit 1, printing `  graphics.res`.
- In a throwaway repo where a `graphics.res` exists only in a past commit, the
  default tree scan is clean while `--history` reports `graphics.res` (exit 1),
  confirming history-mode path extraction works.
- `sh -n` reports no syntax errors (POSIX-clean; no bashisms).

### AC-5 — .gitignore boundary (SC-5) · Tooling
With a `game/` install directory and `examples/samples/` bytes on disk,
`git status --porcelain` listed neither, and `git check-ignore` confirmed both
`game/` and `examples/samples/` are ignored. `.idea/` and `SDD/` remain ignored;
no third-party/clone patterns are present.

### AC-6 — synthetic suite green with no game (SC-6, P-2) · Test
`go test ./...` is green with no game installation present. The only test-bearing
packages — `internal/archtest`, `internal/notices`, `pkg/game` — build their
inputs in code and read no game path, so the result is independent of any install
(P-2).

### AC-7 — no hardcoded asset path; config/flag-driven (SC-7) · Review + unit
A grep over shipped `pkg/` and `cmd/` for install-path fragments (`GOG`,
`Program Files`, `Allods`, a drive path) returns nothing. `cmd/againrom` sources
the root only from the `-assets` flag and the `AGAINROM_ASSETS` env: with the flag
set it reported the flag value; with only the env set it reported the env value;
with neither it reported `no asset root configured`. `game.ResolveAssetRoot`'s
precedence (flag > env > empty) is covered by a synthetic unit test. (Under Git
Bash the displayed absolute paths were MSYS-rewritten — a shell artifact; the
source of the value, not its spelling, is what these runs demonstrate.)

### AC-8 — architecture & provenance docs (SC-8) · Review
`docs/ARCHITECTURE.md` and `docs/PROVENANCE.md` both exist. The set of packages
and their permitted edges in ARCHITECTURE's DAG table is identical to the
`internal/archtest` allow-map (same 14 packages; e.g. `pkg/sim` → stdlib only,
`pkg/vfs` → `pkg/formats/res`, `pkg/game`/`cmd/againrom` → any `pkg/*`).
PROVENANCE states the distribution boundary (game assets enforced by the guard;
third-party source excluded by review policy), the game-first evidence hierarchy,
the clean-implementation procedure, the own-data-only rebuild policy, and the
separate research repository/submodule as the source of format facts.

## Derived properties

- **P-1** (no game data at any tracked commit): enforced by AC-4's `--history`
  mode; evidenced above.
- **P-2** (test result independent of a game install): evidenced by AC-6 — the
  suite is synthetic and green with no install.
- **P-3** (DAG acyclic; `pkg/sim` stdlib-only): the allow-map is a strict
  layering (acyclic by construction) and the `sim` stdlib-only rule is asserted by
  the check, including over `sim`'s test files.

## Limitations & residual risk

- The asset guard matches by path/extension on tracked content: a game asset
  renamed to a non-listed or absent extension is not detected, and untracked or
  submodule-resident bytes are intentionally out of scope (they are the
  developer's, covered by `.gitignore`). The blocklist is extended if such a case
  is found.
- The determinism wall is enforced structurally (imports only) now; behavioural
  determinism (no IO/clock/float at runtime) is intent to be proven on an entity
  in a later story, per the spec — it is not claimed here.
- `--history` scans ref-reachable objects; objects only in an unreferenced
  dangling state are out of scope, which is adequate for the tracked-repo
  invariant.

## Conclusion

Every applicable acceptance criterion (AC-1…AC-8) and derived property
(P-1…P-3) has reproducible passing evidence with no game installation present.
The skeleton builds, vets, and tests clean; the dependency DAG, the game-asset
guard, the licence/notice inventory, the `.gitignore` boundary, the
config-driven asset root, and the architecture/provenance docs meet the contract
as scoped. Confidence is bounded to what a near-empty skeleton can demonstrate:
the DAG check is exercised by one live edge plus synthetic violations, and the
notice inventory is trivially empty but guarded against future drift.
