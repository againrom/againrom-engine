# Mods edit characters

## Intent and authority

A mod changes or replaces one named character with files only: `data/characters.toml`, names in `text/<lang>/strings.toml`, and the existing Starlark call that loads the data file. Authority is owner direction (tester remarks 42 and 43, owner decision 8: an example mod that replaces or edits specific characters). Item 42: take the girl in armour, strip her, make her one more kind of peasant woman. Item 43: name her "Лучница". ROM1 differences are rows `DIV-1900` to `DIV-1905` in `docs/divergences/mods.md`. Without a mod, every hash, SAV byte, scenario and release witness is unchanged.

## Which girl

The screenshot of item 42 is not available. Observation (instrument: a dump of the `Humans` collection of `data/data.bin`, EN install, 215 rows): the female archers are rows with type 14 and sex 1.

| rows | what they are | armour |
|---|---|---|
| `NPC06_1` to `NPC06_4` | the tavern's mercenary type 6, face 3 | hard leather at tier 1, dragon leather at tiers 2 to 4 |
| `PC_Naira` to `PC_Naira_4` | the female archer hero rows | leather to adamantium |
| `A_Brigand1` to `4` | enemy women archers, face 10 | leather |
| `A_PeasantGuard1` to `4` | scenario peasant women archers, face 9 | none |
| `M141_BrigandFemaleXBow` | a crossbow woman, type 15 | leather |

Choice (inference): `NPC06`, because it is the armoured female archer a player can hire and inspect in the tavern (type 6), her armour is drawn on the card, and the shipped `A_PeasantGuard` family is already the peasant-woman archer with no armour, which gives "one more kind of peasant woman" a shipped meaning. Alternatives: `PC_Naira` (a hero, not a hireable kind), `A_Brigand` (enemies a mission places) and `M141_BrigandFemaleXBow` (a crossbow woman, type 15). The example mod's target is one line, `target = "NPC06"` in `pkg/modrt/testdata/mods/archer-girl/data/characters.toml`; changing it and `kind` redirects the example.

## As built

Data file (`pkg/mod/characters.go`). `[[character]]` tables. Keys: `target` (required), `name` (a text key, English or Russian text), `kind`, `face` (1 to 127) and `strip` (an array of `weapon`, `shield`, `armour`). At least one edit besides `target`. Every refusal carries mod, file and line: unknown key or table, wrong type, face range, unknown or repeated strip group, missing text key, name longer than 40 characters, an edit that changes nothing.

Loading (`pkg/modrt/characters.go`). The script calls `game.data.add("data/characters.toml")` in `init`; the file is loaded once, only in `init`. Text comes from the language of the base, Russian falling back to English per key (as for items). `modrt.Result.Characters` carries the rows in load order. `cmd/againrom` calls `FrontEnd.SetModCharacters` after `SetModItems`, and `-check` prints `againrom: characters edited=N` only when a mod carries characters.

Edit (`pkg/game/modcharacters.go`, `pkg/data/rowedit.go`). A target is a definition row by its full name, else every row named the target followed by `_` and a tier number. A target that names no row refuses the launch. `kind` is a row name, else the kind followed by the target's own tier, else the first row of the kind followed by digits; the target takes the type, face and sex columns (slots 16 to 18) of that row. `face` is applied after `kind`. `strip` empties cells: `weapon` is cell 0, `shield` cell 1, `armour` cells 2 to 9 (every armour and jewellery cell). `data.EditedRows` lays the edits over the `Humans` collection and replaces `Table.Humans` and the install's `Humans`, so every reader of a row sees the edited row; the row count, names and other rows are unchanged. Several edits of one row merge in load order.

Names (`mapload.ModContext.CharacterName`). A name is kept per definition row, as the install's own byte alphabet (`encodeSaveLabel` with the font's language selector, so a Russian install draws the Cyrillic text; a character that alphabet cannot draw refuses the launch with mod, file and line). It replaces the class name in the tavern candidate card (`tavernCandidateDetail`), the party panel (`partyPanelSubject`: the town shop and the mission inventory) and the mission roster after a join (`canonicalizeJoinedRoster`). `SetMods` keeps the characters already set, as it keeps items.

Example mod (`pkg/modrt/testdata/mods/archer-girl`, test data): `target = "NPC06"`, `kind = "A_PeasantGuard"`, `strip = ["armour"]`, name `Archer girl` and `Лучница`. The mod loads from `builds/current/mods` like `heavy-armor`.

Saves. SAVE writes the hired girl from her current state: face byte 9, type 14, the bow in the weapon slot, nothing else worn. The `AgainromMods` leaf gains nothing: the mod set is already recorded (`DIV-1802`), the same set rebuilds the row edits at launch, and LOAD without the mod refuses naming it. The name and the row edit are not in the SAV (`DIV-1904`).

## Proof

- Unit: `pkg/mod` (parsing and every refusal with file and line), `pkg/data` (`EditedRows`: edited rows only, ignored slots, the base unchanged), `pkg/modrt` (example mod in both languages, refusals, one load, load order), `pkg/game/modcharacters_test.go` (family and exact targets, kinds by tier, a one-row kind, face after kind, strip groups, merge order, every refusal, `SetMods` order, names).
- Release, EN and RU, one root at a time (`pkg/game/modcharacters_release_test.go`, gated by `AGAINROM_ASSETS`):
  - Every `Humans` row of the modded table equals the unmodded row except `NPC06_1` to `NPC06_4`; those four take the type, face and sex columns of `A_PeasantGuard1` to `4` tier for tier, keep every other parameter, keep the weapon and shield cells and hold no armour cell. The row count and names are equal.
  - The tavern card of type 6 shows the language's name and the unmodded card does not; types 4, 7 and 10 show the same name in both. The hired girl has face 9, class 14, the bow only, the same statistics and weapon as the unmodded hire, and the party panel names her. The witness fails on the unmodded hire.
  - A town SAV with the girl hired is a SAV; a cold LOAD under the mod restores face, direction, class, definition row, worn set, pack and name; a second SAVE and LOAD carry the same state; LOAD without the mod refuses by the mod mark. Loss control: the unmodded hire saved and loaded fails the witness and has another face.
  - In a mission the hired girl's party record passes the same witness and her map entity is drawn with the mod name; a mission SAVE and cold LOAD under the mod restore both, and a mission LOAD without the mod refuses by the mod mark.
- Screenshot (offscreen, never skipped): `review/story1278-mod-characters/tavern-archer-girl-<base>.png` beside `tavern-original-<base>.png`, the tavern card of type 6 with and without the mod, EN and RU.

## Open debt

- The map sprite is the class sprite; `kind` with another type changes it (`DIV-1902`).
- The tavern list portrait keeps the shipped picture (`DIV-1905`).
- A person a map places from the edited row, and a scenario speaker, keep the class name (`DIV-1901`).
- `strip` drops all armour cells or none of them; no per-slot strip, no added equipment.
- No statistics key: the stripped girl's defence is that of a person wearing nothing (`DIV-1903`).
- Whether the original reader accepts the saved hired record is Unknown (`DIV-1904`).
- Original SAV-corpus saves were not loaded under a mod.
