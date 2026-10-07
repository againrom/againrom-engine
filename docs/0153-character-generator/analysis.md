# 0153 — character-generator analysis

## Classification

Intensity is **spec-anchored / static**. Statistics, the selected skill and the starting weapon
cross the generator-to-party boundary and seed hashed simulation state. Name and presentation stay
outside the canonical world form.

Terrain is brownfield in `pkg/ui` and `pkg/game`. The transient preview adapter is greenfield, while
the Card must reuse the brownfield production sheet presenter. `pkg/data`, `pkg/mapload` and the
existing bitmap readers provide
existing behavior and are reuse-only unless planning finds a missing total adapter.

## Architecture

`pkg/ui` owns the 640-by-480 virtual frame, input dispatch, hit testing and drawing. Its import grant
does not include `pkg/data`, archive formats or `pkg/game`. `pkg/game` owns the install-backed setup,
definition tables, party construction and archive reads. It may pass plain values and decoded images
to `pkg/ui`.

`pkg/data` owns the point-cost curve, skill names, class-and-sex base rows, starting equipment and
figure draw order. `pkg/mapload` owns the pure party-spawn derivation used before a world exists.
The existing in-game card is composed by `ui.RenderPanel` from a `ui.PanelSubject`; the existing
doll is composed from a party member's figure, face and worn set.

This direction permits a live preview without a preview world. Moving archive reads or character
arithmetic into `pkg/ui` would add a second source and violate the import graph.

## Existing generator

`pkg/ui/chargen.go` is a pure model. It currently holds three cycling rows followed by four
statistics. It opens from supplied start indices, changes one statistic only when its inclusive
bounds and total budget permit the candidate, reports the remainder, and returns a copy of the
current legal result. The class row selects one of two five-name skill lists while retaining the
skill position.

`pkg/game/chargen.go` supplies sex, class and skill in that order, followed by Body, Reaction, Mind
and Spirit. It supplies the cost table through the declared ceiling and maps a confirmed result
through `ChargenParty`. That path selects the class-and-sex base row, trains one visible skill,
resolves its starting weapon, fills the worn set, and selects the figure directory and face.

The application screen is one keyboard-only diagnostic list. `stepChargen` applies Unicode text,
Backspace, row movement, adjustment and Enter. `drawChargen` uses the debug font and has no pointer
targets or source images. Name editing limits the UTF-8 byte length rather than the stored game-byte
length. Confirm validates only the statistic spread. An empty result name is replaced with `Danath`
while the party is built.

The picker creates a fresh model whenever a mission row enters generation. Escape returns to the
screen that armed generation. A successful callback opens the selected mission through the same
front-end entry path as other mission starts. The redesigned flow must insert pre-create and detail
inside that one gate without letting either stage reopen or retarget the picker row.

## Existing preview seams

`ChargenParty` creates a complete one-member party without creating a mission or world.
`mapload.PartySpawnWithTable` derives the combat values and pools that mission start uses. The
in-game character projection already copies the same hero statistics, skills, experience,
protections, sight and weapon into `ui.UnitCharacter`. A generator adapter can therefore build a
`PanelSubject` without repeating the arithmetic.

`composeUnitFigure` already reads the class-and-sex figure directory, face, body list, worn set,
secondary layers, mage-cloak rule and held-item order. It returns a 160-by-240 image or no image when
the base cannot be read. Reuse prevents the generator and in-game doll from differing in dress or
layer order.

The in-game panel is a framed 300-by-273 native render. Fitting it into a narrower generator box
made its source-font glyphs unreadable and was rejected. The owner requires one shared production
component across normal gameplay and generation, including model, statement, font conversion,
bounds and frame. The generator therefore moves the centre surface and copies that component at
native pixels. Parity covers the complete framed image for the same subject and 19 context-free
statement rows. The unplaced preview states `CELL -, -`; the confirmed player states its live cell.

## Install-backed controls

A read-only entry census over both preserved installs found the localized statistic plate at
`main.res::graphics/chrgen/leftup.bmp`. The pictorial controls are in `graphics.res`, not
`main.res`: `interface/chrgen/fighter`, `interface/chrgen/mag`, and
`interface/chrgen/precreate/heroes`. Pre-create's 640-by-480 mask gives disjoint hot pixels for the
four overlapping portrait patches. The ten skill families each carry `on`, `shine_off` and
`shine_on`; the four combined class-and-sex families carry `on`, `l` and `lon`. Fighter and mage
each also carry a 320-by-480 `column.bmp` plus a same-sized indexed hit mask. The column is the
central surface; the individual skill images are native patches for its five pictured positions,
not five full-width row backgrounds.

Direct visual comparison established two placement corrections the earlier plan missed. The native
`buttonok.bmp` belongs at `(468,373)` over the lower `BOOK` marking and mask index 180 activates it;
the far-right ornate brooch is mask index 160 and is Back to the main menu. Pure-black rectangles in
the state patches are keys, not visible pixels. Their non-black rectangular backgrounds remain part
of the art.

The same lawful resource census names `amulet.bmp` as the 112-by-204 brooch highlight and ten
20-by-20 statistic-button images under `interface/chrgen/buttons`. A byte-level read of the cited
statistic constructor and draw routine establishes value/minus/plus rectangles at x
`82..102/107..127/132..152`, rows separated by 32 pixels from y 54, and the remaining-value
rectangle `[46,123)×[181,203)`. The source column's complete fighter and mage hot extents fit inside
source x `72..252`, so that crop can move to destination x `300..480` without resampling or losing a
control. This frees `[0,300)×[207,480)` for the native shared Card.

The same census found `main.res::text/main.txt` and the compact fonts in `graphics.res` at
`font2/font2.{16,dat}` and `font3/font3.{16,dat}`. The two installs share control paths but differ in
the localized string bytes and statistic-plate pixels. Automated tests must synthesize these nodes;
release evidence may read the lawful installs.

## Name and continuity

The current party record carries `Name` as per-character presentation state. The save envelope and
campaign transition preserve it. It is outside `sim.World` and does not change the world digest, so
this story needs no byte-form version.

The original input boundary stores one through ten converted bytes. Windows-1251 input is converted
to the engine's stored byte space for selector 1; ASCII is unchanged. Final acceptance refuses an
empty name and ASCII-case variants of `Self` and `Computer` before multiplayer predicates. This
project has no multiplayer participant population.

## Planning risks

- A preview must allocate no world and mutate no party, clock, purse, save or random state.
- One class change must replace all five skill controls while retaining the selected position.
- The selected skill must be the sole visible level-10 slot and must select the shown weapon.
- Card and doll must be rebuilt from the same draft on every accepted change.
- A pointer gesture must belong to one drawn rectangle and must not reach a destination screen.
- Stored-byte conversion must happen before the ten-byte capacity check.
- The lower-left Card must remain pixel-identical to normal gameplay, including native frame and
  glyphs; no later hover/refusal text may overwrite its rectangle.
- A click sequence across all five skills must leave exactly the Draft's index selected and restore
  every prior patch to rest.
- Both preserved roots must resolve every required source string, control image, font and doll layer.
