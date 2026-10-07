# Story 1301: hover help and the mission screen

## Intent

Adopt k137 (EXP-0454) and k138 (EXP-0455) for the known defects B7 (hover help)
and B9 (mission screen), change the engine where a High part of a claim is
visibly contradicted, and state the rest as divergence rows.

## Authority

Current pin: k159 (`769071b656873de6a24a4cdaa312da54a9fdab29`). The evidence
origin is k137 (EXP-0454) and k138 (EXP-0455). B1 holds: the earlier open questions of rows `DIV-200`,
`DIV-201`, `DIV-284`, `DIV-286` and `DIV-1882` to `DIV-1885` are answered by
these claims.

- B7: `TEXT-096`, `TEXT-097`, `TEXT-098`, `MENU-066` to `MENU-069`.
- B9: `MENU-070` to `MENU-072`, `MISSION-063` to `MISSION-066`,
  `UNIT-STRUCTUSE-137`, `UNIT-STRUCTHIT-138`.

The owner accepted widget 8 replacing the `DIV-344` mission statistics card
while the structure readout is drawn. The ordinary card returns when no
structure readout is drawn. Its geometry, font and precedence stay the same.

## As-built behaviour

- Spellbook popup (`TEXT-096`, `TEXT-097`). `SpellCharacteristics` carries the
  record level, `skill + mind - 30` kept in a byte and clamped at 100, and the
  range and duration lines are filled at it. Damage is the low byte of the
  double products of `TEXT-096`, computed in `pkg/game` because `pkg/sim` may
  hold no float. The damage line shows whenever the row's maximum damage is
  non-zero; the Damaging gate is gone, so Heal states its damage line. The
  caption level stays the signed sum clamped to 0..100.
- Map list (`MENU-067`). The size column prints the decoded width and height
  minus 16.
- Widget 8 (`MENU-070` to `MENU-072`). In the fourth slot the readout draws the
  `building.txt` name, `Health` and `current/maximum` (1000 when the maximum is
  zero) at the claimed offsets, font and inks, for the hovered structure, else
  for the structure a plain click selected while no unit is selected. The
  actual `Viewer.Draw` composition suppresses `missionCard` only while it
  draws `structureReadout`. A hovered unit, a unit selection or no readout
  restores the card; an inactive structure selection does not suppress it.
- Structure selection (`MISSION-065`). A plain click selects a destructible
  structure that answers a hover hit; a rectangle never does; a selected unit, a
  right click or a click elsewhere clears it. `selectionSummary` reports `0x20`.
  The selection is viewer state only (`DIV-2109`).
- No change: attribute row boxes already equal `MENU-066` apart from the
  label's unknown extent; the digit grouping equals the `TEXT-098` loop.

## Ledger

Rows updated: `DIV-200`, `DIV-201`, `DIV-247`, `DIV-248`, `DIV-284`, `DIV-286`, `DIV-344`,
`DIV-1882` to `DIV-1885`. New: `DIV-2105` (minimap blips and fog blocks),
`DIV-2106` (usable structure without a pool), `DIV-2107` (cast damage
arithmetic), `DIV-2108` (readout over the card, now closed), `DIV-2109` (viewer-only
structure selection). `DIV-2110` to `DIV-2112` are returned unused.

## Proof

- `pkg/sim/spellrecord_test.go`: the record level wraps at sums below 30 and at
  286 and up; range and duration at several powers. Loss control: removing the
  byte wrap fails it.
- `pkg/game/spell_test.go` `TestThePopupStatesTheRecordAtTheWrappedByteLevel`
  and the release test `TestReleaseSpellbookPopupStatesTheRecordAtTheActorsPower`
  (EN and RU, all 28 installed rows against an independent formula, plus a Heal
  hover in a mission). The same loss control fails the release test.
- `TestReleaseMapListSizeColumnPrintsTheHeaderDwordsMinusSixteen` (EN and RU).
  Loss control: a border of 0 fails it.
- `pkg/ui/structurereadout_test.go` and
  `TestReleaseWidgetEightReadsTheHoveredThenTheSelectedStructure` (mission 20,
  EN and RU): hover wins over selection, the three lines and inks, plain-click
  selection, rectangle and indestructible skips, right-click clear. Loss
  control: reverting the click path fails both.
- `TestMissionColumnReplacesTheCardOnlyWithADrawnStructureReadout` records
  actual `blitColumnLayer` calls. Hovered and selected structures draw readout
  without card; hovered unit, unit selection and no selection restore card.
  No font draws neither. The installed widget witness records the same Draw
  seam on EN and RU and verifies hovered-unit and selected-unit restoration.
  Loss controls force the card always on or always off; both fail the installed
  composition assertions. Evidence is under
  `review/story1301-final-cd515f10/author/` outside this repository.

## Review and continuation

The sole adversarial report is `pipeline/reviews/story1301-review-e55b8c6.md`:
RETURN on `e55b8c6c53708c90390f5406ffd50d4fc82c4a96`. Its returning finding is
the readout/card overprint. This bounded correction implements the accepted
replacement and adds composition proofs. The report remains unchanged.

The original production commit `df1034f4` did not begin with a version bump.
This continuation commits game 0.57.0 and its regenerated Windows resource at
`6413f819` before the new production correction. Starter stays 0.3.1. Current
released main `0a9fdc3e` is reconciled, including pause freeze and dim.
Focused and ordinary checks cover the candidate. The serialized merge owns
the canonical final chain.

## Open debt

The ordinary/unit statistics card remains the owner-directed deviation of
`DIV-344`; the structure readout no longer overlaps it (`DIV-2108` closed).
Cast damage keeps integer arithmetic against the record's double (`DIV-2107`).
Pool-less usable structures give no hit or use route (`DIV-2106`). The
minimap's object walk and fog blocks are not built (`DIV-2105`). The fold's
skip tests stay unmodelled (`DIV-1882`).
