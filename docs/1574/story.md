# Every shipped trigger row is a wizard item

## Intent

The map editor's trigger wizard read 582 rows of the original scenarios as «Другое» (Other): a
row the wizard keeps whole and only Details edits. Every kind of those rows (census
`review/editor-other-census/CENSUS.md`, Part 1: conditions C1..C15, actions A1..A17) becomes an
ordinary wizard item, as data in the one item table, with EN and RU words. A shipped row reads as
its item, a new row of the item writes the shipped form, an unedited row saves byte for byte, and
the variable parameter offers the map's variables and a new one. The only rows still Other are the
pairs of two IDs no node has (C10) and the distance check written over two units (C2 Unit, Unit),
whose meanings no claim states.

## Authority

- Owner direction (story brief): every census kind is a wizard item; multi-comparison kinds take
  the comparison as a row choice; labels state a Medium or unnamed kind; C10 and the C2 Unit, Unit
  row stay Other; the item list stays usable at 1920x1080, grouped by subject, with no new widget
  kind. The map editor is our program: no research and no divergence row.
- Census (EN and RU, base `5295a6c8a`): EN 38 maps, 421 triggers, 1285 rows, 582 Other; RU 34
  maps, 418 triggers, 1282 rows, 582 Other; the shape list is the same in both. Rows per kind and
  their forms are the census table's.
- Claims the item table and the census cite: TRIG-CMP-006 (comparison codes; above 5 never true),
  TRIG-CHECKGEOM-130, TRIG-SACK-022 and TRIG-ITEMTEST-040 (checks 2, 14 and 17 write 0 or 1, so
  `!= 0` states `== 1`), TRIG-DIST-014, TRIG-BIND-010 (an ID no node has; the C10 inference),
  TRIG-ACT-004, TRIG-GROUP-005,
  TRIG-GROUPBOUNDS-136, TRIG-ADDITEM-027, TRIG-TAKEITEM-038, TRIG-GIVEALL-025, TRIG-CAST-033,
  TRIG-EFFECTTIME-034, TRIG-CELLTAIL-035 and TRIG-PROPERTY-036 (selectors 6, 15 and 16).
- Variables and check results share registers 0..99; a new check takes the register of its index.

## As built

| kind | builder |
|---|---|
| wizard item (condition and action) | `pkg/ui/mapeditorwizard.go` `wizConditions` (23 items) and `wizActions` (31 items, four of them order variants); new fields as data: `chooseCmp`, `also`, `rightType`, `holds`, `subject`, `variant`, parameter `right`; no per-kind branch |
| wizard row reader | `readCondition`, `readPair`, `readParams`, `reads`, `readAction`: a pair of a constant and a check reads mirrored (C3 reversed pair reads as «Отряд рядом с точкой»); a constant compared with itself by `==`, `>=` or `<=` reads as «Всегда (в начале)» |
| wizard row writer (ui plan) | `rowsPlan` and `wizardNode`: an edited mirrored row writes the check on the left; a right-side parameter writes a second node of the same check (C1 and C11 two-check forms) |
| trigger writer (host) | `pkg/game/mapinspectionwizard.go` `writeWizardTrigger`, unchanged in shape; a comparison code above 5 is written when it equals the code the slot was read with, whether or not the row's nodes were edited (the wizard offers such a code only on the row that was read with it), so the one shipped pair with code `0xFFFFFFFF` (`131.alm`, TRIG-CMP-006) is editable and unedited rows stay byte for byte; `WriteTrigger` translates the register refusals into `editor.wizard.registers` and `editor.wizard.checksfull` |
| field choices (registry) | `pkg/game/mapinspectionscriptchoices.go` `scriptTargets`: variables, items, structures, spells and the property selectors (view kind `ScriptPropertyParameter`, from `scriptParameterKind`); the wizard reads them through `parameterChoices` |
| new variable register | `scriptTargets`, the one allocator: «Новая переменная» is the highest free register, so the next new condition, which takes the register of its index, does not collide |
| item list window | `pkg/ui/mapeditorobject.go` `choiceWindow`: a field with `ListAll` opens with every row that fits below it in the window, ten at least; the picker's row limit is raised to that window |
| words | `pkg/words/text/{en,ru}/strings.toml`: item, parameter, subject and refusal words |
| static guard | `internal/archtest/wizarditems_test.go` |

Kinds and their items:

| census | item |
|---|---|
| C1 | `c.variable` (variable vs number, comparison chosen); `c.variables` (two variables) |
| C2 | `c.unitcell` (Unit, cell; comparison chosen) |
| C3 | `c.always` (self-compare); reversed pair: `c.unitnear` |
| C4 | `c.sackgone`, `c.sacklies` |
| C5 (check 15) | `c.playernear`, widened: comparison chosen |
| C6 (check 17) | `c.hasitem`, `c.lacksitem` |
| C7 | `c.structure` (a switch, by the maps' own names) |
| C8 | `c.health` (property 6) |
| C9 (check 6) | `c.unitnearunit`, widened: comparison chosen |
| C11 | `c.lose`, `c.losetwo` (acts without a trigger) |
| C12 (check 5) | `c.unitalive` (`!= 0`); `c.unitdead` widened (`!= 1`) |
| C13 (check 2) | `c.heroarea` widened (`!= 0`) |
| C14 | `c.target` (meaning not certain) |
| C15 | `c.itemnear` |
| A1, A10 | `a.addvar`, `a.setvar` |
| A2, A12, A14 | `a.takeitem`, `a.giveitem`, `a.giveall` |
| A3..A6 | `a.order` with Stand Ground, Follow, Attack and Defend, each with its signature's parameters |
| A7, A8, A15, A9, A16 | `a.castcell`, `a.castunit`, `a.trap`, `a.celltime`, `a.unittime` |
| A11 | `a.property` |
| A13 | `a.replace` |
| A17 | `a.action7` «Действие 7 (значение не установлено)» |

The item list shows a shut heading row per subject (mission, units, players, groups, cells, items,
structures, spells, variables) and each item under it; the order variants show as one entry
«Приказ группе», and the row's command chooses the variant.

## Proof

Focused tests, each failing on the base tree (`d3e5c6fc3`, exported with `git archive`; the new
file copied in; every test failed):

- `pkg/game/mapeditorwizardkinds_test.go`:
  `TestTriggerWizardReadsEveryCensusKindAsItsItem` (every shipped form, EN and RU, 1920x1080; C10
  and C2 Unit, Unit stay Other), `TestTriggerWizardKeepsEveryUneditedCensusRow`,
  `TestTriggerWizardNewRowOfEveryCensusKindWritesTheShippedForm`,
  `TestTriggerWizardEditedReversedPairWritesTheCheckOnTheLeft`,
  `TestTriggerWizardNewVariableThenANewConditionAreAccepted`,
  `TestTriggerWizardRefusesAConditionOnTheLastRegisterInTheEditorsWords` (EN and RU),
  `TestTriggerWizardItemListsOpenWholeOnTheSmallestScreen` (1920x1080, EN and RU).
- `internal/archtest`: `TestCheckWizardItems` (self-test) and
  `TestTriggerWizardReadsAndWritesEveryKindThroughTheItemTable`: the reader and writer functions
  test no opcode literal and no item label, and no wizard item is built outside the two tables.

Release test, not run in the lane: `pkg/game.TestReleaseTriggerWizardReadsEveryShippedRowAsAnItem`
(population row with no root column) opens every trigger of every installed map at 1920x1080:
the rows read as Other are exactly the C10 pairs and the C2 Unit, Unit checks, and the map saved
without an edit carries its script record byte for byte. The next release runs it on EN and RU
with `TestReleaseMapEditorWizardApplyWithoutEditKeepsEveryTrigger`,
`TestReleaseTriggerWizardOpensShippedTriggers` and the changed release tests
`mapeditscriptparams_release_test.go` and `mapeditortrigger_release_test.go`.

Goldens: none moved (side panels, plain views, technical views and the modes walk pass unchanged).

## Open debt

- The «Условия» and «Переменные» lists, MCP trigger commands and second-game installs are out of
  scope. ROM2 maps were not opened.
- Expected remaining Other rows per root, from the census: EN 5 (C10 4, C2 Unit, Unit 1), RU 5.
  The release test confirms it on the installs at the next release.
- A shipped variable on a low register equal to the condition count still refuses a new condition
  until the variable or a condition goes.
- Int and Const constants are read as one kind (inference; the census shows both forms per kind).
- The Details editor of action 34 now offers the property selector as a choice field.
- «Новая переменная» in Details (checks 19, actions 3 and 8) also takes the highest free register,
  because `scriptTargets` is the one allocator; Details writes different bytes from the base's.
- Actions 29 and 30 read «Задать оставшееся время эффектов на клетке/отряде»: per
  TRIG-EFFECTTIME-034 they rewrite the remaining time of effects already present and do nothing
  when there are none.
- Wizard notes built from the editor's own words (a bad number, an unset field) and Details notes
  are encoded when set and again when drawn, so in RU they show `?`. The host refusal note is
  encoded once.
