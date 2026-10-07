# Story `1032` — saves written by older builds load again

## Result

The owner's `.ags` files under `implementation/saves/` load in the game where their byte-form version
is 50 or above. `cmd/savemigrate` prints every file's version and what upgrading it would lose, and
with `-w` rewrites a file at the current version keeping the original beside it as `.bak`.

The number someone can point at: **how many of those files load, printed by a committed command,
before and after this story.** The seat counted 65 `.ags` files on 2026-08-22 with `find saves -name
'*.ags' | wc -l` — 43 in `saves/` and 22 in `saves/rescued-20260822/`. Their version distribution is
not measured; B1 measures it.

## Why this story exists

`pkg/sim/binary.go:1978` refuses every byte form whose first byte is not `formatVersion`.
`formatVersion` is 57 and reached that value on 2026-08-22 at 15:23 in `dcdb3171`, with story `1029`;
version 56 arrived earlier the same day with `1025`. Every bump orphans every save already on disk,
because a fixed encoder does not help a file already written.

`cmd/saverepair`'s header records "five of his seventeen current-format saves" on 2026-08-22. That
commit is `6349af9e`, 00:59 the same day, when the current version was 56. Those seventeen files are
not current now. That is the standing cost this story removes.

**Owner directive, 2026-08-22**, three answers given in this session:

1. **An old save loads with explicit defaults, and the game says out loud what was substituted.** The
   load names each substituted section; a silent substitution is refused by this contract.
2. **Versions 50 and above are read.** Below 50 the reader refuses, and the refusal names the version.
3. **`cmd/savemigrate` reports by default and rewrites in place under `-w`, keeping a `.bak`**, the
   way `cmd/saverepair` already does.

Directive 1 reverses a decision `pkg/sim/binary.go`'s own header argues at length: that an absent
section is not a gap for a reader to fill but a claim about the simulation, so an older form is
refused rather than decoded into an invented world. That argument is not wrong and this story does not
delete it. Owner directive outweighs it, and the answer to the argument is B3: what would have been
invented silently is named to the player instead. **Update that header** rather than leaving it
asserting a policy the code no longer follows.

## What this build does today

Every line below is a premise to check in your own worktree, not a fact to build on.

- `World.UnmarshalBinary` refuses at `pkg/sim/binary.go:1978`, with the message `unknown byte form
  version %d, this build writes and reads %d only`.
- `pkg/sim/binary.go`'s header documents the version ladder, version by version, with the trade each
  bump made. It stops narrating around version 48; versions 49, 50, 56 and 57 are documented at their
  own sites instead. **The ladder from 50 to 57 is not written down in one place. Writing it down is
  part of B1.**
- The envelope is `pkg/game/save.go`: magic `AGRMSAVE`, a one-byte envelope version, a label length,
  the label, then the payload. **The envelope version is not the byte-form version** and has moved
  once. Establish which of the two refuses the owner's files before designing anything.
- `pkg/sim/binary_test.go` already builds older byte forms for its own peel tests — see
  `pinBytesPreMapUnitID` and the comment beginning `EVERY OLDER PEEL STARTS HERE`. Read that
  machinery before writing new fixtures.
- `cmd/saverepair` is the precedent for the tool: it reads the envelope through `pkg/game` and the
  world through `pkg/sim`, authors no layout of its own, reports by default and rewrites under `-w`.
- `cmd/savecheck` exercises the save/load loop against a real install in two separate processes. It is
  the natural place for the integration witness.

## The five behaviours

### B1 — the version census

A committed command prints, for each `.ags` file it is given: the envelope version, the byte-form
version, whether this build loads it, and what upgrading it would lose. Running it over a directory
prints per-version counts and a total.

This is first because every other number in this story quotes it. A count in `closure.md` is evidence
only if a committed command prints it.

### B2 — the reader accepts versions 50 through 57

`UnmarshalBinary` accepts the eight versions and refuses the rest. Each older version's absent
sections are supplied from **one declared per-version table**, not from branches spread through the
reader. A version below 50 is refused with a message naming the version, what it lacks, and that it
can be added.

The substituted value for each absent section is a decision this story records. Where the honest
default is "no such thing in this save" — an empty script state, an empty casting section — that is
what it is. Where a default would assert something false about the simulation, prefer the empty form
and disclose it under B3 rather than inventing a plausible number.

### B3 — the loss disclosure

The load names every section that was substituted, in the player's own words rather than the format's.
`AMBER`'s disclosure rule applies literally here: *POSITION only* was true of a file and read as
*your positions are carried*. "No script state" is a format sentence; what the player needs to know is
what it means for the mission he is about to resume.

The list is derived from the version, not maintained by hand beside it.

### B4 — upgrade is one road

Reading an old form and writing the current one is one function. The in-game load path and
`cmd/savemigrate` both call it, so a file the tool upgrades and a file the game loads cannot diverge.
A separate migration path inside the tool is the defect this behaviour exists to prevent.

### B5 — `cmd/savemigrate`

Report by default. `-w` rewrites in place, keeping the original as `.bak`, and refuses to touch a file
it cannot read whole. A file already at the current version is reported and left alone.

**The owner's own saves are not the lane's test corpus.** Copy what you need into your own scratch
directory and run `-w` there. Do not run this tool with `-w` against `implementation/saves/`,
`saves/rescued-20260822/`, or anything under `gameversions/`.

## What is deliberately not in this story

- **The original game's own `.sav` format.** `DIV-026` records what a genuine ROM1 save does and does
  not restore. Untouched here.
- **Making a save this build writes loadable by the original game.** That is the owner's requirement 2
  of 2026-08-12 and it is not this story.
- **The envelope version.** This story reads envelope version 1 only.
- **Versions below 50.**
- **A new byte-form version.** This story is not expected to bump `formatVersion`. If the work turns
  out to require a bump, that is a result to report in the return, not a step to take quietly: a bump
  orphans the same files again.

## Ceiling

**Four adversarial passes.** Five behaviours, and the byte form is hashed simulation state, which is
the condition the ceiling rule names. Reaching the ceiling is a scoping diagnosis, not a failure: land
what works, open the remainder as its own story, and say in `closure.md` that the story was cut too
large.

## Domains

Persistence, Sim Core, and Client for B3's notice. Three, which is the expected set; if you find
yourself in a fourth, say so rather than growing the story quietly.

## G2 — the limit this implies

The per-version table is the seam. A later version becomes one row rather than a new branch, and the
same table is what B1 prints and what B3 reads. Say in `spec.md` whether reading an old save can
change a shipped file's bytes: it cannot, and the only files this story rewrites are the owner's own
saves, under `-w`, with a `.bak` beside each one.

## Divergence rows this story is expected to owe

`DIV-254` through `DIV-258` are reserved for this story in `PIPELINE-STATUS.md`. Expect one row per
class of loss, and one row for the property that an upgraded save is indistinguishable from a save
taken natively at the current version, if that turns out to be true.

**If the range is spent, stop and ask.** Do not allocate a number from inside your worktree:
`pipeline/next-div-id.sh` scans the two ledgers and `PIPELINE-STATUS.md`, and an unmerged lane branch
is in none of them, so a number you pick can be handed to somebody else in the same hour.
