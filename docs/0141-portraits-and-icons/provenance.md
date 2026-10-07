# 0141 — provenance

Research pin `cdfe33d`. The pin moved mid-story, from `2bb2c76`, and that is the one deviation from
freeze discipline worth stating: two experiments this story had already disclosed as blocking landed
on research master while the branch was open, and the last commit bumps the pin and takes both.

## What the picture is, and which arm an actor takes

| Claim | Confidence | What this story took from it |
|---|---|---|
| `UNIT-PICT-035` | High | Compose-or-load is ONE test on the class — a type id below `0x1a` sets the bit both picture builders test, at or above it the field stays zero. The two arms are exclusive. Also: the thirteen human classes' `InfoPicture` is dead data. |
| `UNIT-PICT-036` | High for both formatters and the tier-1 truncation; **Medium** that these are the only two consumers | `graphics\infowindow\<InfoPicture>.bmp` for the info panel with no tier, `<InfoPicture><tier>` for the dialogue panel with the digit NUL-ed at tier 1. The two addresses differ on every class that ships four pictures. |
| `UNIT-PICT-037` | High for the field and the leaf; Medium for which window paints it | A structure's picture is the same leaf off `structures.reg`'s `Picture`, also with no tier — which is why `PortraitPath` is a formatter of its own and not a special case of the tiered one. |
| `UNIT-PICT-038` | High | The G2 limits carried as named constants: name ≤ 15 + NUL, 160x240 compiled in at twelve sites, face ≤ 127, four figure directories fixed by a jump table, and the tier digit unbounded in the formatter. |
| `SPR256-PICT-043` | High for the census; **Unknown** for the two trailing bytes on 85 of 86 nodes | The tree is 24-bit uncompressed Windows BMP, `dataOff=54`, `nClrs=0`, 160x240 on 86 of 86 EN and 81 of 81 RU nodes — the whole contract of `pkg/formats/bmp`, including its refusal to trust the header's own `imgSize` (0 on 85 of 86) or the file's length. |
| `REG-PICT-083` | High for the counts and the two set comparisons; Medium for `Palette == files` | Thirteen unit classes name a picture that exists at no tier; the RU root ships thirteen structure sections naming a node `GRAPHICS.RES` does not have. Both are why every load path here treats absence as ordinary and never as an error. |

## The person's own figure

| Claim | Confidence | What this story took from it |
|---|---|---|
| `DAT-HUMANS-008` | High | Slot 18 is the gender cell, read by the streamer into a local and dropped — and re-read by the constructor, which stores `gender + 0x21` for a fighter or `+ 0x23` for a mage over slot 16's type id. |
| `DAT-ACT-006` | High | The same overwrite from the actor side, on both spawn streamers. |
| `ALM-CLS-054` | High for the store and its gate; Medium for the end-to-end identity | That overwrite is what makes `[0x20,0x40)` the human range on the wire. |
| `HERO-APPEAR-041` | High | Inside that range the client banks `typeID - 0x21` — bit 0 the sex, bit 1 the fighter/mage axis — and forces the drawn class to 1. This is the arm a placed person takes. |
| `HERO-CLASS-013` | High, **amended**, and one clause **retracted** | The sex test is `typeID ∈ {0x22, 0x24}`, which fixes 1 as the female addend. The retracted clause is the provenance sentence that `+0x4c` is copied out of the `Data.bin` definition; `EXP-0133` overturned it — the bit is derived at spawn from a positive `ManaMax`. That does not touch the addend/base split this story uses, and it is why the corroboration below is stated as mana rather than as a column. |
| `HERO-DOLL-078` | High | The four figure directories `mfighter`/`mmage`/`ffighter`/`fmage` selected by those two bits, and the `<dir>\<face>.256` leaf. |
| `SPR256-EQUIP-042` | High | The four directories are stocked 31, 13, 10 and 5 faces deep — unequally, which is what makes "every placement lands on a shipped sheet" a test rather than a tautology. |

## The speaker

| Claim | Confidence | What this story took from it |
|---|---|---|
| `DLG-NPCTAG-018` | High for the subscript path; Medium for the 59/59 census | The `<npc=N>` number reaches the npc array unmodified — the tag names an `npc<n>` SECTION of `scenario.res::npc.reg`, not a placement. |
| `REG-NPC-088` | High for the two stores and their flag gates; Medium for the 33/33 corpus join | `Picture` is stored into a synthesised dialogue actor's typeID and `Face` into its face byte, so the portrait that follows is `UNIT-PICT-036`'s formatter — and `Face` is therefore the TIER DIGIT on the portrait arm and the sheet number on the figure arm. |
| `REG-NPC-089` | High for the `RECT` identification, the field order and the dead pair; Medium that `R0740` is the only reader; **Unknown** whether `X2`/`Y2` were ever live | `PortraitX1`/`Y1` is a per-speaker window origin; `X2`/`Y2` are never read again. The window is a fixed 72x96, the absent-key default is the literal 72x92 at visual rows 8..100, and the destination is the bare point `(8, 7)`. |
| `REG-NPC-091` | High for the bitmap census and for the two row orders being the only survivors; **Medium** for BOTTOM-UP being the live one | The window's origin is an ordinary top-down offset into the picture. This is the weakest load-bearing grade in the story and its falsifier is named in the claim: read the loader that fills the canvas, or watch one `npc132` dialogue on a running original. |
| `REG-NPC-058` | High for the domains; **Unknown** for what the keys index | The key census and the eleven `Flags` tokens with `!` negation. Its Unknown gloss is what left the face seam empty from 0079 until `REG-NPC-088` landed; the corrected `PortraitX1`/`Y1` counts (48 each) come from here. |
| `ALM-CLS-038`, `MISSION-ARM-006` | High | The support for the reading this story ELIMINATED — a placement's secondary key is the `npc.reg` subscript on the NPC arm. Real support, and still the wrong binding for a dialogue tag. |

## The icons

| Claim | Confidence | What this story took from it |
|---|---|---|
| `MAGIC-ICON-024` | High | `interface\SpellBook.bmp` is 480x85 and is blitted whole; an icon is a POSITION at `x = 6 + 38*(i%12)`, `y = 6 + 38*(i/12)`, side 36. The engine's only slot-to-spell mapping is a 24-entry table, transcribed verbatim, and ids 11, 17, 27 and 28 are in no slot. 24 is an engine limit stated twice. |

## Ours by choice

- **The tier reaches the info column.** `UNIT-PICT-036` is explicit that the original's info panel takes no tier. The owner ruled otherwise on 2026-08-11 having played the build ("an orc is green, red, blue, black"). What is reproduced is the address; which window uses it is ours. Disclosed at the site, in the build README, and again in the spec.
- **A tier a class has no file for falls back to that class's first picture.** The original addresses a node that does not exist. Three of the sixteen portrait classes need this.
- **The class axis of a placed person's figure is a range test on the type-id column** (`0x17..0x18`), used as a proxy for the `+0x4c` bit the engine reads. Corroborated twice — every one of the 57 rows carrying `0x18` has mana or a spellbook and none of the 153 rows below `0x17` has either, which is exactly what the `HERO-CLASS-013` retraction predicts; and the fifteen `npc.reg` person records stating a `Mage` flag agree 15 of 15.
- **The pane's destination point is pane-relative.** `REG-NPC-089` reads `(8, 7)` off the call and does not say what it is measured from. `8 + 72 + 8` is 88, the pane's width exactly; panel-relative would put the picture outside the pane altogether.
- **The pane's ground is black and opaque**, and the picture composites over it (owner, 2026-08-11).
- **Nothing selected shows nothing** (owner, 2026-08-11), and an actor whose picture the install does not carry falls back to its own drawn world frame.
- **The bar's cell is 38 px** so a 36x36 icon lands inside a one-pixel border unscaled. Nothing in this tree resamples a picture, so the cell follows the art.
- The archive spellings `infowindow/`, `interface/SpellBook.bmp` and the `.bmp` extension are conventions; the archive's own lookup is case-insensitive and separator-agnostic, so none of them asserts how the index writes the node.

## Open

- Whether the `infowindow` canvas is filled bottom-up is Medium (`REG-NPC-091`). Every window origin in this build rests on it.
- What the two bytes past the pixel run are on 85 of 86 portrait nodes is Unknown (`SPR256-PICT-043`). This decoder ignores them.
- `PortraitX2`/`PortraitY2` have no located reader and no established intent. Nothing here reads them.
- The empty gender cell. One campaign placement of 462 states none; the streamer's sentinel rule leaves the constructor's own default standing and that default is not established anywhere. This build draws a man.
- A speaker's picture and a placement's picture are two statements and the shipped data disagrees: of the fifteen `npc.reg` person records carrying both a `Face` and a `DataBinID`, the registry's `Face` differs from the `Humans` row's face column on five, and its `Female` flag from that row's gender column on one. Nothing here reconciles them.

## Removed

The 0140 doll-box fallback — a non-party unit's own drawn world sprite, shipped as a disclosed
substitution because no published claim said where a monster's doll art lived. `UNIT-PICT-035` says
the fallback was the wrong SHAPE as well as the wrong picture: a monster has no doll. The world
sprite survives as a third arm and is now what a missing FILE falls back to.

The spellbook cell's three-letter abbreviation as a placeholder. It stays as the fallback for the
four spells that genuinely have no icon.
