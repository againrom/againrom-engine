# AGENTS.md — engine

This repository is the public game. This file is its whole rule set. The
brief that opens a task supplies anything task-specific: reserved IDs, extra
gates, and where the result is returned.

## Boundaries

- `knowledge/` is the pinned public snapshot of research and the authority on
  ROM1 behaviour this repository can cite. Read one claim with
  `cd knowledge && go run ./tools/claim <ID>` and cite the claim, not a
  remembered paraphrase. Againrom code and tests are never evidence about ROM1.
  `knowledge/` is read-only here; only its pin moves.
- The owner decides what Againrom does. A difference from ROM1 is one row in
  the matching subsystem file under `docs/divergences/`, chosen by the row's own
  Subsystem text; `docs/DIVERGENCES.md` is the index and row-format spec and
  takes no rows. `internal/divledger` and `cmd/divcensus` read
  `docs/divergences/*.md`.
- Never commit an install byte, extracted game asset, owner save, screenshot,
  generated render, original program code, or source whose licence does not
  admit it into a GPL-3.0-or-later work. A licensed open-source component
  enters only as `docs/PROVENANCE.md` admits it, with its notice and licence
  text. Lawful installs are read-only inputs. A tool that can write takes an
  explicit output path and never defaults into an install. The game's
  automatic player profile is the owner's exception: when the executable is
  inside an install, use `Againrom/` beside it, or the user configuration
  directory's `Againrom/` when that profile cannot be written. Only that
  selected profile may be written; installed files and explicitly selected
  install destinations remain fenced.
- One checkout has one committer. Preserve unrelated and pre-existing changes;
  never clean, reset or rewrite another checkout's work.

## Attribution

Nothing identifies the tool, model, vendor or session that produced a change:
not a commit message, tag, branch name, pull request, review report, code
comment or repository file. That excludes `Co-Authored-By` trailers naming an
assistant, session links such as `Claude-Session:`, "Generated with" lines,
model names and model IDs, and task trailers. Engine and knowledge commits
carry the project identity `Againrom <noreply@againrom.invalid>`; all other
commits carry the owner's Git identity. Story, experiment, claim and
divergence IDs are project identifiers and remain allowed. Naming the model or
effort an agent runs on in agent or tool configuration is configuration, not
attribution. `.claude/settings.json` switches off Claude Code's automatic
commit and pull-request attribution.

## Shape of the code

- `pkg/sim` owns deterministic game state and rules. `pkg/mapload` and other
  decoders turn installed data into typed values. `pkg/game` owns session,
  input, UI orchestration and persistence wiring. `pkg/render` draws; `cmd/*`
  are thin runnable tools and witnesses. Dependencies point inward: a renderer
  is never the authority for simulation state, and a loader never enacts
  gameplay policy.
- `pkg/game.FrontEnd` is the composition root and declares no field of its own.
  Every field belongs to one of five embedded components: `InstallResources`,
  `RuntimeServices`, `CampaignSession`, `Presentation`, `PersistenceContext`.
  `CampaignSession` is exactly what `resetSessionForNewGame` drops, and a test
  enforces that equality. `internal/archtest` ratchets the total field count and
  the number of non-test functions reaching three or more components through a
  FrontEnd value. Both may only fall; a fall rewrites
  `internal/archtest/composition_baseline.go` in the same commit
  (`go run ./internal/archtest/cmd/composition`).
- `sim.Command` stays a compact tagged union, but nothing outside
  `pkg/sim/command.go` writes one as a composite literal. Use the typed
  constructor for its kind (`MoveTo`, `Attack`, `CastAt`, `Equip`,
  `SetPlayerParameter` and the rest): pure field assignment, no validation,
  clamping or defaulting. A new command kind needs a constructor.
  `internal/archtest` holds production literals at zero and ratchets the test
  count in `internal/archtest/commandliteral_baseline.go`
  (`go run ./internal/archtest/cmd/commandliteral`).

## Naming and comments

- No identifier, file name or comment carries a story or experiment number.
  `internal/storyguard` enforces this for Go identifiers, `.go` file base names
  and comment text, and ratchets directory names (`docs/<NNNN>` exempt),
  non-`.go` file names, struct tags, story-shaped string literals and comment
  groups over 12 lines. Non-test identifier and `.go` file counts are zero;
  every other count is a falling-only baseline.
- A genuinely numeric identifier that only coincides with the range (a code
  page, a physical constant, a frozen gob field name) goes in `notAStoryNumber`
  in `internal/storyguard/scan.go` with its reason; `TestNotAStoryNumber` pins
  that list, so the same commit states what was added.
- A comment says what the code does and why. It is not a log of how or when that
  was decided, who reviewed it, or which story wrote it. A bare `DIV-` or claim
  ID may stand alone; a retired spec clause, calendar date, raw ROM1 address,
  `FUN_` name or `EXP-` citation never belongs in code. `internal/storyguard`
  ratchets each form and total comment bytes: a rise fails the build, a fall
  updates `internal/storyguard/baseline.go` in the same commit. Keep a comment
  group to 12 lines or fewer; the count of longer groups may only fall.
- A comment-only change proves itself with
  `scripts/check-comment-only-change.sh <base-ref>`, which entitles it to skip
  the release, scenario and milestone chains. A sweep that drives a counter to
  zero names `internal/storyguard/baseline.go` with `--allow-moved`.

## Compatibility surfaces

- Simulation and saves are compatibility surfaces. Never use wall time, global
  randomness, map iteration order, pointer identity or presentation-only state
  to decide hashed behaviour. A new persisted field needs a deterministic default
  for every old save this project has written. Reject corrupt input with a
  bounded error; never allocate from an unchecked installed count.
- An exported field name on a gob-persisted struct is a wire-format identifier.
  `encoding/gob` silently drops a stream field the receiver does not declare, so
  renaming the field reads old saves clean with the value gone. Renaming the
  type is free. Keep the name, or add a decode path for the historical one
  (`pkg/game/savehistoricalwire.go`) and prove it with a test that encodes under
  the historical name.

## Lane work

1. Reproduce the smallest failing behaviour and name its production route.
2. Add a focused regression test or an independently constructed witness.
3. Implement the complete player-facing slice, including persistence and UI
   only where that result reaches them.
4. While editing: `git diff --check`, `gofmt`, focused tests; `go build ./...`
   when command or public compile surfaces change; `go vet ./...` for
   concurrency, unsafe or broad refactors. Before each commit that changes Go
   code: its focused tests and the ordinary `go test` of every package it
   touches. On the final commit, once: `gofmt -l` clean,
   `go test -trimpath -count=1 ./...`, `scripts/check-no-game-assets.sh`, plus
   the gates the brief names.
   A tool you run to inspect behaviour from a `wt-*` worktree is built with
   `scripts/build-tools.sh <output-dir> [package...]`; a plain `go build`
   there stamps the enclosing repository's revision, not the worktree's.
5. For shipped-data, save or visual behaviour, name the EN and RU release
   witness that must run. Fixture-only tests never claim install validation.
6. Commit, push the exact branch SHA, and return base, candidate, changed files,
   evidence and remaining Unknowns.

Read `docs/HARNESS-CASEBOOK.md` only when a rule appears inapplicable, a review
repeats the same failure shape, or a new rule is proposed.

## Prose

Write documents in English and owner chat in Russian.

- Lead with the result. State facts directly.
- Use short sentences, concrete nouns, exact counts, paths, commits, and
  confidence levels.
- Separate observation, inference, owner direction, and Unknown. Do not turn a
  bounded search into a universal claim.
- Name the input and instrument beside a derived number. A conclusion may claim
  no more than the evidence excludes.
- Prefer one technical term throughout. Do not use synonyms to make repeated
  facts sound new.
- Do not narrate the order in which facts were discovered. Git and journals
  carry chronology.
- Do not repeat a conclusion for emphasis, dramatize certainty, address a
  document's reader personally, or add motivational prose.
- Use headings that name content. Use a table only for exact repeated fields and
  a list only when order or enumeration matters.
- A negative result says what was searched, over which population, and what was
  not found.
- A review finding names the player or hashed-state effect first, then the
  mechanism and evidence.
- Historical records remain frozen. Editing a living document does not require
  rewriting quoted or archived prose.

Before keeping a paragraph, ask: does it change a decision or preserve
evidence; is the same fact already in a canonical file; can a script print it
instead? If all three answers are no, delete it.
