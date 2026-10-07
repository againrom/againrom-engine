# 0143 — provenance

## Research claims consumed

**None.** This story consumes no claim and allocates none.

That is the finding and not an omission. The format written to disk here is **ours to author**: it
is our own snapshot in our own envelope, and no fact about the original game is needed to write it,
read it, refuse it or list it. There is no `SAV-` id in this story's code, tests or contract, and
none was invented.

The original game's `.sav` is a different file with a different contract — a 16-byte header, a
run/literal codec, an MFC object graph, an obfuscated `Player` record — and the claims describing it
are consumed by the stories that read that file, not by this one.

## Ours by choice

| Choice | Why it is ours and not derived |
|---|---|
| the magic `AGRMSAVE` | our file, our identifier; it deliberately does not resemble `Asg&` |
| the envelope's field order and widths | authored |
| `saveVersion` starting at 1 | our namespace, independent of `pkg/sim`'s `formatVersion` |
| CRC-32 (IEEE) over the payload | stdlib, sufficient for "refuse a corrupt file"; not a security property |
| `encoding/gob` as the payload | plan D-1 |
| `save-YYYYMMDD-HHMMSS.ags` | authored; the player types nothing |
| `saves/` beside the executable | the owner's ruling, 2026-08-12 |
| four entries `RETURN / SAVE / LOAD / EXIT` | the owner's ruling, 2026-08-12 |
| `L` on the main menu for LOAD GAME | authored — see below |

## The one place a claim would have been used, and was not

**Which brooch button is LOAD GAME.** `pkg/render/menu` binds exactly two of the eight: NEW GAME,
computed from the placement table because the owner's knowledge is about a position on screen; and
button 8, a literal because the original's click dispatcher posts `WM_CLOSE` for it. The other six
are reported undecoded.

So this story binds none of them. The LOAD GAME window is reached from a key, with a line on the
menu saying so. When the button is decoded — or when the owner names its corner, which is the same
evidence class NEW GAME already rests on — binding it is one statement and the key can stay beside
it.

## Facts taken from our own tree, re-measured in this lane

| Fact | Where | Checked |
|---|---|---|
| nothing under `pkg/` or `cmd/againrom/` writes a file outside tests | grep for `os.WriteFile`, `os.Create`, `os.OpenFile`, `os.MkdirAll` | holds at `60d55fa`; every hit is a `_test.go` |
| `pkg/sim`'s byte form is at `formatVersion = 41` | `pkg/sim/binary.go:706` | holds |
| `pkg/ui` may name no simulation type | `pkg/ui/flow.go`'s own headers | holds, and FR-10 is written to it |
| the town's whole state is five fields plus the campaign | `pkg/game/town.go`'s `Town` doc | holds; FR-4 carries exactly those five |

## Stale summary corrected

`pipeline/subsystems/audio-saveload.md` was written before 0142 landed. Two of its statements no
longer hold and are not relied on here:

- "there is no campaign layer above the map at all: no money, no roster that survives a map change"
  — `pkg/game/town.go` has held gold, the won set, the offer lists and the taken set since 0142, and
  `FrontEnd.Carried` has held the roster across a map change since before it. FR-4 is written over
  what is there now.
- "`formatVersion = 25` at `pkg/sim/binary.go:336`" — it is 41 at `:706`.

Its save/load half is otherwise accurate, and its "what would close the gate" items 1 and 2 are this
story's FR-1 and FR-5.

## Open

- The brooch button for LOAD GAME (above).
- Whether a save should be refusable for naming a mission this install does not ship. It is refused
  today by the opener that cannot address the mission, with that opener's own sentence; a purpose-
  written message would be better and is not this story's.
