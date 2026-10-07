# plan — 0150 the explored map

## Design decisions

**DD-1 — `pkg/formats/sav` imports `pkg/formats/reg` to parse the store.**
The store's signature dword is `0x31415926`, which is the value `reg.Parse` accepts, and its framing
is `reg`'s framing: `0x18`-byte header, `0x20`-byte node records, `u32` pool length, pool. Writing a
second parser inside `pkg/formats/sav` would be a second copy of one format.

This is the first edge between two format leaves and `internal/archtest`'s allow-map is where it is
declared. The empty allow-set `pkg/formats/sav` carried was written (0144 P-1, D-7) against the
**map** tier: a leaf that could import the map tier would be one refactor from resolving the block
delta itself instead of reporting it. `pkg/formats/reg` is not the map tier. It is a sibling leaf
held to the standard library by its own empty allow-set, so the grant adds no external dependency and
no knowledge of a map, an archive or a world.

**DD-2 — `reg` gains `Size`, and the split is done in `sav`.**
`reg.Parse` tolerates bytes past the pool end, so it already reads the store out of a tail that has a
trailing region behind it. What it does not report is where the store ends. `reg.Size(data)` returns
the framed extent `0x18 + 0x20*nodeCount + 4 + poolLen` and validates that it fits. `sav` calls it
once and slices.

`Size` is in `reg` and not in `sav` because the arithmetic is the registry format's, not the save
format's, and because a `.reg` file inside a `.res` container has the same question.

**DD-3 — `File.Tail` becomes `File.Store` and `File.TailRest`.**
`Tail` was documented as carried verbatim and never opened. It becomes two fields. `Store` is the
store's own bytes to its declared extent, or nil where the tail does not frame as one; `TailRest` is
everything after it, or the whole tail in the nil case. `Marshal` appends `Store` then `TailRest`, so
their concatenation is what the field held before and the round trip is unchanged.

The parsed tree is held beside them as an unexported field and reached through accessors, so `Open`
parses once and a caller asking twice does not.

**DD-4 — A malformed tail is carried, not refused (spec FR-2).**
A save is evidence. `Open` accepting a file before this story and refusing it after would be this
story destroying a reader's access to a file for the sake of a section that file does not have.

**DD-5 — The decoded plane is per-cell bytes, and the width is the caller's.**
`Fog/Data` carries `W*H` cells and does not carry `W`. `sav` decodes to a flat plane of `sum(Data)`
bytes and reports the count; it does not divide by a width it does not have. The consumer supplies
the map's own dimensions and checks the product. That keeps the format leaf free of a map, which is
the property DD-1's allow-set grant is bounded by.

**DD-6 — The serialized byte form does not change, and version 43 is returned unused.**
The brief allocated a byte-form version on the premise that persisting the explored map would widen
`pkg/sim`'s form. It does not. Exploration is not simulation state in this tree: story 0118 put it in
`pkg/game`'s `fogPlane` because it is per-participant view, and `pkg/game`'s own save envelope has
carried it since 0143 as `SnapshotResidue.FogCols/FogRows/FogExplored`. Spec FR-7 is therefore
already met by the envelope, and this story witnesses it rather than building it.

Moving exploration into `pkg/sim` to mirror the original's tile plane would change every world digest
in the tree, for no behaviour a player could tell apart. It is not done. `formatVersion` stays 41 and
43 joins 28, 30, 33, 37, 40 and 42 as a permanent gap.

**DD-7 — The apply seam is `missionOpener`, after `openMission`.**
The fog plane is built inside `newMapWorldWith`, which runs inside `openMission`, which runs after
`StartMissionFrom`. The original save's `prepare` closure runs *before* `StartMissionFrom`, in the
one window where the map's unit records are still records. So the plane cannot be written from
`prepare`.

`prepare` changes shape from `func(*alm.Map) error` to `func(*alm.Map) (*originalFog, error)`: it
reads the save against the decoded map, whose dimensions it now has, and returns what must be applied
once the world exists. `missionOpener` applies it one statement after `openMission`, beside the line
that applies our own format's residue, and for the same reason — after the constructor's tick-0 push,
so the first frame draws the restored exploration.

**DD-8 — `applyExplored` is the one writer, and `applyResidue` delegates to it.**
Two restore paths now write the same plane. They share one method: dimension check, OR in, push. Our
own format's `applyResidue` keeps its behaviour by calling it.

**DD-9 — The trailing region's first dword is reported and not interpreted.**
`savtool` prints it because it is the one field of that region research has measured. Nothing reads
it.

## Work

| # | Change | Requirements |
|---|---|---|
| P-1 | `pkg/formats/reg`: `Size` returns the framed extent. | FR-1 |
| P-2 | `internal/archtest/dag.go`: allow `pkg/formats/sav -> pkg/formats/reg`, with DD-1's reasoning at the entry. | FR-1 |
| P-3 | `pkg/formats/sav`: split the tail into `Store` and `TailRest`; parse the store once; keep `Marshal` byte-identical. | FR-1, FR-2, FR-3 |
| P-4 | `pkg/formats/sav`: `Fog()` decodes `FirstState` and `Data` into runs and a per-cell plane. | FR-4 |
| P-5 | `pkg/game`: `applyExplored`; `prepare` returns `*originalFog`; `RestoreOriginal` and `ResumeOriginalSave` decode and carry. | FR-5, FR-6 |
| P-6 | `pkg/game`: witness the envelope round trip of the explored plane. | FR-7 |
| P-7 | `cmd/savtool` gains `fog`; `cmd/savecheck load` prints the restored cell count. | FR-8 |
| P-8 | Correct `OriginalSaveNote` and the counted resume report. | FR-9 |

## Risks

**R-1 — The store parse could reject a save the tree reads today.**
Bounded by FR-2: a store that does not frame is not a store, and the file still opens. Measured over
the whole preserved corpus in `verification.md`.

**R-2 — The linear order could disagree.**
`TERR-EDGE-024` gives the save's order as `idx = col + row*W`; `pkg/sim`'s `cellIndex` is
`y*Width + x`. They are the same expression. Checked by reading both, and by the restored count
matching the file's own count.

**R-3 — The drawn extent may look larger or smaller than the restored cell count.**
The render gate is untouched (FR-6) and how the original expands recorded cells at draw time is not
decoded. `verification.md` reports what was restored and states that nothing was tuned.
