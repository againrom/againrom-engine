# 0091 — analysis

**Intensity:** spec-anchored / static. **Terrain:** brownfield in all three areas — `pkg/sim`'s
sight law, `pkg/mapload`'s spawn block, `pkg/ui`'s opening view — each has behaviour to preserve.

## What was not known, and what was looked at

Two constants this tree authored are replaced by numbers the original carries. They are unrelated
in code and share only that act.

### The sight range

`0090` left the march's range as one package constant, 5 cells, disclosed as its D-1. The reason
recorded there was `AI-SIGHT-006`'s writer count: a `disp:a5` sweep returns ten hits over ten
owners, of which **two** are writes — the actor constructor's default 5 at `L00260`, and one
store in an arm of the per-actor state machine. That count is accurate and the conclusion drawn
from it is not, which is what was checked first here rather than taken on report.

A displacement sweep sees an instruction that names `+0xa5` as a displacement. Two writers cannot
be named that way and both were read:

- `DAT-HUMANS-008` maps the `Humans` streamer slot by slot; slot **8** lands at `+0xa5`. The store
  is inside a helper (`R0281`, `MOV byte ptr [ECX],AL`) that holds a **pointer** — the
  displacement is on the cursor, not on the actor. `UNIT-STREAM-001` puts the `Units` streamer's
  slot **10** at the same byte through the same helper family.
- `HERO-SIGHT-007` has the hero recompute write the whole `u16` at `+0xa4`; `+0xa5` is that word's
  **high byte**, so the instruction names an address one lower.

So the instrument saw everything it could see. `pkg/data` was then read to check the second half of
the premise: `UnitDef.ScanRange` (slot 10) and `HumanDef.ScanRange` (slot 8) already decode that
column on both bands, and `unitCtorDefaults.ScanRange` is 5 — the constructor's own value, which is
where 0090's constant came from.

A name that has to be kept apart: `UnitDef.Sight` is a **different** field, defaulting to 0, with
no column and no reader anywhere in the tree. It is not `+0xa5`. Nothing here touches it.

**The corpus was measured before the shape was chosen** (`verification.md` carries the run). Over
both roots the `Units` column takes nine distinct values 4…12 across 56 rows and the `Humans`
column four values 4…7 across 210 rows; on mission 10's own 35 placements it is 5, 6 or 8, and 30
of the 35 are not 5. So the constant is wrong for most of the mission the milestone drives, which
is what made this the current hypothesis for that drive.

Not settled here, and left as such: whether the human recompute runs at spawn for an `.alm`-placed
person. `DAT-HUMANS-009` carries the same question at Medium about the `Defence` column. It is
avoided rather than answered — a placed person takes the streamed column, a generated hero the
derive, and nothing has to decide which of the two would win.

### The opening view

`0088` shipped 20 columns as AUTHORED, and disclosed it as an **upper bound**: 640/32 with the side
panel undecoded. `SESS-VIEW-028` decodes the panel — the view's rect is the screen minus a
160-pixel right strip, and the spans it yields are 640x480 -> 15x15, 800x600 -> 20x18, 1024x768 ->
27x24. The same row establishes that a consumer selecting no resolution runs the 640x480 arm, from
two branch displacements landing on it and from `InitInstance`'s own `"-640"` fallback.

`MISSION-VIEW-019` was read for the centring, which is not this story's to move: the opening origin
is `heroCell - viewportSpan/2`, arriving as a network packet rather than as a store to the view.
`MISSION-VIEW-020`'s persisted `[View] X/Y` restore is a second path onto the same camera and is
graded Medium on inputs that are not facts about the image; it is not modelled and is disclosed.
