# Tasks — 0122-trigger-opcodes

**Reading key.** `FR-n`, `D-n` → `spec.md` §The contract, §Divergences. `AC-n`, `P-n` →
`spec.md` §Acceptance, §Derived properties. `DD-n`, `R-n`, `SC-n` → `plan.md`.

Tasks are ordered. T2 must land before T3, T4 and T5; T1 is independent. Each leaves
`go build ./...`, `go vet ./...` and `go test ./...` green on its own.

---

## T1 — two doc blocks that outlived their own uncertainty *(hotfix)*

**Boundary.** Comments only. **No behavioural change of any kind**, and no test is added,
removed or altered.

**Files**

- `pkg/sim/script.go` — `MODIFY` (the doc block above `scriptDistance`; the paragraph inside
  `runCheck`'s group-count case)

**Covers** DD-11

Each block announces an open question the ledger has since closed, and this tree's behaviour is
right in both cases: the metric is settled, and the group count's exclusion of the dead is
settled from the reap's side rather than from the arm's.

**Fences.** Keep the byte-width caveat on `scriptDistance` — the mask-before-subtract ordering is
graded Medium and is **still open** (DD-11). Do not turn a corrected comment into a second
overstatement. Do not touch any function body.

**Done when:** neither block claims an open question that is closed; `scriptDistance`'s block
still records what remains open; `git diff` shows comment lines only.

---

## T2 — a compiled check names a player *(implementation)*

**Boundary.** The record, the binder and the form. **No new arm** — `scriptCheckSupported` and
`runCheck` are not touched, so nothing yet reads either field.

**Files**

- `pkg/sim/script.go` — `MODIFY` (`ScriptCheck` gains four fields and their doc paragraph)
- `pkg/sim/scriptbinary.go` — `MODIFY` (`scriptCheckLen` 63 → 73, encode, decode, offset map)
- `pkg/sim/binary.go` — `MODIFY` (`formatVersion` 26 → **31** and its doc block)
- `pkg/sim/binary_test.go`, `pkg/sim/hash_test.go` — `MODIFY` (version backstop; `pinDigest`)
- `pkg/mapload/script.go` — `MODIFY` (the `typePlayer` case, its comment, the bound-reference
  struct, the check literal)
- `pkg/sim/scriptform_test.go` — `ADD`

**Covers** FR-1, FR-2, FR-3 · DD-1, DD-2, DD-3, DD-4, DD-5 · D-1 (AC-6, R-1, R-4, SC-3)

**Fences.** Do not touch `ScriptInstant` or `scriptInstantLen` (DD-2). Append at the check
record's **tail**. Take the new `pinDigest` from the failing test's own output (DD-5). Instants
still bind the **first** player only.

**Done when:** a check carrying both references round-trips byte-identically; a version-26 form
is refused naming 26 and 31; a node with a player parameter *between* two plain parameters still
lands both where it did (R-4); the version doc block states that 31 was chosen without an
allocation (D-1).

---

## T3 — how many, and how near *(implementation)*

**Boundary.** Check arms 8 and 15. One walk of the world's entities, filtered by owner.

**Files**

- `pkg/sim/script.go` — `MODIFY` (two opcode constants, `scriptCheckSupported`, two `runCheck`
  cases)
- `pkg/sim/scripttrace.go` — `MODIFY` (`ScriptSilenceNoPlayer`)
- `pkg/sim/scriptplayer_test.go` — `ADD`

**Covers** FR-4, FR-5, FR-9 · DD-6, DD-7, DD-8 · D-2, D-3 (AC-1, AC-2, P-1, P-2, R-3, SC-1)

**Fences.** Build **no membership or owner index** (DD-6) — scan `w.entities` in its own order,
for the reason the group count's doc block already gives. Use `scriptDead`, not a fresh health
test. Do not extract a shared helper for the two arms. Both distances go through
`scriptDistance`/`scriptByte`. Do not touch the byte form.

**Done when:** each arm has a test that fails when its own case body is removed; arm 8's zero,
arm 15's empty-set `0xff` and the no-player silence are separately witnessed; a trigger reading
either register is no longer inert.

---

## T4 — what one player thinks of another *(implementation)*

**Boundary.** Check arm 10 and instant arm 10, over the relation matrix that already exists.

**Files**

- `pkg/sim/relations.go` — `MODIFY` (one new writer beside `turnHostile`)
- `pkg/sim/script.go` — `MODIFY` (two opcode constants, `scriptCheckSupported`,
  `scriptInstantSupported`, one `runCheck` case, one `runInstant` case)
- `pkg/sim/scriptdiplomacy_test.go` — `ADD`

**Covers** FR-6, FR-7 · DD-9 · D-4 (AC-3, AC-4, P-3, SC-4)

**Fences.** The write is a method on `Relations`; nothing outside `relations.go` may touch
`cells`, and the offset comes from `relationIndex` so an out-of-range slot stays one rule (DD-9).
Do not make the write symmetric, do not clamp the addend, and do not add a lock test — all three
would be plausible and all three are wrong. Do not notify, re-scan or invalidate an order. The
matrix is already in the byte form; do not touch it.

**Done when:** a mirror cell is untouched; bits 2..7 survive; `p2 = 5` on a cell of 4 gives 9 and
not 5; a locked pair is overwritten; an out-of-range slot changes nothing; a check naming fewer
than two players writes nothing; each behaviour fails when the arm's body is removed.

---

## T5 — two arms that are dead, and are dead here *(implementation)*

**Boundary.** Check opcodes 11 and 13. No register, no gap, no inertness.

**Files**

- `pkg/sim/script.go` — `MODIFY` (two opcode constants, `scriptCheckSupported`, one `runCheck`
  case covering both)
- `pkg/sim/scriptdead_test.go` — `ADD`

**Covers** FR-8, FR-9 · DD-10 (AC-5, SC-2)

**Fences.** Take the answer from `scriptCheckSupported` alone (DD-10): the arms are **supported**
and reach a `case` that returns having written nothing. Do not add a second "dead arm" table, do
not record a silence, and do not write a register — not even the value already there. No shipped
map authors either opcode, so this is reachable only by a customised map; do not claim a corpus
figure (R-2).

**Done when:** a trigger whose condition names an opcode-11 or opcode-13 check is **not** in
`InertTriggers()`; neither opcode appears in `Unsupported()`; the register such a check names
holds whatever it held before the pass; each of the three fails when the two opcodes are removed
from `scriptCheckSupported`.

---

## Traceability

| | FR | D | DD | AC | P | SC |
|---|---|---|---|---|---|---|
| T1 | — | — | 11 | — | — | — |
| T2 | 1, 2, 3 | 1 | 1, 2, 3, 4, 5 | 6 | — | 3 |
| T3 | 4, 5, 9 | 2, 3 | 6, 7, 8 | 1, 2 | 1, 2 | 1 |
| T4 | 6, 7 | 4 | 9 | 3, 4 | 3 | 4 |
| T5 | 8, 9 | — | 10 | 5 | 1 | 2 |

AC-7, P-4, SC-5 and SC-6 are the gate and the witness pass; they belong to no single task and are
answered in `verification.md`.
