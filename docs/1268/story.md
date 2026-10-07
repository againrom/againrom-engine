# 1268 - remaining special tooltips

## Result

Re-audit of every hover target the six claims name against the tree at
`a9e39a18`. One bound target was marked unreachable in `DIV-1270` and is not:
the monster spell list on a creature's information card. Placed creatures with
class spell columns carry a nonzero `KnownSpells` bitmask and class spell slots
(`pkg/mapload/spell.go`, `unitSpellbook`), so the card states the SPELLCASTER
caption and the hint shows. This story proves it end to end on both installs
and corrects the row. No other claim-established target is both unbound and
fully specified; each remainder is listed below with the reason. No simulation,
save or hashed state changed.

## Authority

TEXT-HOVER-048, TEXT-HOVERSET-049, TEXT-HOVERPAINT-053, TEXT-HOVERCHAR-050,
TEXT-HOVERROOM-051, TEXT-HOVERTEXT-052, UNIT-SPELL-007.

## Inventory

Engine state at `a9e39a18`. Bound rows cite the production function.

| claim | target | state | this story |
|---|---|---|---|
| 048, 053 | shared 500 ms crossing, 25 s life, font2 box, lower-left anchor | bound (`tooltipController`); right-held and button-edge timing and wrapping differ by design, `DIV-1270` | none |
| 050 | card helper main155-170/171-180/188 | bound (`panelTooltipSlot`) | none |
| 050 | generator getters main247-256, 273, skill pictures | bound (`chargenTooltip`) | none |
| 050 | attribute +/- and value rectangles, live value | prose only | Unknown: the claim gives no join format |
| 051 | town mask main233-237, school 171-180, shop 54-65/74, character corner 8-14/52-53/121-122, tavern card and worn items | bound (`tooltipTargetWithSurface`, `characterCornerTooltip`, `shopTooltip`) | none |
| 051 | mission inventory bar gold, main74 | no money cell exists in the mission pack | not applicable |
| 052 | world-map site text | bound | none |
| 052 | map-list row text and size, dialogs.txt[134] | bound (`pickerTooltip`) | none |
| 052 | map-list dialogs.txt[135]/[136] | unbound | no per-map source in `pkg/formats/alm` |
| 052 | spellbook popup, main117/118/123/124, availability bit | bound (`spellInfoLines`) | none |
| 052 | spellbook popup, main182-187 and 217 | unbound | Unknown: the claim names the captions (Speed, Resistance, Sight, Maximum Damage Probability, rays, Minimum Damage Probability, Absorption) and "present live fields", and gives neither the field each caption reads nor its units. Binding would assign fields by guess, so none is bound |
| 052 | monster spell list, main192 + spell.txt | bound; reachable on mission 51's ogre turtle | witness added, row corrected |
| 049 | 21 specialized bodies | covered by the rows above | none |
| 049 | inherited getter and its hint setter | unbound | Unknown: no setter or constructor binding recovered |

## As-built behaviour

No production code changed. A creature that holds class spell slots paints the
installed SPELLCASTER caption on its card; hovering that row for the
configured delay shows the installed main.txt[192] heading followed by the
installed spell.txt name of each known spell, ascending spell ID, at the
common hint placement. Moving off the row hides it. A creature with no spell
slot paints no caption and offers no target.

## Proof

- `TestReleaseTooltipMonsterSpellListOnCreatureCard`
  (`pkg/game/monsterspellhover_release_test.go`): opens mission 51 through the
  player's opener, selects the ogre turtle, finds the spellcaster row through
  the production pointer, checks the hint is hidden before the delay and
  visible after it inside the frame, that its text is the installed heading
  plus the installed names of the turtle's slots, that moving off the row hides
  it, and that a plain creature's card offers no such target. Registered in
  `internal/gatedtests/testdata/population.txt`; runs on EN and RU.
- Screenshots: `review/story1268-special-tooltips/{en,ru}/` (maplist-hint.png, monsterspell-hint.png), produced offscreen by
  `cmd/tooltipshot`, run with `-mission-limit 60`; it finds the turtle (mission 51, entity 12, known spell Acid Stream in EN) by its own scan of missions.

## Open debt

- Spellbook captions main182-187/217: bind when a claim gives each caption's
  field and units.
- Chargen attribute live value: bind when the join format is recovered.
- Map-list dialogs.txt[135]/[136]: a per-map source is not decoded.
- Inherited getters: need recovered setter bindings.
