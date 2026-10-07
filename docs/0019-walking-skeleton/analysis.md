# Analysis — the headless deterministic core

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — the profile names simulation determinism by hand as a cross-cutting engine contract; no watcher tool exists, so it is discipline |
| Terrain — `pkg/sim`, `pkg/mapload` | **greenfield**: each holds a `doc.go` and nothing else |
| Terrain — `internal/archtest` | **brownfield**: the wall's behavioural half lands beside a shipped structural check whose negative tests pin what it does and does not catch |

## The wall the baseline would have deleted

`cleandocs/0019-walking-skeleton/spec.md` FR-9 requires `pkg/sim` to import `pkg/formats/alm`. Here
that edge is forbidden, and the refusal is executable rather than a convention:

- `internal/archtest/dag.go` gives `"pkg/sim": {}` — an empty intra-module allow-list;
- `CheckSimTests` extends the same rule to `pkg/sim`'s **test** files;
- `dag_test.go` drives the evaluator with `pkg/sim -> pkg/formats/res` under the case name *"sim
  importing a formats package (determinism wall)"* and asserts the violation is named;
- `pkg/sim/doc.go` and `docs/ARCHITECTURE.md` state it in prose, written at 0000.

Satisfying the baseline would mean deleting an enforced invariant **and its own negative test** to
accommodate a document that predates both. The transform therefore lands in `pkg/mapload`, whose
allow-list is already `{pkg/formats/alm, pkg/data, pkg/sim}` and which today holds `doc.go` alone —
the tier the DAG created for exactly this join. `pkg/sim` stays stdlib-only and the allow map is not
touched.

## What that check does *not* cover, and the criterion that follows

The baseline's AC-9 — no `os`/`time` import and no float in `pkg/sim`'s non-test sources — is
sometimes read as redundant against `archtest`. It is redundant in neither half, and a landed test
says so in as many words. `Check` classifies an import whose first segment has no dot as stdlib and
**always permits it**; `dag_test.go` carries the case

    name:     "stdlib in sim is not a DAG violation (behavioural wall deferred)"
    imports:  {"pkg/sim": {"time", "math/rand", "os"}}
    wantEdge: ""      // expect no violation

and `docs/ARCHITECTURE.md` states the same limit: "the standard library still contains hazards
(`time`, `os`, `math/rand`) and floats need no import, so the import check does not by itself
guarantee behavioural determinism, and this document does not claim it does."

So what stands today is the **tier** wall — no other `againrom` tier, no external module, tests
included. The clock, the filesystem and floating point are all reachable from `pkg/sim` right now,
and that is the half this story closes. Nothing about the new check requires editing the allow map
or retiring the case above: that case is about the DAG evaluator and stays true beside it.

Both `AGENTS.md` and `docs/ARCHITECTURE.md` describe the behavioural half as "proven on a concrete
entity in a later (walking-skeleton) story". This is that story, so those two sentences stop being
true the day it lands, which is why the contract asks for them.

## What else the baseline names that this repo does not have

- `openrom` is `cmd/againrom`, and `sim.FromALM` is `mapload.FromALM`. The baseline's follow-up
  story number is a roadmap position, not a commitment this repo has made.
- Its provenance-basis preamble, and its closing section that exists only to report having no
  research item, are both refused outright by the doc-budget content bans, independently of size.
- Its FR-4 asks that "any observable state difference changes the hash". A 64-bit digest cannot
  promise that; the contract asks instead that no field escape the digest, and witnesses the
  single-field changes one at a time.
- Bounds arrive from the loader and constrain nothing, in the baseline and here. Worth saying out
  loud rather than leaving to be discovered: a skeleton with no collision and no clamp is a unit
  that walks off the map, so the contract discloses it instead of implying a limit it does not have.

## The one game fact, and the one identity we chose not to use

`alm.Unit.X`/`Y` are `uint32` fixed-point `/256` (`pkg/formats/alm/alm.go`), so the baseline's "8.8
position truncated to whole cells" is `value >> 8`. Backed at High; the ledger carries the citation
and the pin did not move for this story.

Entity identity is the more interesting one. The type-6 record carries a unique id of its own that
`alm.Unit` does not expose, so unit-slice order is **our** identity and not the game's. It is filed
in the ledger so a later story cannot inherit the assumption silently.

## What we looked at

`internal/archtest/{dag.go,dag_test.go}`, `pkg/sim/doc.go`, `pkg/mapload/doc.go`,
`pkg/formats/alm/alm.go` (`Unit`, `Map`), `docs/ARCHITECTURE.md`, `AGENTS.md`, both check scripts,
and in research `claims/alm.md` after `claims/retracted.md`.

Open, and disclosed rather than guessed: how long a tick is in the original (nothing here fixes one
— the tick is an index, not a duration); whether the original advances its units in any particular
order; and what it does with a unit ordered outside the map.
