# 0132-corpse-drops-worn — tasks

## Legend

**Kind** — `impl`, both. No other kind occurs here.

Order is strict: T1 lands before T2, because SC-3 is measured as the difference between two runs of
the same tool and the tool must exist before the behaviour changes.

## T1 — the drive reports a victim's holdings and the ground under him

**Kind:** impl · **Covers:** FR-8, DD-6

**Files:** `cmd/missionrun/main.go`, `cmd/missionrun/main_test.go`, `internal/archtest/dag.go`.

**Boundary.** Only the attack arm gains output; the waypoint arm, the census, the tracer and the
outcome lines are untouched and no existing line changes wording or order. The holdings are read
before the order is issued and the ground after the arm returns; neither read advances a tick or
issues a command.

`internal/archtest/dag.go` gains `pkg/data` on `cmd/missionrun`'s allow-list and nothing else, with
a comment saying why the tool names that tier.

**Scope fences.** Nothing in `pkg/` changes: nothing is exported to serve this and no helper is
added to `pkg/data`. No new flag — the report is unconditional on an arm that runs only when an
attack was asked for.

**Done when:** an attack line is followed by the victim's slots and container as they stood before
the blow and by the sack on his cell after, or by a statement that there is none; the code formatter
is a pure function with a synthetic install-free test covering a code the tables resolve as a weapon
and one they resolve as nothing; SC-1's gate is clean.

## T2 — a body drops every slot, and 0123's contract stops contradicting the code

**Kind:** impl · **Covers:** FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, DD-1, DD-2, DD-3, DD-4,
DD-5, DD-7

**Files:** `pkg/sim/step.go`, `pkg/sim/carry.go`, `pkg/sim/startequip_test.go`,
`pkg/sim/corpseloot_test.go`, `docs/0123-corpse-loot/spec.md`.

**Boundary.** Inside the crossing block only. No signature changes, no struct gains a field,
`binary.go` is not opened and the form version is not touched. Every doc block still asserting the
old two-slot rule — at the death path, at the retiring predicate, at the two slot constants — is
corrected; one left standing is a defect of this task.

`docs/0123-corpse-loot/spec.md` gains one sentence in each of its first two requirements naming this
story as what supersedes the armour clause. Nothing there is renumbered, no other requirement is
edited, no other file in that folder is opened.

**Scope fences.** No recompute of a derived number. No gold. No template-name suppression. No
pick-up-side change. `cmd/` is not touched.

**Done when:** AC-1 to AC-9 and AC-11 each have a named test in `pkg/sim`, the test asserting the
old rule is turned over in place rather than deleted, and SC-1's gate is clean.

## T3 — the same-tick tie-break is asserted, not disclaimed

**Kind:** impl · **Covers:** FR-6

**Files:** `pkg/sim/corpseloot_test.go`.

**Boundary.** Test file only — no package source is opened, nothing outside the two co-located-death
tests moves. The test that pins two loadouts felling on one cell gains the **opposite** case: the
same fixture killed by the two commands in the reverse order, asserting the two blocks swap and
nothing inside either does.

Its doc block currently records the tie-break as an unwitnessed gap, on a reading of FR-6 that no
longer stands. Replace that block with what FR-6 now says and what the test now shows; leaving a
comment that describes the contract as unmet is the defect this task exists to remove.

**Scope fences.** No production code. No new fixture helper. The other death tests are not
retitled and not rewritten.

**Done when:** both orders are asserted in one test or in two named ones, each naming AC-11; the
doc block claims no gap it does not have; SC-1's gate is clean.

## T4 — the loader's suppression gate stops giving a reason that has gone

**Kind:** impl · **Covers:** DD-8

**Files:** `pkg/mapload/spawn.go`, `docs/0128-wear-the-whole-row/spec.md`.

**Boundary.** **Comment and contract prose only — no behaviour changes.** The gate keeps exactly
the slots it empties today; not one index is added or removed, no signature moves, and no test's
expectation changes. The doc block above the starting-loadout gate argues from "a corpse gives up
only those two slots"; that premise is gone, and the block must instead say that the gate is
narrower than the drop and what follows from it for an NPC-templated body.

`docs/0128-wear-the-whole-row/spec.md` gains one sentence, in the requirement that states the gate,
recording the same thing. Nothing there is renumbered and no other requirement is edited.

**Scope fences.** Do not widen the gate. Do not touch `pkg/mapload/fromalm.go`, `pkg/sim`, `cmd/`
or any other story's docs. If you conclude the gate must move to be correct, STOP and say so
instead of moving it.

**Done when:** no comment in `pkg/mapload` argues from the two-slot drop; `0128`'s requirement says
the gate is narrower than the drop; SC-1's gate is clean.

## Traceability

| Requirement | Task |
|---|---|
| FR-1, FR-2, FR-3, FR-4, FR-5, FR-7 | T2 |
| FR-6 | T2, T3 |
| FR-8 | T1 |
| DD-1, DD-2, DD-3, DD-4, DD-5, DD-7 | T2 |
| DD-6 | T1 |
| DD-8 | T4 |
