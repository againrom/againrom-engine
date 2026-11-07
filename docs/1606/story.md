# The map editor's people, grouped by role and shown as the game draws them

## Intent

The unit popup's identity choice tells the map maker who each person is.
People are grouped by role, and inside the other rows by the class sheet the
mission draws. A mercenary type is one cell, and its level is a separate
choice. Every person cell shows the sprite the game draws on the map, never a
portrait or a bare paper doll. Map editor work counts zero toward a release.

## Authority

The owner asked for this: «надо дальше начать делить Людей в редакторе ... еще
важно чтобы мы могли понять какой у них спрайт в игре (не портрет!)». The owner
answered «Как рекомендуешь» to the seat's four proposals:

1. People are grouped by role, and inside a role by the sprite the game draws.
2. A mercenary type has one cell, and its level 1..4 is a choice.
3. A cell shows the map sprite.
4. This is one map editor story.

Further owner rulings: one builder per kind. The map editor is our own
program, so no claim, research or DIV row is needed. No game behaviour, save
or simulation path changes.

## As-built behaviour

### Groups

`MapInspector.identityGroups` (`pkg/game/mapinspectionidentity.go`) builds
the people block in this order. A group with no member is left out.

| Head (EN / RU) | Rule |
|---|---|
| Story heroes / Сюжетные герои | a named npc character (`mapload.UnitNamed`) |
| Player characters / Персонажи игрока | a Humans row whose name starts `PC_` |
| Mercenaries / Наёмники | a row `mercenaryRowFromName` reads as `NPC<type>_<level>`, type 3..15; one cell per type |
| Others: fighters / Прочие: бойцы | any other row whose type composes a figure and is neither a horse type nor a mage figure |
| Others: mages / Прочие: маги | the figure `mapload.PlacedFigure` gives is a man or woman mage (types 0x17, 0x18) |
| Others: riders / Прочие: всадники | `data.FigureHasHorse` (types 0x11..0x15) |
| Others: <class> / Прочие: <class> | a type that composes no figure (`!data.ComposesFigure`) or is a hero type (`data.FigureIsHero`); one head per class name of the sheet the mission draws, after the fixed heads |

`personRole` reads the type from the Humans row. When
`mapload.PlacedClientHero` gives a second-game client hero type, that type is
used. Inside each other head the men come first, then the women, each in
table order. Women are the figures `FigureDir.Female` names.

On the EN and RU installs, every Humans type is below 0x1a: fighters
0x1..0xf, riders 0x11..0x15, mages 0x17 and 0x18. This confirms the seat's
reading. No "Others: <class>" head appears on either install.
`PlacedFigure` reads rows 17..21 (`ManHorse_*`) as women: the gender column
is -1, and the spawner's default gender is 1. So these rows come after the
men in the riders head.

### Mercenaries

`identityChoices` collects each type's levels into `mercenaryLevels`. It
offers only the lowest level's row as the type's cell, which is level 1 on
both installs. The other levels are not choices.

`identityField` shows a placed mercenary of any level as his type's cell.
`levelField` adds one `ui.InspectionField` with key `level` (EN "Level",
RU «Уровень») after the identity field. Its choices are the type's levels,
and its value is the placement's level.

`setField` routes `level` to `setMercenaryLevel`. That function keys the
record to the type's row of the chosen level through `keyIdentity`, which is
the identity field's writer. So the change is one `SetFields` edit with undo
and redo. Only the class, tier, definition and npc-flag keys move.

`setIdentity` does not touch a mercenary when the popup sends back his type's
cell. Without that guard, applying the popup with another field changed
would reset the level. `AddUnitIdentity` keys a new placement through
`keyIdentity` directly.

### Pictures

`identityPictures` → `identityFigure`:

- A creature keeps its placed tier sprite (`unitPreview`).
- A person or named character takes `classPreview` of `personDraws()`.

`personDraws` places every person and named identity on one new map:

1. It makes a blank placement record with `placeNewUnit`.
2. It starts the map as a mission starts it, with `mapload.StartCampaignMission`.
3. It settles the look as a mission's opening does, with `canonicalizePartyAppearance`.
4. It takes the class sheet and tier from `missionAppearanceArt` and `entityTiers`.

A composed named character (records 21..24) is started on the table his start
composes him on. The new `mapload.PlaceOn`, which is `mapload.Place` that
also returns that table, supplies it. A start that refuses its placements is
started again one placement at a time, so only the refused placement draws
nothing.

`classPreview` (`pkg/game/mapinspectionadd.go`) is the body of `unitPreview`
taken out for reuse: the first frame of `TierFrames(tier)` at the default
facing, `SelectUnitFrame(..., false, 0, 0, 0)`. Portraits and paper dolls no
longer reach the palette. The hover tip is unchanged: the cell label
`identityName`, which is the Humans row name, or the npc record name and
number.

### Kinds and builders

| Kind | Builder | Extension |
|---|---|---|
| The identity choice's groups | `identityGroups` | new heads as data (`personGroupWords`), a role per identity (`personRole`), a head per unfit class, men then women |
| An identity cell's picture | `identityPictures`, `identityFigure` | a person's picture comes from the mission art path; the npc portrait and `composeDoll` calls are removed |
| A unit's map sprite | the mission's art path: `StartCampaignMission`, `canonicalizePartyAppearance`, `missionAppearanceArt` (`pkg/game/world.go`), `entityTiers` | called unchanged |
| A class sheet's first frame | `unitPreview` | its body became `classPreview`, and `unitPreview` calls it |
| A unit popup field and its edit | `Fields`, `setField`, `identityField`, `setIdentity` | one more field (`levelField`); `setIdentity`'s writer became `keyIdentity`, shared by the level |
| The placed record of an identity | `mapload.Place`, `mapload.IdentifyAt` | `Place` calls the new `PlaceOn`, which returns the table it resolves on |
| A mercenary's type and level | `mercenaryHireTypeFromName` (`pkg/game/originalparty.go`) | its body became `mercenaryRowFromName`, which also returns the level, and the hire calls it |
| Words | `pkg/words/text/{en,ru}/strings.toml` | new group heads, the level field and its note; the removed heads `npc`, `others`, `heroes`, `pc` |

No second builder was added.

## Proof

### Focused tests

Focused tests on a synthetic table, `pkg/game/mapinspectionpeople_test.go`.
The fixture is `unitIdentityEditor` with its Humans table replaced by 17 rows:

- a woman and a man fighter (the man is npc record 8's row);
- `PC_Hero` and `PC_Mage`;
- `NPC03_1..4` (type 0x18) and `NPC10_1..4` (type 3);
- a woman and a man mage (0x17, 0x18);
- a woman and a man rider (0x13, 0x15);
- a type 0x1c row of class "Odd class".

Each class sheet has its own frame size.

- `TestEditorIdentityPeopleGroupByRoleAndDrawnClass`, EN and RU. It checks
  the people heads and their members: story [named 8], player [3, 4],
  mercenaries [5, 6], fighters [2, 1], mages [14, 13], riders [16, 15],
  "Others: Odd class" [17]. It also checks that no level 2..4 row is a choice.
- `TestEditorIdentityPeopleCellsDrawTheMissionSprite`. Each person and named
  cell equals, pixel for pixel, `classPreview` of the art that a separate
  mission start of that one placement gives through `missionAppearanceArt`.
  That art is also the type's own sheet. No cell equals the npc portrait or
  the bare doll, and every row of one type draws one picture.
- `TestEditorUnitMercenaryLevelSwitchesTheRowWithUndo` checks the level route:
  - The cell places `NPC03_1`.
  - The level field reads 1 with choices 1..4.
  - Level 3 rekeys the record to `NPC03_3` and keeps every other editable field.
  - Choosing the type's cell again changes nothing.
  - Undo restores the level-1 bytes, and redo restores the level-3 bytes.
  - Level 5 is refused.
  - A unit that is not a mercenary has no level field.

### Golden changed

`TestEditorIdentityGroupsKeepTableOrderAndEveryRow`
(`pkg/game/mapinspectionidentitygroups_test.go`) changed its people heads and
members. The old heads "NPC", "Others", "Heroes" and "PC" are replaced by
"Story heroes" [named 0, 7, 8], "Player characters" [2] and
"Others: fighters" [5, 6, 1]. The reason is this story's new grouping. The
creature groups and the check that the grouped values equal the offered
values are unchanged.

### Install census

The census is a temporary headless test, removed before the commit, run
through `NewMapInspector`, `identityChoices`, `identityGroups` and
`identityPictures`, mission 1. EN and RU give the same counts:

| Head | Cells |
|---|---|
| Story heroes | 23 |
| Player characters | 20 |
| Mercenaries | 13 |
| Others: fighters | 89 |
| Others: mages | 38 |
| Others: riders | 11 |

- Mercenaries: 13 types (3..15), each with 4 levels.
- People cells: 194. Three have no picture: rows 17 `ManHorse_Unarmed`,
  18 `ManHorse_Sword` and 20 `ManHorse_Lance`. Their types 0x11, 0x12 and
  0x14 have no class sheet in `units.Classes`, so the mission art path gives
  them none.
- The second-game installs (rom2-ru, rom2-en) do not open in
  `NewMapInspector` at base `30dd2af67`: "world/data/data.bin: databin:
  group C Armors entry 4 at +0x363b: parameter array of 65280 value(s) runs
  past the end". This was shown from a `git archive` extract of the base, so
  no second-game census exists.

## Open debt

- The three rider rows with no class sheet draw no cell picture. What the
  game shows for them on a map is not established here.
- The second game's map editor cannot open its installs at base, so the
  second-game grouping (client hero types through `PlacedClientHero`) is
  proven only by code path, not on an install.
- The release witnesses for EN and RU run at the next release, starting with
  `TestReleaseMapEditorIdentityGridUsesInstalledRowsAndTiers`.
