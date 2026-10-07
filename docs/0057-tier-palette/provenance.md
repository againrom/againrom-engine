# Provenance — the tier is a file, not a formula

Intensity: **spec-anchored / static**. Terrain: **brownfield** in `pkg/game` and
`pkg/render/terrain` (a landed unit-draw path gains an input), **greenfield** for the `.pal`
decoder.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — 256 entries of `[B,G,R,X]` at offset `0x36`, `X` never read | `PAL-FILE-001` | High |
| FR-1 — the shared-owner file is a different shape (`0x4000` whole, no seek, first two bytes `00 00`) and is excluded rather than fitted | `PAL-FILE-001` | High |
| FR-2 — the name is BUILT: the sprite entry's directory + `palette` + the tier's digit for tiers above 1 + `.pal`, and tier 1 has no digit | `PAL-NAME-003` | High |
| FR-3 — the tier count is the `units.reg` `Palette` key; 0 on 18 classes, 1 on 3, 4 on 13 | `PAL-KEY-002` | High |
| FR-4 — no closed form reproduces a tier's colours from tier 1: per-channel affine, a general 3x3 matrix with offset, an HSV rotation and an index remap each fail at their own optimum, so a consumer must read the file | `PAL-XFORM-006` | High |
| FR-5 — tier 1's file is byte-identical to the sheet's own embedded palette (13/13 classes), so tier 1 is not a recolour | `PAL-XFORM-006` | High |
| P-3 — palette entry 0 is identical across every tier of every class (39/39) and is the transparent key | `PAL-XFORM-006`, `SPR256-PAL-012` | High |
| FR-7 — the tier is the actor's `face`, the definition table's Units column, reaching the drawable and subscripting the class's palette array | `PAL-FACE-005` | High |
| FR-7 — every table row whose class key resolves to a unit class carries a `face` inside `[1, Palette]`, 0 violations on every root | `PAL-CORP-008` | High |
| FR-5, FR-9 — the tier's whole colour treatment is that shipped palette: the SAME builder, arguments and blit lookup every other sprite already uses, and no arithmetic distinguishes a tier | `PAL-TIER-004` | High |
| FR-5 — the builder those arguments name is the 16-row mode-2 ramp built from the sprite's own palette, indexed once per sprite by the sun's ambient byte | `TERR-LIGHT-059`, `TERR-LIGHT-060`, `TERR-LIGHT-064` | High |
| FR-3 — the four-tier ceiling is a structure-width fact of the engine, and what lifting it costs is a shipped `paletteN.pal` node plus a registry and a table row; the wire field already reaches 63 values | `PAL-LIMIT-009` | High |
| AC-8 — every palette node this story reads is byte-identical across all three roots | `PAL-CORP-008` | High |

The builder in `PAL-TIER-004` is the one this tree already has: `0044-sprite-lighting` landed
`SpriteChannel`/`SpriteRow` from `TERR-LIGHT-059/060/062`, and the claim's arguments
(`nLevels = 0x10`, mode 2, `useTint = 1`) are that ladder's own. Nothing about the ramp moves here;
only which 1024 bytes go into it.

## Ours by choice

| Choice | Why it is ours |
|---|---|
| The decoder refuses a stream whose first two bytes are not `BM`. | The engine seeks and reads and checks nothing. The magic is the discriminator research names between the two shapes, and refusing on it turns the shared-owner file from a silently wrong read into a named refusal. |
| A tier is a second **frame slice** over the sheet's own pixels, not a second blit path. | The palette already rides on a frame; the claim says the tier feeds the existing builder different bytes. A palette parameter threaded through the blit, the texture cache and both renderers would be a path beside the one that works. |
| A tier's frames share the base frames' pixel memory and copy only the colour table. | The pixels are identical by construction — same sheet, same frame — and everything downstream reads through the pointer and writes nothing. |
| Where a tier's colours equal the sheet's own, the tier IS the sheet's own frame slice, pointer for pointer. | Two identities for one picture would upload two textures for one image and make "these tiers are the same colour" unobservable downstream. |
| The tier ceiling is enforced as a clamp on a named constant rather than as a refusal. | The engine's own overrun past four is undefined behaviour we decline to reproduce; clamping draws the class rather than failing a run on registry data. |
| A class whose art is substituted for drawing takes the substituted class's own tiers. | Research separates no corpse arm here. One rule over whichever class supplies the frames invents less than a second rule saying corpses are never tiered, and it is total either way. |
| The still is a contact sheet: one frame index, the class's own, drawn once per tier left to right on a neutral field. | Every panel then differs in colour alone, which is the whole question the picture is asked. |

`PAL-FACE-005`'s byte reaches our drawable a different way from the engine's: there is no client
state message here, so the tier is read from the definition-table row a placement resolves to, at
map open, and carried beside the world. The number is the same column; the path is ours.

## Divergence, disclosed

- **Team colouring is not implemented.** `PAL-OWN-007` is the `Palette == 0` arm — 18 of the 34
  classes, every human and hero — and this build draws those classes in their sheet's own colours.
  Two things stop it being a short step rather than a choice to hurry: the claim's own
  identification of `k` as the owner's colour slot is **Medium**, resting on the sixteen
  sub-palettes' shape because no writer of that record's `+0x08` was located; and this tree's
  simulation carries **no owner at all**, so there is nothing to subscript with. Named as a
  follow-up in the spec's Out of scope.
- **The shadow pass does not exist here**, so the greyscale seventeenth table
  (`(0x10, mode 5, useTint 0)`) has no consumer to be right or wrong for.
- **`palette_.pal`** ships beside every `palette.pal` and matches no name the loader builds
  (research Open). It is not read.

## Open — deliberately assigned no meaning

- Which shade object the engine's shadow arm selects (`L10418`/`L10387`), and whether it is
  used at all — research Open.
- The writer of the owner record's `+0x08`, the one store that would raise `PAL-OWN-007`'s
  selector above Medium.
- `units/heroes/human.pal` and `units/heroes_l/human.pal`: two further 16384-byte blobs no located
  call site opens.

## Removed — what a reader might expect and this story does not assert

- **That a tier's colours can be predicted.** They cannot, and the spec forbids the attempt rather
  than leaving it merely unimplemented.
- **That tier 1 is a recolour.** It is not: tier 1's file is the sheet's own palette, and the
  spec's equal-palette clause is what makes that observable instead of asserted.
- **That the tier means anything to the simulation.** It reaches no world field, no byte form and
  no digest; it is presentation state born with a map and dropped with it.

## Not consulted

`PROJECT/`, `PROJECT/cleandocs`, the standalone research tree, any third-party source. Research
facts come from the submodule at its pin.
