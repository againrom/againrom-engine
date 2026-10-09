# ROM1 chat cheats and Alt console

## Intent and authority

The owner reverses the former console omission in DIV-2077 and requires the ROM1 chat cheats and debug keys. An optional starter checkbox and `-chicken` command-line flag apply `#Chicken` at every fresh mission start. The option defaults off. ROM2 is outside this change.

Base: `a975806f367c1d8e1b456d40b305d863942a2ffc`. The authority is the pinned public knowledge snapshot: `MENU-099` through `MENU-114` for the command chain, gates, mutations and SAV consequences; `MENU-062` for Alt keys; `AI-378`, `AI-397` and `AI-398` for the console. `HERO-MODDK-161`, `HERO-STAT-001` and `UNIT-147` supply the actor modifier, primary-stat and knowledge meanings. Claims establish ROM1 behaviour. Implementation tests establish Againrom behaviour.

## As built

The production route is mission `App` input, the `FrontEnd` destinations, the live `mapWorld` command parser, and ordinary simulation or client mutations. Enter opens chat; Enter submits; Escape closes; Backspace removes one Unicode character. The editor admits at most 256 UTF-8 bytes and draws with the installed mission font and text encoder. Its layout, limit and mission-only lifetime are the smallest choices under DIV-2546.

The parser uses the 13 case-sensitive prefixes in the original chain order, including the two aliases of one kill command. The participant gate refuses all chat cheats on maps declaring more than one participant. Nine commands require privilege above 50. `#modify`, `#event` and `#Chicken` do not require privilege. Alt console keys have the privilege gate and no participant gate (`MENU-099`, `MENU-100`, `MENU-101`, `AI-397`). Alt dispatch uses the live mission context, including Help, notices, Drop Gold, the mission Game Menu and mission Save/Load choosers. The same context admits Alt+S; town and non-mission choosers do not.

| Command | Result | SAV surface or runtime lifetime |
|---|---|---|
| `#create [count] Gold` or item name | A positive first count token is used; otherwise the count is one and the whole argument is the name. Gold is added to the hero owner's purse. Items enter the ordinary hero inventory. Failed construction sends notice 6; success sends notice 7. | Player `Money`; actor inventory and ordinary Item records. Installed item names and factory grammar are bounded by DIV-2541. |
| `#modify self/army +god` | Writes six modifier protection words and six damage-kind bytes to 100, then uses ordinary actor derivation. | Actor modifier block `UD4`. |
| `#modify self +spell id` and `+spells` | Inserts the selected valid spell or ids 1 through 28 in the hero book. These are arms of the same `#modify` command. | Actor `HasSpellbook` and `Spells` references, ordinary Spell records. |
| `#modify self/army +knowledge` | Reprojects real knowledge without privilege; privilege above 10 selects level 15 for the client card and tooltip. | Diary counts remain unchanged. The client override is transient. DIV-2454 is closed. |
| `#summon [count] name` or `#summon hero name` | Constructs ordinary installed Units first, then Humans, owned by the caller's hero. `hero` forces count one and preserves the ordinary actor type. Uses current mission difficulty, fresh identities and nearest free cells. | Ordinary actor, equipment and inventory records. Names, the 4096-count bound and placement are DIV-2542. |
| `#killall` or `#kill all` | Writes health -50 for every actor whose owner's relation toward the caller is hostile, including the caller when its diagonal relation is hostile. Sends notice 7. | Actor health and ordinary death continuation. Immediate transition timing is DIV-2077. |
| `#kill cheaters` | Demotes every other privileged Player and writes health -50 on its actors. Sends no notice. | Actor health persists. Privilege does not. Immediate transition timing is DIV-2077. |
| `#kill name` | Selects the exact current Player name, writes health -50 on its actors, and sends notice 7. | Actor health. Name projection and a missing match are DIV-2543; immediate transition timing is DIV-2077. |
| `#pickup all` | Takes every ground Sack through ordinary pickup, credits gold, moves inventory, removes the Sacks, sends notice 7 and posts `All sacks picked up`. | Player `Money`, actor inventory, Item and Sack records. |
| `#show map` | Reveals the client exploration plane, freezes its fog clear and gives enemy cards level 7. Sends notice 7. | Client reveal flag is transient. Ordinary exploration writes remain on the existing SAV path. |
| `#hide map` | Clears the client reveal flag without clearing exploration already revealed. Sends notice 7. | Runtime flag only. |
| `#victory` | Opens the retained ordinary Victory/Continue panel. Victory, or Continue followed by later Victory, uses ordinary campaign completion: carry the party, record the mission done and reach its successor or town. Side missions retain that panel and the same completion route. Sends no notice and does not force the simulation outcome. | Ordinary continuation supplies any saved campaign changes; the command adds no SAV field or explicit win-latch override. Side presentation is FIDELITY-DEBT and the saved-latch consequence remains Medium/Unknown in DIV-2545. |
| `#event n` | Opens the existing event dialogue for the numeric id without privilege or a cheat notice. A missing event opens no panel. | Runtime dialogue only; missing text is DIV-2544. |
| `#Chicken` | Sets privilege to 255, posts the original enable-cheating line and installed notice 5. A suffix still matches the prefix. | No SAV state. Fresh-mission and cold-LOAD policy is DIV-2540. |

Notices 5, 6 and 7 use installed `main.txt` entries 221 through 226 and the current Player name (`MENU-111`). Console replies use the existing local mission message line; the absent connection broadcast and unread original presentation are DIV-2547.

MENU-109 establishes at High that cheat victory reaches the same client win arm as the ordinary win packet. Its side-mission arm posts the mission-end message instead of constructing the Victory/Continue panel. MISSION-VICTORY-030 establishes that this message reaches the win-latch test and tears the mission down; SESS-END-011 establishes the campaign win path back to town for side missions. The existing real-win side panel is retained here under the owner minimum rule; this presentation difference is DIV-2545, FIDELITY-DEBT. The client arm's effect on a SAV-carried latch is Medium and remains Unknown.

Create, each summon and pickup-all preflight the ordinary import limit of 2^20 item/effect values across living and dying actors. Each carried instance contributes one plus its effect count; equipment uses the same width. Create includes cumulative merges, summon includes its proposed pack and equipment, and pickup-all includes every ground Sack. Over-capacity mutations refuse before expanding holdings; pickup-all refuses before any Sack transfer. These bounds prevent a mutation from exceeding ordinary cold-LOAD capacity (DIV-2541).

Alt+B through Alt+Y, except S, enter the console. D and T toggle the turn and script trace flags and print their state. Q toggles the simulation's AI admission override and prints Safe mode state. H prints the six original help lines. I prints bounded current tick and active counts under the original statistics headings; the unread segment, AI, script, activating and average definitions remain DIV-2548. U prints per-group actor counts and source experience totals in deterministic owner/group order; actors without source backing supply no experience total. Unread formatting and the unexamined virtual call are DIV-2549. The other 17 admitted letters do nothing. D/T trace consumers are unknown, so no trace stream is invented (DIV-2550).

Alt+S independently requests a screenshot from a focused mission context, including open chat, mission popups and mission Save/Load choosers. It needs no privilege. The Draw hook captures the completed App frame, including its final overlays and pointer, and the selected launch save store writes `screen0000.png` through `screen9999.png` without overwriting existing files. The output is PNG, limited to 8192 pixels on each axis, 2^24 pixels total and 16 MiB. Other contexts refuse the request. Format, naming, destination and outside-mission behaviour are the smallest choices under DIV-2546; the original screenshot writer remains unread.

Privilege, knowledge override, map reveal flag and trace toggles start clear for every fresh mission and ordinary cold SAV LOAD. Safe mode also starts false there; the internal deterministic simulation binary retains it for continuation. Its original save lifetime is unknown. The launch option invokes the same `#Chicken` arm only at fresh mission starts, retains that arm's participant gate, and does not grant privilege on LOAD. The starter stores the option in its own launch INI and forwards `-chicken` only when selected (DIV-2540).

## Proof

The installed test inventory has fifteen command and launch witnesses plus one screenshot publication witness. Each uses the App input or mission-entry route. The required release runs execute them on EN and RU.

| Installed surface | Witness test |
|---|---|
| Create gold and inventory; ordinary SAV and cold LOAD | `TestReleaseCheatCreateUsesAppAndOrdinarySAV` |
| Create an item by its localized installed alias; ordinary SAV and cold LOAD | `TestReleaseCheatCreateLocalizedAliasUsesAppAndOrdinarySAV` |
| God, single spell, all spells and knowledge; ordinary SAV and cold LOAD | `TestReleaseCheatModifyUsesAppAndOrdinarySAV` |
| Summon actor, metadata, live/cold source figure and App pane, guard group and post-load App movement | `TestReleaseCheatSummonUsesAppAndOrdinarySAV` |
| Named kill, both kill-all literals and kill cheaters | `TestReleaseCheatKillsUseAppAndOrdinarySAV` |
| Pickup every Sack, gold and inventory; ordinary SAV and cold LOAD | `TestReleaseCheatPickupAllUsesAppAndOrdinarySAV` |
| Show map, hide map, victory, event and Chicken cold reset | `TestReleaseCheatClientCommandsAndChickenColdReset` |
| Campaign cheat Victory completion and ordinary SAV cold LOAD: main successor, main town and side town, including Continue followed by later Victory | `TestReleaseCheatVictoryUsesCampaignCompletion` |
| Six acting Alt keys, help and console replies | `TestReleaseCheatAltConsoleThroughApp` |
| Alt console privilege and execution while Drop Gold is open | `TestReleaseCheatAltConsoleThroughDropGold` |
| Knowledge resend without privilege | `TestReleaseCheatKnowledgeWithoutPrivilegeKeepsOrdinaryCounts` |
| Launch option does not grant privilege on cold LOAD | `TestReleaseChickenLaunchFlagDoesNotRaiseColdLoadPrivilege` |
| Participant refusal for every chat cheat | `TestReleaseCheatParticipantGateRefusesEveryChatCommand` |
| Ordinary chat and Player names use the installed alphabet | `TestReleaseCheatOrdinaryChatAndNamesUseInstalledAlphabet` |
| Launch option off/on at each fresh mission | `TestReleaseChickenLaunchAtEveryMissionStart` |
| Alt+S publication into the selected profile using a supplied App frame | `TestReleaseScreenshotAltSSelectedProfileSuppliedFrame` |

Focused construction, simulation and input controls:

- App routing and editor: `TestCheatChatRoutesEveryCommandThroughAppInput`, `TestCheatAltRoutesEveryLetterThroughAppInput`, `TestCheatChatConsumesGameplayAndEditsBoundedText`, `TestCheatInputRespectsFocusPopupAndMissionEntry`, `TestCheatChatUsesMissionFontAndInstallEncoder`, `TestCheatChatAdmitsHelpAndRetainsDraft`, `TestCheatChatAllowsAltConsoleWithoutClosingDraft`, `TestCheatAltConsoleAndScreenshotUseMissionContext`.
- Reveal lifetime: `TestCheatMapRevealPreservesClientExplorationAcrossFogPushes`, `TestCheatMapRevealFreezesVisibilityUntilHiddenFogRefresh`, `TestCheatMapRevealDoesNotChangeTemporaryRevealPolicy`.
- Simulation and cold continuation: `TestCheatGodWritesWholeModifierAndUsesSourceDerive`, `TestCheatGodNativeBackingAndSpellInstancesSurviveColdLoad`, `TestCheatKillGoldItemAndPickupUseOrdinaryState`, `TestSafeModeOverridesDispatchAndNativeContinuation`, `TestCheatSummonFreshIdentityNearestPlacementAndColdLoad`, `TestCheatCursePreservesCurrentPoolsAndResetsSourceAttributes`.
- Holding bounds: `TestCheatItemRefusesExpandedValuesAndCumulativeMergeAtomically`, `TestCheatSummonRefusesEffectExpansionBeforeAllocation`, `TestCheatPickupRefusesAggregateExpansionBeforeAnyTransfer`.
- Summon dispatch: `TestCheatSummonJoinsSavedDispatchAndPreservesExactPlayer`, `TestCheatSummonRefusesExhaustedSavedGroupsAtomically`, `TestCheatSummonCreatesNativeGuardGroup`.
- Installed definition construction: `TestCheatItemKeepsFactoryEffectsAndExactInstalledNames`, `TestCheatItemMatchesExactInstalledCP866Alias`, `TestCheatActorUsesItsExactRowAndCompleteConstructor`, `TestCheatActorPrefersUnitsForSharedNamesAndHeroFlag`, `TestCheatActorFallsBackToHumansWhenUnitTypeIsZero`, `TestCheatActorCanSummonAndColdLoadWithoutNativeBacking`, `TestCheatCreatureKeepsOrdinaryDifficulty`.
- Launch option: `TestChickenFlagDefaultsOffAndAcceptsBooleanForms`, `TestChickenFlagReachesTheFrontEnd`, `TestChickenSettingDefaultsOffAndRoundTrips`, `TestChickenCheckboxIsOffAndPlayForwardsOnlyItsFlag`, `TestReleaseChickenLaunchAtEveryMissionStart`.
- Screenshot input, bounds and publication: `TestCheatAltScreenshotBypassesChatAndMissionPopups`, `TestScreenshotAltQueuesOneDetachedFrame`, `TestScreenshotFailureConsumesRequestAndAllowsNextFrame`, `TestScreenshotBoundsRefuseBeforePixelAllocation`, `TestScreenshotPublicationPreservesExistingNames`, `TestScreenshotOutputRequiresExplicitSafeDirectory`, `TestScreenshotInvalidFrameCreatesNoOutput`, `TestScreenshotPublicationFailureRecoversPrivateStaging`, `TestScreenshotKeepsLaunchDirectoryAfterSaveBrowse`, `TestScreenshotRejectedPathLogsAndConsumesCapture`.

The installed mutation witnesses write an ordinary SAV, cold-load it through the game route and check the affected state. The summon witness checks the source face and class axes, current equipment picture, App pane pixels, retained original hero metadata, selection, App move admission and movement across subsequent dispatches. The focused binary controls cover internal deterministic continuation separately.

The campaign cheat-victory witness presses Victory through App input in main mission 10, reaching successor 20, main mission 30, reaching town, and side mission 151, reaching town. The main-town and side-town cases also use Continue followed by later Victory. It checks the unchanged simulation outcome and world hash before acknowledgement, then destination, mission-done state, chapter and payment through ordinary successor or town SAV and cold LOAD. These controls check Againrom continuation; the original SAV-latch consequence remains Medium/Unknown.

The kill witnesses establish health -50 and valid ordinary SAV continuation. They do not establish original death timing: the current ordinary property mutation applies the fallen, action and defence transition immediately, while the researched helper writes only health and later-tick death remains Medium (DIV-2077). A faithful deferred state requires native and SAV reader admission plus SAV action projection and their continuation gates. The brief excludes SAV writer changes, so this timing gap remains open.

The screenshot publication test supplies a completed App frame and checks input and PNG publication separately from runtime capture. The external `framecapturewitness` exercised real Ebiten `RunGame` and `App.Draw` on the installed ROM1 EN and RU inputs. After two updates for each input, Alt+S produced a 1024x768 PNG with 47,207 distinct colours for EN and 47,164 for RU. Visual inspection confirmed installed map art, fog, interface and pointer in both captures, including RU Cyrillic text. The original screenshot writer remains unread.

Required candidate gates:

- `gofmt -l` clean and ordinary tests of every touched package.
- `go test -trimpath -count=1 ./...` once.
- FULL release set through `scripts/check-release-tests.sh`, once on EN and once on RU.
- `scripts/check-no-game-assets.sh`.

Execution receipts, elapsed times and the exact committed candidate are returned with the external evidence required by the brief.

## Open debt and Unknowns

DIV-2540 through DIV-2544 and DIV-2546 through DIV-2550 record the authored lifetime, item and actor name sets, placement, missing targets and events, chat presentation, console reply presentation, statistics and trace consumers. The original mission/town privilege lifetime, Safe mode SAV lifetime and unread statistics must not be claimed as established.

The retained side-mission Victory/Continue panel differs from MENU-109's High mission-end-message arm and remains DIV-2545, FIDELITY-DEBT. Campaign completion uses the ordinary real-win route. The saved-latch consequence remains Medium/Unknown; no explicit override is added for it.

The current single-player client has no phase-3 host console or remote connection lifecycle. `#disconnect id` and `#curse id` cannot be entered through that original route. The curse simulation mutation has a focused test, but that does not implement the host UI or network effects (DIV-2551, FIDELITY-DEBT).

Cheat kills apply the fallen, action and defence transition before the next tick. This known timing gap is DIV-2077, FIDELITY-DEBT. A faithful HP-only intermediate state needs native and SAV reader admission plus SAV action projection and their continuation gates. The brief excludes SAV writer changes; those changes require separate compatibility work.

Chat's native Backspace stored-row restore and byte-prefix edit remain DIV-2290, FIDELITY-DEBT; the former no-chat wording is removed from that row and DIV-2078. F1 opens help over chat and retains the draft. Original screenshot output format, naming, destination and outside-mission admission remain Unknown in DIV-2546.

The former console absence and the level-15 cheat producer absence no longer describe this implementation. DIV-2454 is closed. ROM2 cheats and the other command-line switches whose effects remain unread in `MENU-113` are not claimed here.
