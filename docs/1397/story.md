# A mod chooses the body a weapon is drawn with

## Intent and authority

A mod draws a weapon with another hero body (the War Hammer with the mace
body), and supplies a body sheet of its own. The engine gains the feature; no
mod ships. Owner direction: the mod plan entry "a mod draws a weapon with
another hero body, then a mod-supplied body sheet". The game without mods draws
every weapon as before.

ROM1 authority for the unmodded game, unchanged: `HERO-APPEAR-052` and
`ANIM-108` (the body is entry D - 1 of the 26-entry `heropicture.txt`),
`HERO-FIGURE-060` (the held-layer paint order is a six-name predicate on that
name), `HERO-APPEAR-042` and `REG-UNITS-050` (the forced dying bodies).
Differences: `DIV-2912`, `DIV-2913`, `DIV-2914` in `docs/divergences/mods.md`.
The file formats are in `docs/MODS.md`.

## As built

Two data files, loaded with `game.data.add` (`pkg/mod/bodies.go`,
`pkg/modrt/data.go`, `modrt.Result.Bodies`):

- `data/weapon-bodies.toml`, `[[weapon]]`: `weapon` (row name), `row`, `body`.
- `data/bodies.toml`, `[[body]]`: `name`, `heroes`, `heroes_l`, `frame`,
  `origin`, `directions`, `move`, `attack`, optional `idle`, `weapon-last`,
  `selection`.

`FrontEnd.SetModBodies` (`pkg/game/modbodies.go`) runs after `SetMods`. It
checks every choice and sheet against the install, builds each supplied body
under both directories through `composeHeroBody`, the one hero body builder a
shipped body also takes (`pkg/game/heroart.go`), and adds them to the front
end's unit bundle. The choice reaches every reader of the body list through
an overlay entry of the front end's own filesystem (`againrom/mod-bodies.txt`,
read by `ReadBodyList`), so the mission, inventory, town and figure composers
need no new argument. `mapload.ModContext.Bodies` carries the same list for the
save writer. `cmd/againrom -check` prints `weapon bodies=N body sheets=M`.

`data.BodyList` holds the shipped entries, the weapon choices and the
supplied bodies. `data.HeroAppearance` answers the drawn name from the choice
and the class key from the shipped entry. A row the shipped list leaves blank
stays blank. The `applies-to` list decides where a mod loads; ROM2 reads its
own `heropicture.txt` (the same 26 names) and its own sheets. A mod for both
games lists `rom1`, `rom2-en` and `rom2-ru`.

### Consumers of the body

| consumer | with a choice |
|---|---|
| hero sheet and geometry: `LoadHeroBody`, `partyArt`, `rosterBodyArt`, `refreshAppearance`, `missionports` | the drawn body |
| party member `Body` in memory (`canonicalizePartyAppearance`, `syncJoinedHeroes`, `MissionParty`, campaign return, town screen, chargen, original party) | the drawn body |
| doll and map figure paint order (`FigureHeldLast`, `FigureDrawSteps`; inventory, figures) | the drawn body's six-name test; a supplied body's `weapon-last` |
| class key: entity `Class` (hashed), `UnitNameIndex`, `FindHumanByType`, potion and siege-hire blocks, `ComposesFigure` | unchanged, the shipped entry's |
| swing sound, attack delay, cast and shot class (`spellClientClass`) | unchanged, the class key's |
| fallen hero body (`HeroBodyName` dying arm, corpse link) | unchanged, the shipped dying body of the directory |
| SAV party policy `Body` (`capturePartyPolicy`) | unchanged: `BodyList.SavedBody` writes the shipped entry's name |
| bones art of a placed roster hero (`missionAppearanceArt`) | a supplied name matches no class and takes the equipment path, drawn with the supplied body |

## Proof

- `pkg/data`: a choice changes its own row only and keeps the class key; a
  blank row resolves nothing; held-last follows the drawn body; a supplied
  body's shield form; `SavedBody` returns the unmodded name for every row
  with and without a shield.
- `pkg/mod`: both files, every key and every refusal with its line.
- `pkg/modrt`: the example mods load on the four bases; refusals; two mods
  cannot supply one body.
- `pkg/game` (fixtures drawn by code): a supplied sheet is cut so every one of
  its 88 frames is reached by the frame selectors; the choice round-trips
  through the filesystem; install refusals name mod, file and line.
- Release, `pkg/game/modbodies_release_test.go`, run on `ru`, `en`, `rom2-ru`
  and `rom2-en`, all pass. The example mods are under
  `pkg/modrt/testdata/mods/`; the test draws the PNGs into a temporary copy.
  - `TestReleaseModBodyDrawsTheWarHammerHero`: opened wielding the War Hammer
    and equipping it in mission 10, the hero is drawn with `axeman2h` without
    a mod, `clubman` under `war-hammer-mace` and `hammer2h` under
    `war-hammer-body`; the entity class equals the unmodded one.
  - `TestReleaseModBodyLeavesWorldAndSaveUnchanged`: the same session (hammer
    worn from the start, or equipped; a strike; 40 ticks) gives equal World
    hash, world bytes and, on ROM1, mission SAV bytes with and without each
    mod's body data, while the drawn body differs. ROM2 writes no SAV yet.
  - `TestReleaseModBodyRedrawsAfterALoad` (ROM1): a mission SAV written under
    the mod loads cold under it and draws the mod's body again.
- Existing mod and hero figure release tests (`^TestRelease.*Mod[A-Z]`,
  `TestReleaseFigureRegression*`, the hero type and fallen body tests), `ru`
  then `en`: pass.

## Open debt

- A supplied body has no owner shading, no `spritesb` silhouette, no wind-up,
  dying or bone block; its other registry fields come from class 1
  (`DIV-2914`).
- Sound, attack timing, cast and shot follow the shipped class key. A mod
  that wants a weapon to fight as another body changes a hashed rule and needs
  an owner decision.
- `applies-to` has no word for both ROM2 editions; a mod lists `rom2-en` and
  `rom2-ru`.
