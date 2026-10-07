# Story 1304: Shadow writers and the translated pair

## Intent

Known defect B15, remainder of story 1296. Settle DIV-2067 (which drawables
take the hero sheet pair) and DIV-2068 (the translated arm's population and
overlap) against the new claims. No player-visible drawing changes: the
engine already matches every High claim, so the result is the proof and the
ledger.

## Authority

Pin k141. `TERR-194`, `TERR-195`, `TERR-196`, `UNIT-140`, `UNIT-141`, and the
amended `TERR-192`, `TERR-193`, `ANIM-117`, `PARTY-FLAG-003`, `REG-UNITS-061`.
B1 holds: the research questions carried no expected answer.

## As-built behaviour

- Hero pair. A drawable takes the hero sheet pair and the sheared arm when its
  type id is in `[0x20,0x40)`: the party's player characters, and a placed
  actor made by a Humans constructor mode of 1 (`UNIT-140`). On the installed
  maps that is 4 npc placements carrying `Hero`, over missions 1 to 160 on both
  roots; they draw the composed hero body and its paired `spritesb` sheet
  (`rosterBodyArt`, `missionAppearanceArt`). The other 2,329 non-party actors
  draw their class sheet pair. Hired mercenaries and table-driven Humans keep
  their class sheets: no table row holds an id in the hero range.
- Translated arm. Only a class with nonzero `Z` (`Sonic Bat`, `Dragon`) is a
  `CAirUnit` (`UNIT-141`); its two silhouettes are translated, never sheared.
  Each is placed from its own sheet size. The engine's placement overlaps on
  none of the 273 frames of those two classes, mirrored or not, so the two
  silhouettes darken nothing twice. The blend recolours in place and keeps no
  coverage buffer (`TERR-196`), as the engine's shared blend does, so an
  overlap, where one exists (the sheared pairs of unequal sheet sizes,
  `TERR-192`), darkens twice.
- No code under `pkg/` changed. `flat` in `pkg/ui/shadow.go` still reads
  `PlayerCharacter` as well as the air category; the claim's guard is the hero
  id range, and no shipped placement differs.

## Proof

- `pkg/game.TestReleaseUnitShadowHeroPairFollowsTheHeroTypeID`: starts missions
  1 to 160, resolves every non-party actor's art, and requires a hero body with
  a paired frame for each of the 4 hero-range actors and a class sheet for the
  others. EN and RU give the same counts.
- `pkg/game.TestReleaseUnitShadowTranslatedPairNeverOverlapsOnTheInstalledArt`:
  places both silhouettes of every frame of the `Z` classes through
  `terrain.UnitPlace`, as the shadow layer does, and counts shared pixels:
  0 over 273 frames (134 with unequal sheet sizes), mirrored and not. The
  control, a one-pixel shift of the second silhouette, overlaps 26,655 pixels.
  EN and RU give the same counts, and the counts equal the claim's.

## Open debt

- DIV-2067 stays open, narrowed: the constructor mode at the primary-hero
  command sites, the `AddHero` sites and `L12812` is untraced; a summoned
  unit and a unit a load rebuilds through the creation path were not observed.
  The engine keeps its current behaviour for each. The speaker synthesizer and
  character-screen drawables set bit 0 but were not shown to be drawn on the
  map; they are not modelled.
- DIV-2068 is closed. The stale-global edge of `UNIT-141` is a research
  Unknown with no engine difference.
- DIV-2129 to DIV-2132 were reserved and none is used.
