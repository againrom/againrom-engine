# 0129 — give all: plan

Three tasks, in order: the arm inside `pkg/sim`, then the byte form that carries its new field, then
the binder in `pkg/mapload` that fills it. Each is one commit.

## Design decisions

**DD-1 — a second reference on the existing record, not a record of its own.** Opcode 28 is the only
arm in the vocabulary that names two units, and the tempting alternative was a dedicated compiled
form for it. It is refused for the reason the plain parameters are packed in encounter order:
`ScriptCheck` already carries a resolved pair through exactly this shape, the binder already computes
that pair for **every** node it binds — conditions and actions alike — and throws the second away
when the node is an action. So the field is not new machinery, it is the delivery of a value the
compiler already holds. One record shape, one binder, one place a reference is resolved.

**DD-2 — the arm moves a slice, it does not simulate a loop.** The original takes index 0 and adds it
at the tail until the source is empty. Over a list with no capacity and no refusal, that is exactly
an append of the whole source in its own order, and reproducing the loop step by step would only
create the chance of getting a partial state wrong. The claim's own wording — the whole container
moves, the source is then destroyed — is the append and the clear.

**DD-3 — the self-transfer is refused, not reproduced.** With both references naming one entity the
original's loop never terminates: it removes element 0 and appends it, so the count never falls. That
is undefined behaviour rather than behaviour, and this build refuses it in the same spirit
`script.go` already refuses an out-of-range register subscript instead of overrunning the array — the
one place a file knowingly parts company with what it reconstructs, and only where the original's own
behaviour is undefined. It is unreachable from the corpus: all three shipped nodes name a map unit
and the hero band, and no map unit is the hero.

**DD-4 — the giver's container becomes nil, and that is the re-seating.** A fresh empty container and
a nil slice are the same claim in this package: `Stock()` skips an entity holding nothing, the byte
form writes no record for one, and every entity that has never held anything is in that state
already. So P-3 costs no code — it is what falls out of clearing the slice rather than truncating it,
and a truncated one would encode identically anyway. Written down because it is the choice that makes
"a world where she gave them away" and "a world where she never had them" indistinguishable, which is
a property and not an accident.

**DD-5 — version 38, and the field goes on the instant record's own tail.** 36 is live; 37 is out to
a story running in parallel off the same master this one branched from, and this tree carries none of
its change. The five bytes — a `uint32` value then a flag byte — are appended after the instant
record's last existing byte, so no offset outside that record moves, which is the same treatment the
group pair and then the player pair got inside the check record.

**DD-6 — the arm is added to the supported table and to the dispatch in ONE task.** They are one
decision recorded twice, and the file already says why: a switch with a silent default is exactly how
an unimplemented arm becomes an evaluated-false one. Splitting them across two commits would leave
one commit in which the report and the dispatch disagree.

**DD-7 — the byte form is a task of its own, after the arm.** Precedent, and the reason for it: a
field that exists but is not yet serialized is a field two different worlds can hash equal on, which
is a small and visible debt for exactly one commit, whereas a task that both invents a field and
re-cuts the form is a task whose failure is ambiguous between the two.

## Success criteria

**SC-1 — the tool stops reporting the gap.** `go run ./cmd/almtool script <10.alm>` no longer prints
`instant arm 28: 1 node(s) not implemented`, and every other line of its report is unchanged.

**SC-2 — the potions reach the hero on a lawful install.** Mission 10 started through
`game.StartMission`: the compiled opcode-28 instant carries the entity the map calls unit 21 as its
first reference and the party's hero as its second; before the trigger fires, that unit holds three
`0x0e06` codes and the hero does not; after it fires, the hero holds them and that unit holds
nothing. Run from a developer tool, recorded in `verification.md`, on both language roots.

**SC-3 — the gate.** `go build ./... && go vet ./... && gofmt -l . && go test -count=1 -trimpath
./...` clean, plus `scripts/check-no-game-assets.sh`, `check-doc-budget.sh` and `check-sdd-audit.sh`.

## Traceability

| FR | Task | DD | Witnessed by |
|---|---|---|---|
| FR-1 | T1 | DD-1 | AC-7 |
| FR-3 | T1 | DD-6 | AC-6 |
| FR-4 | T1 | DD-2 | AC-1, AC-4, SC-2 |
| FR-5 | T1 | DD-2 | AC-2 |
| FR-6 | T1 | DD-3 | AC-3 |
| FR-7 | T2 | DD-4, DD-5, DD-7 | AC-7 |
| FR-2 | T3 | DD-1 | AC-5, SC-1, SC-2 |

`P-1` and `P-2` are properties of T1's arm; `P-3` of DD-4, witnessed inside AC-1's assertion that the
giver's list is empty and AC-7's round trip.

## Tasks

- **T1** — `pkg/sim`: the second reference on `ScriptInstant`, opcode 28 in the supported table and in
  `runInstant`, the pour and the four refusals. FR-1, FR-3, FR-4, FR-5, FR-6, P-1, P-2, P-3.
- **T2** — `pkg/sim`: the byte form carries the second reference; version 38. FR-7.
- **T3** — `pkg/mapload`: the binder carries its already-resolved second unit onto compiled instants.
  FR-2.
