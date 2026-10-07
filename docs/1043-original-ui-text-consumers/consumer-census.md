# Story `1043` production consumer census

This census is finite at candidate freeze. It names production reads and paints, not every text
file line and not diagnostics that print text without painting it.

| Population | Transit | Final production consumer | Witness |
|---|---|---|---|
| five retained tables | `LoadInstallWords` retains `main.txt`, `dialogs.txt`, `unitname.txt`, `stats.txt` and `sites.txt`; `Global`, `Dialogs`, `UnitName`, `Stats` and `Site` keep the index spaces distinct | `InstallWords.Words` resolves each finite `ui.Words` field independently | `TestInstallWordsRetainsTheThreeNewLocalTables`; `TestOriginalUITextWordSetExactDifferencesOverBothLawfulInstalls` |
| sixteen-table input boundary | `TestOriginalSixteenTextTableCountsOverBothLawfulInstalls` opens `main.res` plus `patch.res` and walks the decoded CR tables | no production retention beyond the five tables above | the same test reports 16 tables and 1,568 EN or 1,527 RU lines; only credits differs in count |
| town and shop character | `partyPanelSubject` resolves the party member's Human or Unit row and writes `UnitNameIndex`, full detail 7, active words and the original conditional-caption actor operands | `DrawTownCharacterRegion` to `RenderCharacterPanel` | panel word and level tests; generated-player suppression; flat-unit XP-value witness; both-root shared-town release route |
| character generator | `ChargenPreview` resolves the selected Human definition and writes `UnitNameIndex`, full detail 7 and the active words | `RenderCharacterPanel` | `TestChargenPreviewUsesTheConfirmedPartyProjection`; generated-character release route |
| mission character | `mapWorld.View` carries the resolved definition index and the actor flags, XP value and derived type-`0x49` byte on `MapEntity`; a party member may replace the name index with the party definition index | `Viewer.panelSubject` to `RenderCharacterPanel` | `TestEntityDrawCarriesTheOriginalConditionalPanelOperands`; `TestMissionPanelCopiesTheOriginalConditionalOperandsWhole`; mission selection tests |
| shared panel | `PanelSubject.Words`, `UnitNameIndex`, `DetailLevel`, `DetailSet` and `OriginalPanel` | unit name, fixed captions including conditional slots 190/191, numeric forms and level gates in `RenderCharacterPanel` | `TestOriginalPanelDetailLevelsZeroThroughSeven`; `TestCompactPanelPaintsBothConditionalInstallCaptions`; `TestOriginalConditionalPanelCaptionsRenderOverBothLawfulInstalls`; fixed card geometry suite |
| mission item popup | mission worn and pack builders pass the active words and live weapon-damage resolver | `itemInstanceInfoLinesWithWeaponDamage` | installed-label/literal-form test and lawful enchanted-item population |
| shop item popup | doll, shelf, table and pack builders pass the active words and a nil weapon-damage resolver | `itemInstanceInfoLines` | installed-label/literal-form test and lawful shared-town route |
| shop controls | active words keep main slots 72, 70, 71 and 73 | Undo, Buy, Sell and Exit buttons | existing shop control and both-root shared-town witnesses |
| save acknowledgement | successful menu save reads main slot 203 from `FrontEnd.Words` | existing timed message surface | `TestReleaseGameMenuPopulation` checks the exact active-install acknowledgement |
| mission outcomes | main slots 140 and 141 remain in `missionOutcomeText` | existing outcome notice | `TestMissionOutcomeTextComesOffTheViewer`; Victory and Defeat routes remain in the release suite |
| mission and town Esc menus | existing dialog-table fields retain separate mission and town roots | game-menu roots and confirmations | `TestReleaseGameMenuPopulation` exercises campaign, standalone and town populations |
| world-map home | `townScreen.WorldMapView` carries active words | `ComposeWorldMap` paints `sites.txt[0]` and main slot 261 in the existing two-line card | `TestWorldMapCardTextUsesItsOwnOrigins` and world-map adapter tests |
| world-map payment | mission payment and active words enter `drawWorldMapCard` | positive payment paints main slot 262 with `%s: %d c` | the same independent frame witness |
| zero selected | `Viewer.characterPaneView` counts `presentSelected` | `SelectionStatusLines` paints main slots 47 and 48 | `TestMissionSelectionStatusDrawsAtItsOwnTextOrigins` |
| one selected | the same count arm forwards the one complete subject | `RenderCharacterPanel` | `TestMissionPluralSelectionStatusKeysThePresentedCount` includes the one-subject arm |
| plural selected | count greater than one keys both status strings and count | `SelectionStatusLines` paints main slots 49 and 50 plus the count | the same cache-key test and independent frame witness |

The shop category/shopkeeper hover, item-identification prompt and marker-site label are explicit
exclusions. This tree has no corresponding production interaction or painter. Adding one would
change Town & Economy or world-map mechanics, not merely route text. Script dialogue, tips,
briefings and credits are authored-content consumers outside this program-chosen-string census.
