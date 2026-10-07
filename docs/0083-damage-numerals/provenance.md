# Provenance — the evidence behind the floating numeral

Every claim below is read from `research/claims/anim.md` at the pin the submodule stands on. Claims
are cited, never experiments.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — the notification carries the victim's NEW HEALTH as a level, never a delta, and the client subtracts to get the figure | `ANIM-BLOW-019` | High |
| FR-1 — what is drawn is a decimal numeral and nothing else (`"%d"`, one format literal) | `ANIM-NUM-020` | High |
| FR-2 — the figure is built only where the display flag is set AND the held health exceeds the new health | `ANIM-BLOW-019` | High |
| FR-2 — the new health is stored whatever the display flag says | `ANIM-BLOW-019` | High |
| FR-3 — a record already present for the same victim takes the new damage ADDED INTO ITS OWN number rather than becoming a second record | `ANIM-NUM-020` | High |
| FR-4 — the colour table is the VICTIM OWNER's player colour, taken unconditionally because the constructor's colour argument is a literal zero at its one call site | `ANIM-NUM-020`, `ANIM-NUM-021` | High |
| FR-5 — a three-band severity colour is chosen on every landed blow and read by nothing: three writes, zero reads, no address taken; the bands are `>= max/2`, `>= max/4`, below | `ANIM-NUM-021` | High for the absence of a read / Medium for the three targets being colour ramps |
| FR-6 — the life is 1000 ms of WALL CLOCK, taken from a wall-clock call at birth and compared at the draw; a record at exactly 1000 ms survives | `ANIM-NUM-020` | High |
| FR-6 — the life is independent of the game's cadence, so a slower speed means a shorter journey | `ANIM-NUM-020` | High |
| FR-7 — the drift is stepped once per ANIMATION TICK: vertical by `-2` unconditionally, horizontal by one, whose sign is the ownership flag's | `ANIM-NUM-020` | High |
| FR-7 — the animation tick is the game's own paced logic tick, whose period is the speed setting's | `ANIM-CLOCK-001`, `TERR-ANIM-008` (already load-bearing in this tree for the water) | High |
| FR-8 — the initial offset is the victim's own `vt+0x20()` scaled: `16` horizontally with the sign taken from ownership, `-0x30` vertically | `ANIM-NUM-020` | High |
| FR-9 — the display is optional and DEFAULTS TO ON: the map view's constructor sets the flag to 1 | `ANIM-NUM-020` | High |
| FR-9 — the toggle requires the Ctrl latch and is reached from vkey `0x4c` alone — Ctrl+L — solved by scanning the handler's own byte table, 1 of 161 | `ANIM-NUM-020`, `AI-KEYMOD-059` | High |
| FR-10 — the draw issues the same text virtual TWICE at offset positions, a shadow and a face | `ANIM-NUM-020` | Medium (the glyph blit was not followed into the font) |
| FR-11 — the drawn position is an OFFSET FROM THE VICTIM'S OWN screen position, and the drain is the map view's paint rather than a text control | `ANIM-NUM-020` | High |
| FR-11 — the record survives while its draw is skipped, when the victim drawable carries a set flag | `ANIM-NUM-020` | High that the arm exists / Unknown what the flag means |
| Out of scope — a blow also plays two positional sounds, the attacker's swing and the victim's grunt | `ANIM-SND-022` | High |

## Ours by choice

| What the spec fixes | Why it is ours | What would overrule it |
|---|---|---|
| FR-4 — the colour VALUES. The table at the original's colour base is thirty-two bytes per player index and is not decoded. | The spec fixes the RULE (colour is a function of the victim's owner) and authors the values. | A decode of the player colour table. |
| FR-8 — the scalar the offset is multiplied by. `vt+0x20()` is not decoded, so nothing in this tree names what it returns; the spec takes **one**, which reproduces the offsets exactly as `(±16, -48)`. | A plausible substitute chosen silently would be indistinguishable from a decoded number. | A decode of `vt+0x20`. |
| FR-8, FR-7 — that the offset and the drift are in CELL-LOCAL pixels and go through the camera, so they scale with the zoom. | The original has one resolution and no zoom, so the question does not arise there. | Nothing decoded; it is a consequence of this tree having a camera. |
| FR-10 — the shadow's OFFSET. Decoded is that two issues happen; the offsets are inside a routine not followed into the font. | Same reason as FR-4: the shape is decoded, the number is not. | A decode of the glyph blit. |
| FR-9 — that bare `L` keeps the chip key and Ctrl+`L` takes the numerals. | `L` was bound by an earlier story on the tree's own authority, and the decoded key needs the same letter under a modifier. | Nothing; the decoded half is Ctrl+L and it is honoured. |
| FR-6 — the life is tested on the frame's own step rather than inside the paint. | This tree separates a pure step from a draw; both run once per frame, so no frame differs. | Nothing. |
| FR-1 — health is observed once per frame rather than once per notification. | This tree has no per-blow notification; the seam carries state. Two blows inside one frame arrive as one subtraction, which is the sum FR-3's merge produces anyway. | A per-blow event seam, which nothing asks for. |
| FR-12 — the mission tool's absent/present reporting shape. | A developer tool; nothing decoded says anything about it. | Nothing. |

## Open — undecoded, and deliberately given no meaning here

- The **second half of the merge key**. The original keys the merge on the victim AND on a second
  field taken from a global at construction. What that global is was not decoded, so this build
  keys on the victim alone. Where the two differ is unknown and unrepresentable here.
- **What suppresses the draw.** The flag on the victim drawable that skips the draw while leaving
  the record alive has no decoded meaning. This build has no counterpart and draws whenever the
  victim is in the frame's snapshot.
- **The player colour table's contents** (FR-4) and **the glyph blit's shadow offsets** (FR-10).
- **What `vt+0x20()` returns** (FR-8).
- **The three severity colour ramps' contents.** Read at their three stores and at a packing idiom,
  not decoded — and, since nothing reads the choice, of no consequence to a consumer.

## Removed

| Dropped | Why |
|---|---|
| The severity colour band, in every form — a flash, a tint, a three-colour numeral. | `ANIM-NUM-021`: computed on every blow and read by nothing. Rendering one adds what the original deliberately discards. It is recorded in the spec as a deliberate absence rather than left unmentioned, because its absence is the finding. |
| The two sounds of `ANIM-SND-022`. | This tree has no audio at all — no device, no channel manager, no `Sound` array reader. Half-building one hook would be a claim we cannot support. |
| The `-10` health bound and the three-way band the sound hook takes. | It gates the SOUND, not the numeral. A consumer that applied it to the figure would suppress numerals the original draws. |
