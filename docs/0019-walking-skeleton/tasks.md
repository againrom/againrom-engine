# Tasks — the deterministic walking skeleton

Legend: **files** the task may change · **done when** the observable it must leave behind. Every
entry is an implementation task; they land in ascending order, and every dependency of a task is a
task before it.

## T1 — `pkg/sim`: the world, its readers and its RNG

**files** new `pkg/sim/world.go`, `pkg/sim/rng.go`, `pkg/sim/world_test.go`, `pkg/sim/rng_test.go`;
`pkg/sim/doc.go` (the opening sentence's tense alone)

DD-1's types, constructor and readers and DD-5's generator, under DD-9's internal-test placement and
its `reflect` method-set pin.

**done when** SC-1 and SC-3 pass (FR-1, FR-2, FR-5). The package holds no exported mutator at all
and no map type; `doc.go`'s determinism-wall paragraph is left alone — T5 owns it; nothing here
steps, encodes or hashes a world.

## T2 — `pkg/sim`: the command and the step

**files** new `pkg/sim/step.go`, `pkg/sim/step_test.go`

DD-2, over the storage T1 leaves.

**done when** SC-2 passes (FR-3, FR-4). Movement is unclamped and reads no bounds field; the step
becomes the package's one exported mutator and stays so; no digest or byte form appears here, the
test observing tick, entities and the unexported RNG state directly.

## T3 — `pkg/sim`: the canonical byte form and the digest

**files** new `pkg/sim/binary.go`, `pkg/sim/hash.go`, `pkg/sim/binary_test.go`,
`pkg/sim/hash_test.go`; `pkg/sim/world_test.go` (the pin and the sweep alone)

DD-3's encoder, decoder and refusals, and DD-4's digest over them. The three methods this task adds
put T1's method-set pin red until the pin names them and the no-mutation sweep covers the two that
only read — the mechanism DD-9 installed the pin for.

**done when** SC-4 passes (FR-6, FR-7). The offset test writes DD-3's offsets and widths out by
hand instead of deriving them from the encoder, and the pinned bytes and digest are literals rather
than values the test computes. No second version decodes and no migration path exists.

## T4 — `pkg/sim`: frames, the runner and the replay

**files** new `pkg/sim/run.go`, `pkg/sim/run_test.go`

DD-6.

**done when** SC-5 passes (FR-8, FR-9). `Replay` constructs no world, `Run` keeps no reference to
the schedule it was handed, and a log gets no byte form of its own.

## T5 — `internal/archtest`: the behavioural wall, and the documents that stop deferring it

**files** new `internal/archtest/determinism.go`, `internal/archtest/determinism_test.go`;
`internal/archtest/dag_test.go` (one case name), `pkg/sim/doc.go`, `AGENTS.md`,
`docs/ARCHITECTURE.md`

DD-7, landing after `pkg/sim` is complete so the live scan has sources to read.

**done when** SC-6 passes (FR-10). The allow map is unedited and `Check`, `CheckSimTests` and
`Load` keep their signatures and their behaviour; the renamed case keeps its imports and its
expectation; each of the three documents claims exactly what the scan performs, its stated limit
included, and none of them claims the DAG check proves it.

## T6 — `pkg/mapload`: the transform

**files** new `pkg/mapload/fromalm.go`, `pkg/mapload/fromalm_test.go`; `pkg/mapload/doc.go` (the
opening sentence's tense alone)

DD-8 — the first tests this package has carried.

**done when** SC-7 and SC-8 pass (FR-11, FR-12). `internal/synth` and `pkg/sim` gain nothing,
`pkg/data` is not imported, the expected cells are literals, and the post-step comparison runs
against a second built map rather than a struct copy of the first.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, FR-2, FR-5 | DD-1, DD-5, DD-9 |
| T2 | FR-3, FR-4 | DD-2 |
| T3 | FR-6, FR-7 | DD-3, DD-4, DD-9 |
| T4 | FR-8, FR-9 | DD-6 |
| T5 | FR-10 | DD-7 |
| T6 | FR-11, FR-12 | DD-8 |
