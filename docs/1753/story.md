# The second game's Poison Cloud, screen-change music and world map text follow the original

## Result

In the second game the poison a Poison Cloud puts on a unit now lasts the same short time whatever the
caster's power, and each new pulse or cast sets it back to that time instead of adding to it. On the world
map a place's name and description lines are centred over the text area, and the name is drawn in font4,
as in the original. The music at each screen change, the startup movies and the save label already behaved
as the original reads and are now pinned by tests. The first game is unchanged.

## Authority

- Owner: «Да, запускай» (epic «ROM2: полная игра»): adopt EXP-2191, EXP-2195 and EXP-2196.
- R2-ENGINE-1575 (High / Medium), R2-ENGINE-697: the Effect lasts the Effects count times 16 (128) at any power.
- R2-ENGINE-1576 (Medium): a repeat cast keeps the larger duration, never a sum.
- R2-ENGINE-1577 (Medium), R2-ENGINE-998 (High / Medium): the Shield self-cast draw and the item card stat line.
- R2-ENGINE-1636, 1640, 1642, 1643 (High), 1639, 1641 (Medium): each screen change stops the track and starts
  the next screen's own track with no fade; a town gate exit plays no movie.
- R2-ENGINE-1653 (High / Medium): the map text draws its name in font4 and its lines in font2, centred on x 230.
- R2-ENGINE-1650, 1652 (Medium; High for the strings and the stores): the logos order and the mode-2 label.
- DIV-4287, 4835, 2862, 2863, 2866, 3070, 3081, 4098 updated; DIV-5114 written.

## Rules

1. When a second-game Poison Cloud lands its Effect, the Effect lasts its Effects count times 16 (128), at any power.
2. When a second-game Poison Cloud lands again on a unit that holds it, the duration is the larger of the two.
3. When a mod gives a second-game Poison Cloud a duration table, the mod is refused with the reason.
4. When a second-game computer mage casts Shield on itself, the only draw up to its recovery is the 0..3 recovery draw.
5. When a second-game item card names Heal, it has no stat line; Darkness and Light state line 184.
6. When the second game changes screen, the old track stops, the next starts with no fade; a gate exit plays no movie.
7. When the second game's map shows a place's text, the name draws in font4, the lines in font2, each centred on its x.
8. The first game's Poison Cloud, Shield draws, captions and world map do not change.

## Code

| kind | its one builder | change |
|---|---|---|
| spell row loader (effect duration) | `pkg/mapload/spell.go:205` spellRules | the count times 16 in both games; no game input |
| effect duration on a cast | `pkg/sim/spellvalues.go:234` spellPointDurationUnder | lastingRule drops the second game's area row flag |
| effect renewal | `pkg/sim/effect.go:446` attachEffectTransaction | none; pinned |
| computer self-cast draws | `pkg/sim/autocastsetting.go:189` supportCast | none; pinned |
| spell and item popup caption | `pkg/sim/spellarms.go:281` SpellCaptionOf | none; pinned |
| music track at each transition | `pkg/ui/music.go:273` requestFrom, `:228` musicScene | none; pinned |
| map text block | `pkg/game/worldmapstate.go:283` view, `pkg/ui/worldmap.go:479` drawWorldMapText; font word `pkg/game/secondworldmap.go:224` worldMapFont | lines centred; name font from the text row's `name-font` |
| startup movie list | `pkg/base/edition.go:305` StartupCutscenes | comment cites the read |

## Proof

| test | rules |
|---|---|
| `TestPoisonCloudEffectLastsItsEffectsCount`, `TestSecondPoisonCloudAndInvisibilityTakeTheirOwnFormulas` | 1, 8 |
| `TestSecondGamePoisonCloudRenewalAppliesNothing`, `TestSecondGameRenewalKeepsTheLargerMagnitudeAndDuration` | 2 |
| `TestSecondGameFormulaRefusalsNameTheRowFieldAndLimit` | 3 |
| `TestComputerMageSelfShieldDrawsOnce` | 4, 8 |
| `TestSecondGameScrollCaptionStatLine` | 5, 8 |
| `TestSecondGameScreenChangesStopAndStartWithoutAFade` | 6 |
| `TestWorldMapTextBlockCentresEachLine`, `TestSecondMapTextNamesItsFonts` | 7, 8 |

Release tests at the release: `TestReleaseSecondCampaignWorldMapSelectMode` and
`TestReleaseSecondCampaignWorldMapTravelKeepsText` on the EN and RU second-game installs (rule 7).
Evidence: `review/story1753-rom2-presentation-adopt/RESULT.md`, `RESULT2.md`.

## Open debt

- DIV-5114: a second-game item that casts Slow states the Speed line, which the original's card does not.
- DIV-2863: a completion movie's destination track is set and paused before the movie and resumes after it,
  where the original sets it after the movie.
- The map text ink colours, the wrap body, the screens of modes 0, 1 and 3 and the audible results stay
  unread; the original runs named in DIV-3070, 3081, 4098 and 2863 settle them.
