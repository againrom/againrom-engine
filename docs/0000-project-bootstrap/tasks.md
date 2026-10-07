# Tasks — project & repository bootstrap

**Reading key.** `FR`/`AC`/`P` → `spec.md`; `SC-x`/`R-x`/`D-x` → `plan.md` (§Success criteria,
§Risks, §Design decisions). Each implementation task is exactly one implementation commit carrying
trailer `SDD-Task: 0000-project-bootstrap/T<n>`; the developer-run verification task produces
evidence, not a commit, and is outside the task↔commit map.

## Implementation tasks

### T1 — Module and buildable package skeleton  *(implementation)*
Create `go.mod` (`module againrom`, `go 1.26.1`) and one compilable package per tier: `pkg/formats/
{res,reg,alm}`, `pkg/vfs`, `pkg/data`, `pkg/sim`, `pkg/mapload`, `pkg/render`, `pkg/ui`, `pkg/game`
each via a `doc.go`; command entry points `cmd/againrom`, `cmd/restool`, `cmd/regtool`, `cmd/almtool`
each via a minimal `main.go`. `pkg/sim`'s doc states the stdlib-only determinism boundary (D-4).
Covers: FR-1, AC-1; realizes SC-1. Design: D-2, D-3, D-8.
**Done when:** `go build ./...` and `go vet ./...` exit 0, `gofmt -l .` prints nothing, and every
listed package/command directory contains a compilable Go file.

### T2 — Import-graph DAG enforcement  *(implementation)*
Add `internal/archtest/dag.go` (pure evaluator `Check(pkgImports, allow)` + module-tree loader,
whole-module allow-map, fail-closed on unregistered packages, `sim` stdlib-only incl. its tests) and
`internal/archtest/dag_test.go` ((a) the live tree is clean; (b) a table-driven test feeds synthetic
violating edges — including `sim → formats/res` — and asserts a violation is reported naming that
edge). Covers: FR-2, AC-2, P-3; realizes SC-2; mitigates R-1. Design: D-1.
**Done when:** `go test ./...` is green, and the committed table test fails the build if the
evaluator stops reporting/naming a violating edge (asserted on synthetic input, no game install).

### T3 — Config-driven asset root  *(implementation)*
Add `pkg/game/assetroot.go` with `ResolveAssetRoot(flagValue, envValue string) string` (precedence
flag > env > empty; no path literal) and a synthetic `pkg/game/assetroot_test.go`; wire `cmd/againrom/
main.go` to a `-assets` flag and `AGAINROM_ASSETS` env that it resolves through the helper and uses
(so no import/var is unused). Covers: FR-7, AC-7; realizes SC-7; mitigates R-4; contributes the
first synthetic unit test toward FR-6/AC-6. Design: D-5.
**Done when:** `go test ./...` is green; `go vet`/`go build ./...` clean; a grep over `pkg/` and
`cmd/` for install-path fragments (`GOG`, `Program Files`, `Allods`, a drive-letter path) returns
nothing; `cmd/againrom` prints/consumes the resolved root from flag/env only.

### T4 — No-game-assets guard  *(implementation)*
Add `scripts/check-no-game-assets.sh` (POSIX sh): default scans `git ls-files`; `--history` scans
`git rev-list --all --objects` with the path field extracted before matching; case-insensitive,
end-anchored game-asset-extension match on tracked paths only; prints each offender then
exits non-zero, else exits 0. Covers: FR-4, AC-4, P-1; realizes SC-4; mitigates R-3. Design: D-6.
**Done when:** on the clean tree the script exits 0; a force-staged fake `graphics.res` makes it exit
non-zero naming that file; a game asset present only in a past commit is named by `--history` but not
by the clean tree scan; the script is `sh`-clean (no bashisms).

### T5 — Gitignore boundary  *(implementation)*
Modify `.gitignore` to exclude the game install, `examples/samples/`, and the curated game-asset
extensions, while retaining `.idea/` and `SDD/`. Covers: FR-5, AC-5; realizes SC-5.
**Done when:** with a game-install directory and an `examples/samples/` file present on disk,
`git status --porcelain` lists none of them as tracked/untracked-for-add, and `git check-ignore`
confirms each boundary path is ignored.

### T6 — Licence and third-party notices  *(implementation)*
Add `LICENSE` (verbatim canonical GPL-3.0 text), `LICENSES/README.md` (per-dependency-text
convention), `THIRD_PARTY_NOTICES.md` (states the current empty third-party set and the `go list -m
all` regeneration procedure), and `internal/notices/notices_test.go` (stdlib-only unit test:
`go.mod` `require` set equals the notices' listed modules). Covers: FR-3, AC-3; realizes SC-3;
mitigates R-2. Design: D-7, D-9.
**Done when:** `LICENSE` matches the canonical GPL-3.0 text; `go test ./...` is green including the
notices test; the notices file and `go list -m all` agree that no third-party module is compiled in.

### T7 — Architecture and provenance documentation  *(implementation)*
Add `docs/ARCHITECTURE.md` (module tiers; the dependency DAG matching the `internal/archtest`
allow-map; the determinism wall's structural-now / behavioural-later split; the synthetic-only test
convention) and `docs/PROVENANCE.md` (distribution boundary; game-first evidence hierarchy;
clean-implementation procedure; own-data-only rebuild policy; the review-enforced
no-third-party-source stance; the research repo/submodule as the source of format facts).
Covers: FR-8, AC-8; documents FR-6's synthetic-test convention; realizes SC-8.
**Done when:** both files exist; ARCHITECTURE's DAG edges equal the T2 allow-map; PROVENANCE states
the distribution boundary, the evidence hierarchy, and the clean-implementation procedure.

## Developer-run verification task

### V1 — Acceptance evidence run  *(developer-run verification — evidence, not a commit)*
Run and record the acceptance evidence into `verification.md`: `go build ./...`, `go vet ./...`,
`go test ./...`, `gofmt -l .` (all with no game install); inject a real DAG-violating import and
observe `go test` go red naming the edge, then revert; run the asset guard against a force-staged
fake `graphics.res` in the tree and against a game asset present only in history via `--history`;
place a game install / `examples/samples/` bytes on disk and confirm `git status` leaves them
untracked;
grep shipped `pkg/`+`cmd/` for a hardcoded install path; compare `THIRD_PARTY_NOTICES.md` with
`go list -m all`. Covers evidence for AC-1…AC-8, P-1/P-2/P-3, SC-1…SC-8.
**Done when:** every applicable AC and success criterion has a recorded actual outcome or an explicit
limitation.

## Traceability

| Requirement (spec) | Property | Plan criterion | Task |
|---|---|---|---|
| FR-1 | — | SC-1 | T1 |
| FR-2 | P-3 | SC-2 | T2 |
| FR-3 | — | SC-3 | T6 |
| FR-4 | P-1 | SC-4 | T4 |
| FR-5 | — | SC-5 | T5 |
| FR-6 | P-2 | SC-6 | T3, T2, T6 (synthetic tests); T7 (convention); V1 (evidence) |
| FR-7 | — | SC-7 | T3 |
| FR-8 | — | SC-8 | T7 |
| AC-1 | — | SC-1 | T1 · V1 |
| AC-2 | P-3 | SC-2 | T2 · V1 |
| AC-3 | — | SC-3 | T6 · V1 |
| AC-4 | P-1 | SC-4 | T4 · V1 |
| AC-5 | — | SC-5 | T5 · V1 |
| AC-6 | P-2 | SC-6 | T2, T3, T6 · V1 |
| AC-7 | — | SC-7 | T3 · V1 |
| AC-8 | — | SC-8 | T7 · V1 |
