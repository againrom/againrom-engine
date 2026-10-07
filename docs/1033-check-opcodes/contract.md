# Story `1033` — the three check opcodes the campaign authors and this build cannot evaluate

## Result

Eleven of the twenty-eight shipped campaign missions can finish their scripts. Check opcodes 4, 16
and 21 evaluate instead of poisoning their registers, and every trigger that reads one stops being
marked `Inert` before a tick runs.

The number someone can point at: **the milestone census falls to zero for these three arms.**
`pipeline/check-milestone.sh` prints it. `DIV-239` records 10 nodes of opcode 4, 7 of opcode 16 and 16
of opcode 21 — 33 in all, on each root, over maps 40, 60, 71, 81, 90, 91, 100, 101, 131, 140 and 150.
**Measure it yourself before you start and after you finish, and quote both numbers.** The count in
this paragraph was written on 2026-08-22 and a census number is a measurement, not a constant.

## Why this story exists

`scriptCheckSupported` in `pkg/sim/script.go` lists none of the three. Each reaches
`ScriptSilenceUnsupported`, is named in `Script.Unsupported`, and `NewScript` poisons its register, so
every trigger reading it is `Inert` and never fires. That is `DIV-239`, FIDELITY-DEBT, OPEN.

The row was UNKNOWN while no claim gave the three arms. `EXP-0215` gave all three at the 2026-08-22
pin bump, which this worktree carries, so the row is a known gap with a decode behind it.

## What is decoded

Read each row whole with `go run ./tools/claim <ID>` from your worktree's `research/`. This contract
names them and does not restate them.

| Row | Subject |
|---|---|
| `TRIG-CHECK-051` | check 4: its gate, and what it reads on the literal it is gated on |
| `TRIG-CHECK-052` | check 16: the container search, the null answer, and which two points the distance is between |
| `TRIG-CHECK-053` | check 21 and instant 26 as one getter/setter pair over a 16-bit field |
| `TRIG-CHECK-054` | the corpus, the per-arm counts, and each arm's own limit |
| `SAV-BLDG-037` | the save record that reads the same field, from a third site sharing no code with either script arm |

**Check 16 is the one to write carefully.** `EXP-0080`'s vocabulary note reads that arm's distance
from the wrong end, the note is uncited, and `TRIG-CHECK-052` refutes it. A consumer that follows the
note produces a plausible number and the wrong trigger behaviour.

**Check 21's mechanics are High and its meaning is Unknown.** The shipped campaign's own editor labels
for all sixteen authored nodes are switch and puzzle-state names. That Unknown does not block the
work: the arm reads and writes one 16-bit field whatever the field means, and a build that stores and
returns it is faithful to the mechanics without choosing a label. Do not name the field for what you
guess it holds.

## The four behaviours

### B1 — check opcode 4

Evaluated per `TRIG-CHECK-051`, including its gate and the off-gate case, which writes nothing.

### B2 — check opcode 16

Evaluated per `TRIG-CHECK-052`: the container search, the null answer, the distance and its width.

### B3 — check opcode 21 and instant 26, over the minimum per-structure state

**This build has no structure state at all.** `grep -ni structure pkg/sim` returns comments and one
cell-occupancy slot; structures exist as art in `pkg/game` (`terrain.StructureSet`) and as nothing in
the simulation. The arm reads a 16-bit field on a referenced structure and instant 26 writes it, so
this behaviour has to introduce per-structure identity and that one field, in the simulation and in
its serialized byte form.

**The boundary of this story is: identity, one 16-bit field, and its serialization.** If the work
turns out to need more than that — structure health, occupancy semantics, destruction, a second field
— **stop and ask.** Growing this behaviour into structures-as-a-subsystem is a different story and a
different contract.

**This behaviour is expected to bump `formatVersion`.** Say so in your return the moment you know.
Story `1032` is open in `wt-1032` and is building a per-version reader over versions 50 to 57; a new
version is one more row in its table, and whichever of the two stories lands second merges master
first and adds it. Do not coordinate with that lane directly — report to the seat.

### B4 — the census falls, and a witness proves it in a real mission

The three arms leave `scriptCheckSupported`'s absent set and the census's 33 nodes go to zero.

A census number is not a witness on its own: it says a node is no longer refused, not that the
trigger it feeds now fires correctly. The witness this story owes is **a shipped campaign mission
whose trigger fires because one of these arms answered**, run from the tree rather than reasoned
about.

## What is deliberately not in this story

- **Check opcode 12**, decoded byte-identical to 17 and authored zero times. `DIV-243` records story
  `1029`'s exclusion and it stands.
- **What `structure+0x42` means.** The Unknown is the field's semantic, not its mechanics.
- **Structures as a subsystem** — health, destruction, occupancy, art-to-state binding.
- **Instant opcodes other than 26.**

## Ceiling

**Four adversarial passes.** The story reaches hashed simulation state: script registers are hashed,
and B3 adds state to the byte form.

## Domains

Campaign & Scripts, Sim Core, and Persistence for B3's serialization. Three.

## G2 — the limits these decodes imply

`TRIG-CHECK-054` gives each arm's own limit and class, and `DIV-239` already records the split: check
4's gate and check 16's 8-bit X/Y pair are engine constants whose lifting changes no shipped file,
while check 21's width is a stored save-record field whose lifting changes the save record's bytes.
Say each of the three in `spec.md` in those terms.

## Divergence rows this story is expected to owe

`DIV-264` through `DIV-269` are reserved for this story in `PIPELINE-STATUS.md`. `DIV-239` is the row
this story closes; close it rather than opening a fourth row that says the same thing, and move it to
`DIVERGENCES-CLOSED.md` only if all three arms land.

Expect rows for anything B3's minimum structure state does that the original does differently, and for
check 21's semantic Unknown if this build has to make any choice the claim does not give.

**If the range is spent, stop and ask.** Two other lanes are open, each holding a range of its own,
and an unmerged lane branch is invisible to `pipeline/next-div-id.sh`.
