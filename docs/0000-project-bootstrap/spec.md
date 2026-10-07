# Spec — project & repository bootstrap

**Provenance basis.** Pure own-project setup — no game data, no third-party reverse-engineering. This
story stands up the clean, independently-redistributable repository skeleton that every later story
builds on: the Go module and package tiers, the license/provenance boundary, the "no game assets"
guard, and the build/test gates. It establishes the own-data-only rebuild policy in
`docs/PROVENANCE.md` (a deliverable of this story, FR-8); the format/engine stories (0001+) sit on
top.

## Problem / goal

A from-scratch clean repo must be reproducible from our own specs + a user's lawful game install alone,
and must never contain game data or third-party source. Before any format is parsed, the repository
needs: a module and package layout that enforces the architecture's dependency rules, a license and
provenance policy, an automated guard that keeps game assets out of history (third-party source is
excluded by the provenance policy, enforced in review), and green build/test tooling on an empty
skeleton.

## What this establishes (all our own)

- **Module & tiers.** Go module `againrom`. Packages live under `pkg/`, executables under `cmd/`. The
  one-directional dependency DAG defined by this spec (and documented, not sourced, in
  `docs/ARCHITECTURE.md` per FR-8): `pkg/formats/* → stdlib`; `pkg/vfs → pkg/formats/res`; `pkg/data →
  pkg/vfs, pkg/formats/reg`; `pkg/sim → stdlib only`; `pkg/mapload → pkg/formats/alm, pkg/data,
  pkg/sim`; `pkg/render → pkg/sim (read-only), pkg/vfs`; `pkg/ui → pkg/render`; `pkg/game →
  everything`. `cmd/` holds the game (`cmd/againrom`) and per-format dump/view tools.
- **Determinism wall (intent, enforced later).** `sim` advances only via `Step(state, commands) →
  state` on a fixed integer tick — no IO, no clock, no floats. The wall is set up as a boundary here and
  proven on one entity in the walking-skeleton story; it is untestable to retrofit, so the boundary
  holds from day one.
- **License & notices.** GPL-3.0-or-later (`LICENSE`); `THIRD_PARTY_NOTICES.md` inventorying only the
  compiled open-source dependencies (Ebitengine et al.) with their licenses under `LICENSES/`.
- **Provenance policy.** `docs/PROVENANCE.md` (produced per FR-8): the distribution boundary (no game
  archives, maps, registries, sprites, fonts, audio, video, executables, disassembly, or converted
  assets in the repo or releases), the game-first evidence hierarchy, and the clean-implementation
  procedure. Per-story clean specs live under `docs/<NNNN>-<slug>/` (this story is
  `docs/0000-project-bootstrap/`).
- **Asset location from config, never hardcoded.** The game path is engine config / a CLI flag; no
  source hardcodes the GOG directory.

## Functional requirements

- **FR-1 (module & layout)** — `go.mod` declares module `againrom`; the package directories exist per the
  layout above; `go build ./...` and `go vet ./...` are clean on the skeleton.
- **FR-2 (dependency direction)** — the import graph matches the DAG (no `sim` import of `formats`/
  `data`/`render`/`ui`/`game`; `formats/*` import only stdlib + `x/text`). Enforced by an automatable
  check (an AST/import test) so a violating import fails CI, not review.
- **FR-3 (license & notices present & accurate)** — `LICENSE` (GPL-3.0-or-later), `LICENSES/*` and
  `THIRD_PARTY_NOTICES.md` exist; the notices list exactly the modules in `go.mod`/`go.sum` with correct
  licenses; a review step regenerates them when the module set changes.
- **FR-4 (no-game-assets guard)** — `scripts/check-no-game-assets.sh` scans the working tree, and with
  `--history` the full git history, and **fails** if any game archive/map/registry/sprite/font/
  audio/video/executable is tracked. Runs in CI.
- **FR-5 (.gitignore boundary)** — `.gitignore` excludes the game install and any extracted asset
  bytes (`examples/samples/` extracted content) — which must never enter git history or releases
  even when present on a contributor's disk.
- **FR-6 (synthetic-only tests)** — the test convention is that `go test ./...` runs green **without a
  game installation**: tests build synthetic byte streams from the documented format contracts and
  never read GOG files. Developer-run corpus verification is separate (research tools under `cmd/`, not
  the test suite).
- **FR-7 (config-driven asset path)** — the asset root comes from engine config / a CLI flag; a repo
  grep finds no hardcoded install path in shipped code.
- **FR-8 (architecture & provenance docs authored)** — implementation of this story **produces**
  `docs/ARCHITECTURE.md` (documenting the module tiers and the dependency DAG this spec defines) and
  `docs/PROVENANCE.md` (the distribution boundary, the game-first evidence hierarchy, and the
  clean-implementation procedure). These are **outputs**, not prerequisites: nothing in this spec
  depends on reading them. `ARCHITECTURE.md` MUST agree with the import check (FR-2) — the check, not
  the prose, is the DAG's enforcement.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | build | the skeleton repo | `go build ./... && go vet ./...` | clean, no errors |
| AC-2 | unit | an added import that violates the DAG (e.g. `sim` importing a `formats` pkg) | the import check runs | it fails, naming the offending edge |
| AC-3 | unit | `go.mod`/`go.sum` and `THIRD_PARTY_NOTICES.md` | notices verified | every dependency is listed with the correct license; no stale/missing entries |
| AC-4 | tooling | a planted fake game asset (e.g. a bogus `graphics.res`) staged in the tree | `scripts/check-no-game-assets.sh` | non-zero exit naming the file; `--history` also scans past commits |
| AC-5 | tooling | `.gitignore` + a local game install / extracted bytes on disk | `git status` | none of the game data / extracted samples are tracked |
| AC-6 | test | the full suite | `go test ./...` with **no** game install present | green (all tests synthetic) |
| AC-7 | review | shipped source | grep for the GOG install path | no hardcoded asset path; the root is config/flag-driven |
| AC-8 | review | the skeleton repo | `docs/ARCHITECTURE.md` and `docs/PROVENANCE.md` are reviewed | both exist; ARCHITECTURE.md's DAG matches the FR-2 import check (AC-2); PROVENANCE.md states the distribution boundary, evidence hierarchy, and clean-implementation procedure |

## Derived properties

- **P-1** The tracked repository, at any commit, contains no game data (FR-4 `--history` is the
  invariant's enforcement). The exclusion of third-party source is a provenance policy enforced by
  review (`docs/PROVENANCE.md`), not by the asset guard, which keys only on game-asset paths.
- **P-2** `go test ./...` result is independent of whether a game install is present (FR-6).
- **P-3** The dependency DAG defined by this spec is acyclic; `pkg/sim` imports only stdlib. The FR-2
  import check is the enforcement (`docs/ARCHITECTURE.md` documents the DAG but is not its authority).

## Gate check

Every FR is covered: FR-1→AC-1, FR-2→AC-2 (+P-3), FR-3→AC-3, FR-4→AC-4 (+P-1), FR-5→AC-5, FR-6→AC-6
(+P-2), FR-7→AC-7, FR-8→AC-8. The spec is self-contained: every file it names
(`docs/ARCHITECTURE.md`, `docs/PROVENANCE.md`, `LICENSE`, `THIRD_PARTY_NOTICES.md`,
`scripts/check-no-game-assets.sh`, `.gitignore`, `go.mod`) is an **output** of this story, none a
prerequisite to read.

## Out of scope

- Any format parsing (starts at 0001, the `.res` archive) or engine logic.
- The full SDD process playbook content (`SDD/WORKFLOW.md`, `PROFILE.md`) beyond referencing it.
- CI provider specifics (the gates are `go build`/`vet`/`test`, `gofmt`, the import check, the asset
  guard; wiring them to a particular CI service is deployment detail).
- RoM2 support (a future format-independence goal, not new scaffolding here).
</content>
