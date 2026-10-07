# Story `1029` — the two script checks the campaign authors and this build cannot run

## Result

Check opcode 17 and check opcode 9 evaluate. Every trigger in a shipped campaign map that reads
either one stops being permanently inert.

Pointable in `pipeline/check-milestone.sh`'s census, which counts the shipped script nodes this
build refuses and can only fall: **59 to 33 on both roots**, and the three maps whose whole gap is
these two opcodes stop printing a `cannot run` line at all. The story names those numbers in advance
and regenerates the baseline rather than editing it, which is that script's own standing rule.

`DIV-...` rows carry whatever the two arms cannot reproduce exactly, and one row carries the three
opcodes that remain.

## Why this story exists

**G1**: a fully working game on the original assets, both roots. A check the engine evaluates and
this build cannot leaves every trigger holding it inert, so the authored chain behind it never runs.
Fourteen of the twenty-eight shipped campaign maps carry at least one such node today.

The census exists to make that measurable, and the project's own rule is that a story owes a
concrete result someone can point at — a number in that census that falls, or something visible in
`builds/current/`. This story takes the first.

Five check opcodes make up the whole of the census count. Two carry a published claim and are this
story's. Three do not, and a research experiment on those three is queued in the research
lane. **No experiment id appears in this document, deliberately**: a sentence naming one is a
citation to `scripts/check-claim-citations.sh`, which cannot tell a citation from a sentence about
one, and an experiment that has not landed is not in the pin.

## What is decoded

Both rows are in the pin (`790cce1`). **Read each whole** with `go run ./tools/claim <ID>` from
`implementation/research`. What follows names which row answers which question. It is not the
evidence and must not be cited in place of it: a row's own next sentence has cost this project a
landing before.

| Question | Row | Grade |
|---|---|---|
| What check 17 tests, what it reads, what it writes, and that it writes no simulation state | `TRIG-ITEMTEST-040` | High for the bodies and the reads / Medium for the authored counts |
| What check 9 returns, from which fields, and what it does when the order is not a pursuit | `TRIG-TARGETID-032` | High for the branch, field reads and zero result / Medium for the field names |
| Where a check's result goes | `TRIG-STORE-002` | cited by both rows above |
| What the register poisoning of an unsupported arm does today | `TRIG-COND-003` | already built |

`TRIG-ITEMTEST-040` also names check opcode **16** as one of the four callers of the same item
finder. That is a lead for the queued experiment and it is **not** a licence to implement 16 here:
the row establishes which routine 16 calls and nothing about what 16 answers.

## What is not decoded

- **Check opcodes 4, 16 and 21.** They are the other 33 nodes of the census count and no row in the
  pin gives any of them. They stay unsupported, they stay named in `Script.Unsupported`, and their
  registers stay poisoned. One divergence row records that the census's remaining half is a research
  gap. The seat fills that row's experiment reference at the landing, once the experiment is in the
  pin and its id resolves.
- **What check 9 does with a stale target.** `TRIG-TARGETID-032` states that the original's arm has
  no null branch and faults. This build does not fault, so the arm's behaviour there is authored and
  owes a row.

## What this build does today

Premises for the lane to check, not facts to inherit. Each was measured at this seat before this
contract was written; if one does not hold, say so rather than working around it.

- `scriptCheckSupported` (`pkg/sim/script.go`) is the one table that decides what this build
  evaluates, and its own comment says why it is the only place the answer is given. Neither opcode
  is in it. An arm outside the table reaches `ScriptSilenceUnsupported`, is named in
  `Script.Unsupported`, and `NewScript` poisons its register so every trigger reading it is marked
  inert before a tick runs.
- The check arms live in two switches: one before the unit reference is resolved, for arms that name
  no unit, and one after. Both of this story's arms name a unit and belong in the second.
- **`ScriptCheck` has no item reference.** `ScriptInstant` gained one in story `0156` as its sixth
  reference — a packed `uint16` code with its own presence flag, carried rather than resolved, and
  converted from the map file's `Target_Item` by the loader. That field's own doc gives the reasons
  and they apply here unchanged. `pkg/mapload/script.go` already decodes `typeItem` for the instant
  binder at line 314 and drops it when it builds a check at line 325.
- **The compiled script is serialized.** `scriptbinary.go` writes each check at `scriptCheckLen`
  bytes with fields through `o+72`; `formatVersion` is 56 in `binary.go`. Widening the check record
  moves the form.
- `Entity` carries `AttackTarget` and `HasAttackTarget`, which is the pursuit this build has. It
  carries **no authored map id**: a check's `Unit` is resolved by the caller through three
  identifier bands, so the sim holds no reverse mapping from an entity to the id a map file gave it.
  Check 9 returns exactly that id, chosen at run time rather than at bind time, so no compile-time
  binding can supply it.

## Scope — four behaviours

**B1. A check node carries an item reference.** `ScriptCheck` gains it on `ScriptInstant`'s own
terms and for the same reasons that field's doc already states: a packed `uint16`, carried and not
resolved, with its own presence flag, converted by the loader and not by this package. The binder in
`pkg/mapload/script.go` stops dropping it. The byte form widens and `formatVersion` moves once.

**B2. Check 17 evaluates.** The register takes 1 when the named unit's container holds the named
item code and 0 when it does not. No simulation state is written, which is the row's own word. The
comparison is code against code with no mask, because both sides are packed codes.

**B3. Check 9 evaluates.** The register takes the authored map id of the unit the subject is
currently pursuing, and 0 when the subject is not pursuing. The subject's authored id has to reach
the sim for this to be answerable at all; **how** is the spec's decision, and the two candidates are
a field on `Entity` filled by the loader, or a table the world carries beside the entity list. State
which and why. If it is a field, it rides B1's single `formatVersion` move rather than a second one.

**B4. The census falls and the baseline is regenerated.** `pipeline/check-milestone.sh --update`
after the arms land, on both roots, with the new file committed. The story's `closure.md` prints the
before and after counts side by side. **The baseline is regenerated, never edited**: that is the
script's own rule and the reason its header gives for it.

## Out of scope

- **Check opcodes 4, 16 and 21.** The queued research experiment's, not this story's.
- **Check opcode 12.** `TRIG-ITEMTEST-040` establishes it is the same 64 bytes as 17 and that the
  campaign authors it zero times. Implementing it costs one table entry and witnesses nothing, and a
  node no shipped map carries cannot be seen to work. If the spec adds it anyway because the arm is
  literally the same code, say so and record it; do not add it silently.
- **What a trigger does once its register is live.** The trigger machine already exists and this
  story writes registers into it. Any behaviour change a shipped map now shows is the point, not a
  side effect, and `closure.md` records the ones the scenarios reveal.
- **The order model.** Check 9 reads the pursuit this build already has. It does not change
  acquisition, chase, or what makes a unit pursue.

## The twelve aspects

Which apply, so `closure.md` fills each with PASS, N-A or GAP and an in-scope GAP fails the landing.

| Aspect | Applies | What it is here |
|---|---|---|
| Data | yes | the check record gains a reference the loader already decodes and drops |
| Runtime state | yes | whatever B3's design puts the authored id on |
| Simulation | yes | a register write is hashed state |
| Player input | no | nothing this story builds is reachable from a key or a click |
| AI | no | check 9 reads the pursuit; it changes nothing that decides one |
| UI / HUD | no | nothing draws |
| Triggers and scripts | yes | the whole of the story |
| Inventory and equipment | yes | check 17 reads a unit's container |
| Persistence and save-load | yes | one `formatVersion` move, covering B1 and B3 |
| Campaign and session | yes | shipped campaign maps are where both opcodes are authored |
| Shipped content | yes | the census before and after, on both roots, and the three maps that clear |
| Interactions with existing mechanics | yes | triggers that were inert become live, and a shipped map's behaviour changes |

## Domains touched

`DOMAINS.md` names nine. This story touches four:

- **6 Campaign & Scripts** — both arms, the supported table, the trace.
- **2 Sim Core** — a register write is hashed state, and the entity record may widen.
- **9 Persistence** — one `formatVersion` move, covering B1 and B3 together.
- **1 Assets** — `pkg/mapload`'s check binder, which stops dropping a reference it already decodes.

## Ceiling

**Four adversarial passes.** Both arms write registers, which are hashed simulation state, and the
story touches four domains. Four is the rule's answer for either condition; neither compounds it.

Four behaviours is inside the five the project's rule allows without recording a reason. They are
one result: B1 exists only to make B2 possible, B3 shares B1's single version move, and B4 is the
measurement that says the other three landed.

**Reaching the ceiling is a scoping diagnosis, not a failure.** If a fourth pass returns it, land
what works, open the remainder as its own story, and say in `closure.md` that the story was cut too
large.

## The witness

A register this build writes is invisible unless something reads it, so a test that asserts the
register alone witnesses half the story. Every arm needs both: the register's own value, and a
trigger that reads it firing when it should and not firing when it should not.

**The shipped maps are the integration witness.** `pipeline/check-scenarios.sh` and the campaign
sweep both drive real maps off the preserved installs, and three maps stop printing a `cannot run`
line at this story's own landing. Name which trigger chains those three maps have that were inert
and are not.

**Run the install-gated seat gates by name**, from `<seat>`, on **both** roots:
`pipeline/check-scenarios.sh`, `pipeline/check-release-tests.sh` and `pipeline/check-milestone.sh`.
The repository's own chain cannot reach the first two — golden rule 2 keeps an install out of a
test, so those tests gate themselves on `AGAINROM_` variables, and **a skip and a pass both print
`ok`**. Both scripts exit 2 with no install, so a bare run is never a pass. Compare against the
number each prints, not against a number written here.

**`check-milestone.sh` needs a drive of its own.** Set `AGAINROM_MILESTONE_DRIVE` to a `missionrun`
the lane built itself. The default is `builds/current/`, which every worktree on this machine
shares, and overwriting it while another lane is running has already happened.

## Divergence ids

`DIV-239`..`DIV-244` are reserved for this story, allocated with `pipeline/next-div-id.sh` on
2026-08-22 against a highest id in use of `DIV-238`. Allocate from that range and record any tail as
returned unused; a returned id is never reissued.

**If the range runs out, stop and ask the seat.** Do not run the allocator and take the next free
number: it scans the ledgers and one file outside the repository, and an unmerged lane branch is in
none of them. Story `1028` spent two ids nobody could see by doing exactly that.

Rows owed, at least:

- The three check opcodes that remain unsupported, with the census count they hold. Leave the
  experiment reference for the seat to fill at the landing.
- Check 9's behaviour on a stale or absent target, which the original faults on and this build must
  not.
- Whatever B3's authored-id design does not reproduce exactly.

## Gates

Both repositories' Go chains by glob — run the glob, never a remembered list; the glob is the
authority on how many there are — with exit codes captured as `out=$(bash scripts/check-x.sh 2>&1);
code=$?`, never read off a pipe. `go test -trimpath`, because Windows Defender quarantines one test
binary.

Then the three seat gates above on both roots. Report the **number** each script prints, not its
verdict: a count that does not move when work is added is a gate that did not see the work.

Deletion set empty or explained: `git diff --diff-filter=D --name-only <before> <after>`.

No commit trailers of any kind.

## Style

`PROSE.md` governs every document this story writes. English.
