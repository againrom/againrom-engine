# The second game's speech lines and wound voices

## Intent

Give the second game's town talks their recorded lines and its units their
own wound and fall cries, through the builders the first game already uses,
and narrow the rows that stood on readings the claims now give: DIV-4438,
DIV-4347 and DIV-4348.

## Authority

Base `6469d619b` (engine main), knowledge pinned k408 on this branch. Claims:

| Claim | Confidence | Decides |
|---|---|---|
| `R2-ENGINE-802` | High; Medium census model, archive lookup, hero unit | page names, families, formats, letters, sound=, first page and Next |
| `R2-ASSET-208`, `R2-ASSET-209` | High; Medium: the `\` to `/` and case fold, that each looked-up name is present | archive keys; EN 743 requests, 742 names, 176 sections; RU 734, 734, 174 |
| `R2-ENGINE-721`, `R2-ENGINE-804` | High; Medium event names; Unknown senders | easy, hard, dead of the voice; who is voiced; the transitions |
| `R2-ENGINE-718`, `R2-ASSET-185` | High; Medium meanings | the voice by identity word; the files each edition holds |
| `R2-ENGINE-803` | High body; Medium draw and path; Unknown item mapping | shop item speech: not built (DIV-4453) |
| `R2-ENGINE-805`, `R2-ENGINE-806`, `R2-ENGINE-807` | High; Medium: that the slot calls no other code; Unknown: the `R2-ENGINE-805` Unknowns | DIV-4347 and DIV-4348 narrowed; the pick-up key (DIV-4455); the RU test (DIV-4456) |

The first game's town speech (DLG-SOUND-028) and wound voices (ANIM-094) are
unchanged.

## As-built behaviour

Town speech. `townSpeechTagPath` (`pkg/game/speech.go`) is the one name of a
town page's recording. It reads a `townSpeechNames` rule from the host's talk
profile (`pkg/game/towntalk.go`). The first game's rule lists the inn, shop
and training directories and names the text file's leaf, as before. The
second game's rule lists the families talk, about, accept, reject, keeper,
man and guard with their directories (keeper, man and guard under `town`,
about with `t000`), hero NPC 21, the letters f, a, m, s by female + 2 × mage
and the plain letter t. A page names `<dir>/<npc 4 digits><letter><key digits
padded to 3>p<part 2 digits>`; a header's sound= names the recording under
`speech/`. The letter is the hero's when the header names NPC 21 or carries an
`iam*` token; the hero is the starting hero the `iam*` tokens read. The talk
cursor keeps the section key (`talkCursor.openSection`); `setTownDialogue`
resets the line; `TownDialogueActive` follows the campaign's `talking()`, so
the first game's room test is unchanged. The three second-game town hosts set
`dialogue-speech` true. A page plays at speech volume on the speech device at
the first page and each Next and stops at the next page or the talk's end. A
section with no family, and every global map, quest or mission text, is
silent.

Wound voices. `unitVoiceRules` (`pkg/game/commandvoice.go`) gains `wounds`,
the leaf of each hurt event, and `woundWords`, identity words that voice
wounds on any unit. The first game's rules name easy, hard and die; the second
game's easy, hard and dead under `characters/`, and words 1..23. `voiceBank`
(`pkg/game/humansound.go`) asks the one selector `unitVoice` under the
mission's rules; `ui.MapEntity.HurtLeaves` carries the leaves to the shared
hurt events in `pkg/ui/sound.go`, whose transitions already match
`R2-ENGINE-804`. A creature whose word is 0 or that has none keeps its drawn
class's Sound array. An absent file plays nothing.

No new random draw; no hashed state moves; no SAV byte changes.

## Proof

Failing first (`cc64ef9c7`): `TestSecondTownSectionSpeaksTheRecordingItsFamilyNames`,
`TestSecondTownHeroPagesTakeTheHerosLetter` and
`TestSecondGameWoundsPlayTheUnitsOwnVoice` fail on the base and pass after
`b0a38910e` and `62e2916df`. The first game's town speech, shop speech, wound
and voice tests pass unchanged.

Census: `campaigncensus -speech` (`SecondTownSpeechCensus`,
`pkg/game/speech_witness.go`) walks every `town.txt` section through the
town talk's page reader and the name builder for each hero sex and class and
either gate answer, read-only.

| Root | Sections | Requests | Names | Missing | Shop directories | Other unrequested |
|---|---:|---:|---:|---:|---:|---:|
| rom2-en | 176 | 743 | 742 | 0 | 380 | 11 |
| rom2-ru | 174 | 734 | 734 | 0 | 381 | 20 |

These equal `R2-ENGINE-802` and `R2-ASSET-208`: EN 380 shop-directory names
are 379 shop formats and `shop/s01i9p2`, so 12 EN and 20 RU names are
unreachable, the claim's table.

Release test, both ROM2 roots: `TestReleaseSecondTownAcceptSpeaksTheClaimedRecordings`.

## Open debt

- DIV-4453: the bank shop's item speech; the item-to-file numbers are unread.
- DIV-4454: the hero unit lookup, the silence switch's writer and the archive
  lookup callees.
- DIV-4455: the second game's `P` key sends pick-up with a reply; the engine's
  `P` is Patrol.
- DIV-4456: wound senders, the state 2..5 arms, the RU `+0x3f4` test and a
  creature's `+0x1dc`.
- DIV-4347 and DIV-4348 stay open on their narrowed readings.
