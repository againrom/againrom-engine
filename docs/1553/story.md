# The engine writes the Kaarg town file the original plays on from

## Intent

The second game's Kaarg town file is written after the engine's own play, keeps the companion
the Kaarg inn talk joins, and writes the tail words a loaded file held. After a LOAD of an
original mission file the campaign's party is the World's hero, and a mission file holding that
companion restores him as the party member identity 22. The Druids' town (town 3) is not in this
story.

## Authority

- Owner, in the brief: a field takes its value from the current state, then a rule from claims,
  then the loaded document's bytes; a field with no source is a DIV debt row; a part with no claim
  is a research question.
- Owner observation, through the coordinator: in the original, every inn talk in an engine-built
  Druids' town file without members 22 and 23 ended in «FATAL ERROR: can't load
  graphics\equipment\fmage\-58.256» (once `-57.256`). -58 and -57 are 22 - 0x50 and 23 - 0x50, the
  fallback actor's +0x28 of `R2-ENGINE-582` for speakers 22 and 23. Inference, Medium: the
  original's talk needs party members 22 and 23. The owner names them Iglez and Dina
  (`R2-ENGINE-718` voices `igles` and `dina`). Dina is outside this story.
- Claims: `R2-ENGINE-105`, `R2-ENGINE-106`, `R2-ENGINE-525` (party identities, WORD +0x14c),
  `R2-ENGINE-319` (template row and main skill), `R2-SESSION-283` to `R2-SESSION-287` (the town
  writer), `R2-SESSION-032`, `R2-SESSION-033`, `R2-SESSION-035` (the tail's places and pairs),
  `R2-ENGINE-418` (catalog positions), `R2-ENGINE-625` (a party identity's fall gives identity
  - 19 as the failure reason).
- Owner, earlier (DIV-4224): "When Iglez dies, show mission 30's own notice that he is dead?":
  "Yes, his own notice".

## As built

| kind | builder | change |
|---|---|---|
| SAV document producer | `ExportCurrentSave` | unchanged route; its second-game form writes the companion's main skill (`secondTemplateHero`) |
| second-game campaign tail | `secondTownTail` writes it; `pkg/formats/sav/secondtail.go` encodes it; `secondFileSession` reads it | the loaded tail's pairs, the two words after them and each listed place's 16 raw bytes are held as the campaign's `Kept` and written back |
| campaign departure rules | `completeBank` | unchanged |
| second-game party and its identities | `secondparty.go`, read by `restorePartyCharacters` | a loaded Human whose U148 is a bound party identity and whose row is that identity's serverID row is that companion (`originalPartyIdentity`) |
| inn hire | `secondinn.go` | unchanged; an identity companion is not read as a hire |

- Companion at LOAD (`pkg/game/originalparty.go`). A file read from its records alone keeps
  the companion as `npc:<identity>` with `CompanionNPC` set; his identity word is not a mercenary
  type. Before, `persistentOriginalCharacter` dropped him, and reading U148 22 as a mercenary
  type made him a hire with row 5.
- Groups at LOAD (`pkg/game/citygroupcarry.go`, `originalrestore.go`). A file-session town takes
  the Player's groups from the file's actor graph (`cityGroupsFromGraph`), through the same
  remap as an engine document's (`loadedGroups`). Before, the town had no groups and one
  constructed group held every member (DIV-1686).
- Companion at SAVE (`pkg/game/savform.go`). A Human whose row is a `_Hero` or `Start_`
  template holds his largest school as main skill (`R2-ENGINE-319`).
- Tail (`pkg/game/currentsecondcampaign.go`, `secondcampaign.go`, `savform.go`). The campaign
  holds `Kept` (JSON `Kept`, omitted when absent; an older document has none and writes the old
  [[1,1]] and zero).
- Mission LOAD (`pkg/game/originalrestore.go` `planMission`). The campaign's carried party is the
  file's restored party, named, with mission positions cleared. Before, `NextParty` gave a new
  campaign's hero ("Danath" in the fixture, "Aldor" on the install) while the World held
  Gildarius.
- Companion in a mission file (`restorePartyCharacters`, shared by the town and mission LOADs).
  A LOAD of an original second-game mission file restores the Human that passes
  `originalPartyIdentity` as the party member `npc:22` with `CompanionNPC` 22, beside the hero,
  and binds his actor to him (DIV-4336). For the player: Iglez is a party member in the loaded
  mission, as when he joins in the engine's own play; his fall loses the mission with reason 3
  and mission 30's own failure entry (`R2-ENGINE-625`, DIV-4224); the mission's end carries him
  home as identity 22. Before, he was excluded as a temporary ally and rebuilt from the saved
  population as a map-joined ally `join:<entity>`, whose fall gave the script's reason. The World
  hash of each such LOAD changes, and in the owner's `game0017` the party's entity numbers are
  reassigned (Iglez 79 to 77, the two hires 77 and 78 to 78 and 79). A mission SAVE after the LOAD
  writes him as one more party Human of the engine's mission document (DIV-2410): a 242-entry
  Diary record, and every record after it renumbered by one. His name stays the file's empty name
  (DIV-4337).

## Proof

Focused tests in `pkg/game/secondtownfile_test.go`:

| test | 81e26410 | candidate |
|---|---|---|
| `TestSecondTownFileKeepsTheJoinedCompanion` | FAIL: records-alone LOAD carries identities [21] | PASS |
| `TestSecondTownTailWritesTheLoadedWords` | FAIL: pairs [[1 1]], words 0, zero raw bytes | PASS |
| `TestSecondMissionFileLoadCarriesTheWorldHero` | FAIL: "Danath", mind 15 | PASS |
| `TestSecondTownFileAfterMissionTwentyLoadsBack` | PASS | PASS |
| `TestSecondMissionFileLoadKeepsTheJoinedCompanion` | FAIL: the mission's party is [hero] | PASS |
| `TestSecondMissionFileVictoryCarriesTheCompanionHome` | FAIL: the party home carries identities [21] | PASS |

The last two LOAD a synthetic original mission 30 file holding the hero and identity 22 (Humans
row 30, U148 22, no name, a group of his own): the first checks `npc:22` in the mission's party
and the campaign's, and that his fall loses with reason 3; the second wins the mission and checks
him in the party carried home.

The last one is coverage: a LOAD of a mission 20 file, its victory, the map's travel to Kaarg and
SAVE already LOADed back equal on 81e26410, because the victory carries the World's hero.

Inspection, not a test (tools by `scripts/build-tools.sh`; inputs are copies):

- `againrom --headless`, EN and RU: LOAD of the corpus `game0010` (mission 20 won), victory,
  town 2, SAVE. The file LOADs back at 2/2 with Gildarius, 44 gold and the bag; its resave's
  body and tail equal the file's; the EN and RU files differ only in the label. Against the
  81e26410 build only the tail's two words differ (0, 0 there; 93, 6 here).
- `savecheck -party -resave load` of the owner's `game0015`, `game0016`, `game0051`, `game9998`
  and the corpus `game0011`, EN and RU: every resave's tail equals the file's (81e26410: place
  raw bytes zero, `game0011`'s words 0, 0); `game0016` and `game9998` keep identity 22 with two
  groups; `game0051` resaves byte-equal; EN and RU resaves are equal.
- `savecheck -party -resave load` and `-party -campaign-progress load` of the four original
  mission files holding identity 22, on 81e26410 and on the candidate, EN and RU, then
  `savtool compare` of the two resaves. Each World hash is equal on EN and RU:

  | file | mission | 81e26410 | candidate | resave records |
  |---|---|---|---|---|
  | corpus `game0012` | 30 | `f2da4fc3e8398dbd`, `join:125` | `d58a2caa7e4296ef`, `npc:22` | 309 to 310 |
  | corpus `game9999` | 30 | `39a91aff0456e460`, `join:125` | `941135f844acfa4d`, `npc:22` | 309 to 310 |
  | owner `game0017` | 31 | `1650509609203834`, `join:79` | `9be17bf23e4d4039`, `npc:22` | 269 to 270 |
  | owner `game9999` | 31 | `3b73dd6807733425`, `join:77` | `305ec97d203e8584`, `npc:22` | 269 to 270 |

  The record added is Iglez's Diary; the Weapon, Armor and later records after it move by one.

Release witnesses the next release must run on `rom2-en` and `rom2-ru`:
`TestReleaseSecondOriginalTownLoadRoundTrip`, `TestReleaseSecondOriginalTownLoadFollowsTheFile`,
`TestReleaseSecondOriginalMissionLoadIsRefused`, `TestReleaseSecondTownNamedSaveContinuation`,
`TestReleaseSecondTownSaveEntryPoints`, `TestReleaseSecondTownTwoContinuation`,
`TestReleaseSecondMissionTwentyFailureDoesNotLeave`, `TestReleaseCampaignActorInnIdentity` and
`TestReleaseCampaignActorFireAfterTownLoad`, and `TestReleaseSecondOriginalMissionFileKeepsTheCompanion`,
which LOADs the corpus mission 30 files `game0012` and `game9999` and the owner's mission 31 files
`game0017` and `game9999` from `gameversions/saves/2026-10-30` and checks `npc:22`, his reason 3
and the campaign's identities 21 and 22. The mission LOAD's carried party and the shared group
remap also reach the first game's original LOAD, so on `en` and `ru` also
`TestReleaseMilestone2Groups1155`, `TestReleaseMilestone2CurrentProducers1162`,
`scripts/check-milestone2-acceptance.sh` and `scripts/check-sav-roundtrip-gate.sh`.

## Open debt

- DIV-4305: the companion's T1C, T08, URaw184 bytes and his worn pieces' T0E, F47 and Block12
  have no source; record keys are the engine's own.
- DIV-4306: the tail's two words after the pairs follow the last loaded file, where the original
  changes them (93, 6 in mission 20; 184, 6 in Kaarg after it); a place the loaded file did not
  list writes zero raw bytes where the originals write x, y, x, y of its catalog position.
- DIV-4336: the original's own LOAD of a mission file holding identity 22 is not claimed; the
  engine reads his identity word by the town LOAD's rule.
- DIV-4337: Iglez read from an original file keeps the file's empty name, where the engine's own
  join names him from the npcnames line; the name the original shows has no claimed source.
- Member 23 (Dina) and the Druids' town are other stories.
