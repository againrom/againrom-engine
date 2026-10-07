# 0093 — plan

## Shape

Two layers. `pkg/sim` gains a **record** and a second way to advance a tick that produces one;
`cmd/missionrun` gains a **readout** that turns a record into the map's own terms. Nothing between
them, and nothing in the runtime asks whether it is being observed.

## Design decisions

**DD-1 (FR-1) — the record is a return value, not a hook.** `Step` keeps its signature and becomes
a one-line wrapper over an unexported stepper that takes a record pointer; a second exported entry
point passes a fresh record and returns it. The two alternatives were both refused: a callback or
sink **stored on the world** is state beside the digest, and the world's field set is pinned
against exactly that; a **package-level** sink is the same fault for every world at once. A return
value also makes the run reproducible — an observed run holds nothing between two ticks.

**DD-2 (FR-1) — a nil record is the ordinary path.** Every recorder is a method on the record
pointer whose first line returns when the receiver is nil. So the runtime carries no `if tracing`
branch, the ordinary step passes nil, and "observation changes nothing" is a property of the shape
rather than of a flag being read correctly at every site.

**DD-3 (FR-2) — a firing is recorded between the condition holding and the first instant.** An
instant may write a register a later trigger reads, so the values a pair was compared on exist
only at that point in the pass. Recording after the pass would be cheaper and wrong; recording
inside the condition test would mean editing the hot predicate, so the predicate stays untouched
and the registers are re-read at the one instant where they cannot have moved.

**DD-4 (FR-4) — an unimplemented check arm leaves the dispatcher early.** It currently falls
through to a switch it matches no case of, by way of an entity lookup with no side effect. Leaving
before that lookup is the same nothing and is what lets its silence be recorded as *unsupported*
rather than as an unresolved reference it never got as far as testing.

**DD-5 (FR-5) — register→check attribution is a linear scan.** The constructor already refuses two
checks owning one register, so the answer is unique; the largest shipped script has 64 checks and
the question is asked by a line formatter, never on a tick's path. A stored index would be a
second representation of a fact fixed at compile time.

**DD-6 (FR-7) — the readout is a flag on the existing drive, not a new command.** That tool
already starts a mission as the front-end does, issues only ordinary orders and stops when the
world decides; a second tool would repeat all of it to add one sentence. Every advance in the tool
goes through **one stepping seam**, so no advance can escape the readout — and with the readout
off that seam calls the ordinary entry point, not the observed one with its result dropped.

**DD-7 (FR-7) — opcode names live in the readout, not in `pkg/sim`.** The two support tables in
the runtime are the only place "does this build have this arm" is answered; a name table beside
them would be a second answer to it. In the readout a name is presentation, and an opcode outside
the table prints as its number.

**DD-8 (FR-7) — a silence is printed on change, not per pass.** A silent check is a standing
condition and repeats on every pass for the whole mission; printed per pass it buries the events
at a hundred and sixty lines a mission. Firings and self-counted losses are events and print every
time.

**DD-9 (FR-8) — the walk reports the radius it actually met.** The drive answered *arrived* when
the world decided mid-walk, and that answer is what produced the reading that the mission was lost
by an interception short of its waypoint. The decision now ends the walk without claiming it.

## Risks

- A firing whose pairs are recorded from the registers rather than from the comparison itself
  could drift from what was compared if a write were ever introduced between the two points.
  DD-3's placement is one statement in one function, and AC-3a pins the case that would expose it.
- The readout's names are ours. A wrong name is a misleading line, not a wrong measurement; every
  line also carries the numeric opcode's own subscript, so a name can be checked against the code.

## Success criteria

- **SC-1** — a scripted world advanced observed and unobserved from one state agrees byte for byte
  and digest for digest at every tick, across several passes and at least one report.
- **SC-2** — the pinned canonical field sets and the byte form's version are untouched by this
  story.
- **SC-3** — synthetic scripts witness a firing with its compared values, a self-counted loss with
  no firing, each silence reason once, the two by-design silences not recorded, an empty record on
  a non-script tick, and register ownership both ways.
- **SC-4** — the tenth campaign mission, driven with the readout on against both a lawful EN and a
  lawful RU install, names the arm that decides it and the tick it acted on, and the two agree.
- **SC-5** — the same drive with the readout off prints exactly what the pre-story binary printed
  for the same arguments, except where FR-8 changes a status word.
- **SC-6** — a drive cut short by the decision reports it as such.
- **SC-7** — `go build`, `go vet`, `gofmt`, `go test`, and the asset, doc-budget and SDD-audit
  gates.

## Traceability

| FR | DD | SC |
|---|---|---|
| FR-1 | DD-1, DD-2 | SC-1, SC-2, SC-3 |
| FR-2 | DD-3 | SC-3 |
| FR-3 | DD-2 | SC-3, SC-4 |
| FR-4 | DD-4 | SC-3, SC-4 |
| FR-5 | DD-5 | SC-3 |
| FR-6 | DD-1 | SC-3 |
| FR-7 | DD-6, DD-7, DD-8 | SC-4, SC-5 |
| FR-8 | DD-9 | SC-5, SC-6 |

**FR-9** landed as hotfix `65dfdf7`, not as a task of this story; folded into the contract by 0130.
