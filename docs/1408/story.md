# A mission from a mod played by a guest party

## Intent

A mod adds a mission that another character plays alone between two campaign
missions. Nothing of the hero or the party is lost, and that character joins
the party at the next town with what she gained. Prototype: the `reniesta`
fixture mod plays a copy of the installed mission 10 map after mission 20,
with the mage of Plagat. Base: public main `e56dff9e`; knowledge pin k258
from main.

## Authority

Owner direction; no ROM1 research. Every difference is a row: `DIV-3023`
(the guest mission), `DIV-3024` (its SAV), `DIV-3025` (what carries back),
`DIV-3026` (leaving it). Seat rulings: the guest is companion 22 as the town
of chapter 30 would grant her over the stashed party (the mage Reniesta for a
male hero, her counterpart for a female hero); the stash is SAV bytes on every
route; the no-loss digest compares the campaign at the stash with the
campaign after the restore.

Facts checked in code and data: missions 10 and 20 precede the first town
(`Campaign.TownBegins`); mission 10 declares 20 (`AutoAdvance`); the win of
mission 20 offers the town of chapter 30 (`NextMission`); chapter 30's
`AddHero` is `[22]`; with the male release hero the chapter-30 grant is a mage
(probe on the EN root).

## As built

### Data

`data/missions.toml` (`pkg/mod/missions.go`, documented in `docs/MODS.md`):
`[[mission]]` with `key`, `number`, `map`, `texts`, `title`, `briefing`,
`after`, `guest`, `chapter`, `gold`, `carry`. Numbers 1..65535, gold
0..1000000, paths inside the mod, `{lang}` replaced by the game's language,
title and briefing text keys of the mod. The loader (`pkg/modrt`) reads the
map (at most 1 MiB, `mod.ReadInside`) and at most 64 `*.txt` text files when
the mod loads; two mods cannot use one number or one `after`. A refusal names
the mod, file and line.

`FrontEnd.SetModMissions` refuses a number that is a multiple of ten, a
campaign mission or an installed map; an `after` that is no main mission,
auto-advances, or opens no town; a `chapter` other than the town `after`
opens; a `guest` that chapter does not grant. It overlays the map at
`scenario/<number>.alm` and the texts at `main/text/battle/m<number>/` on the
front end's own forked filesystem; title and briefing are encoded into the
install alphabet (`EncodeInstallText`). The launcher calls it after
`SetModBodies`.

### One builder per kind

- Mission start: `enterMission`. The map and texts reach it by address
  through the overlay; its `commitDriver` skips `selectMission` while the
  guest plays, so the waiting campaign keeps its selected mission. `MissionOpener(n)` opens a guest mission with
  the guest's party and purse (`missionOpener` with `initialPurse`), which is
  the party choice of the one opener, not a branch in the mission.
- Party into a mission: the same placement and city seeding
  (`seedCurrentCityObjects`). The guest leads her party of one
  (`StartingHero` while she plays; cleared when she joins).
- Companion join: `addChapterCompanions` -> `carryTownCompanion`. The
  definition row is built by `townCompanion` (split out of
  `carryTownCompanion`); a waiting guest of that chapter and companion is
  taken instead. A held grant (`reniesta-joins-on-talk`) holds her through
  `joinOnTalk` unchanged.
- Save: SAV through `ExportCurrentSave`; load through `RestoreOriginal`.

### Stash and return

`recordWin`'s town branch asks `townArrival.guest` first. For a declared
`after`, `startGuestMission` builds the guest with `townCompanion` over the
carried party, latches the town (`Town.Arrive`), keeps the return city, takes
`Snapshot(false)` and writes it with `ExportCurrentSave`. The bytes are the
stash in `Town.guest`. The shop and the chapter grant are not run:
the town LOAD runs both. A stash failure falls back to the ordinary arrival
and names the reason in the win line.

The guest's win (`FinishMissionWithRoster`) carries her home through
`carryMissionHome` and LOADs the stash through the one LOAD path
(`restoreOriginal`, behind `RestoreOriginal`) with her carried member as the
guest record in place of the stash's own. The load adopts the town, its random
session and the guest; `arriveInTown` stocks the shop
and grants companion 22, who is the carried guest. The world map shows the
return from `after` (`returnMission`). While the guest plays, the waiting town
does not select her mission (`activateLive`).

### SAV fields

`currentSessionData.Guest` (`currentGuest`): mod, key, mission number, the
stash while the mission runs, the guest's start, and her carried member once
she has won and waits for a held grant. A mission SAVE writes the native head
with the guest mission's number; the campaign records are the waiting town's,
and the guest mission is never the selected campaign mission. LOAD refuses (`restoreGuest`, `decodeOriginalSource`) when the mod mark
differs, when no active mod declares the recorded mission under its number,
or when a SAV of a declared mission number carries no guest record.

### Assembly tool

`cmd/modmission -install <root> -map <n> -out <mod folder>`
(`AssembleModMission`): copies `scenario/<n>.alm` and the installed
`event*.txt` and `tips*.txt` of `m<n>` into `mission/<lang>/` of a mod folder
that holds `mod.toml`. It refuses an `-out` inside the install and writes
nothing else.

## Proof

Release tests, EN and RU (`pkg/game/modmission_release_test.go`), headless:
they drive the production route from a new game to the town with no window.

- `TestReleaseModMissionGuestPlaysAndJoins`: the control route without the
  mod reaches chapter 30 with companion 22 from her definition row. Under the
  mod, mission 20's win answers mission 25 (`winMission`); the guest mission
  opens `scenario/25.alm` with the guest alone and purse 0; she kills an enemy
  through the attack order; the map driver's continuation leads to the town
  with the world map returning from mission 20. The digest (every other
  member whole, purse, progress, pending grants from the town SAV, hired
  pools, documents, fame, session choices, chapter) equals the control's.
  She joins with the experience she ended with; the definition row does not
  hold it. A mission SAVE and cold LOAD inside the guest mission gives the
  same digest and the same joined member. Loss controls: a stash with one
  more gold fails the digest; a LOAD without the mod refuses naming it; a
  LOAD of the mission SAV with its guest record removed refuses.
- `TestReleaseModMissionGuestWaitsForAHeldGrant`: with a female hero the
  counterpart the town grants plays; under `reniesta-joins-on-talk` she waits
  after her win, the town SAV holds the held grant and her record, a cold
  LOAD keeps her waiting, the tavern talk joins her in that state.
- `TestReleaseModMissionReadsTheMapFromTheMod`: a changed altitude byte in
  the mod's map reaches the built mission; the install's mission 10 map keeps
  its own.

Focused tests without an install: the parser and loader
(`pkg/mod/missions_test.go`, `pkg/modrt/missions_test.go`),
`SetModMissions` refusals and overlay, `recordWin` with and without a guest,
the join, `restoreGuest` refusals (`pkg/game/modmission_test.go`).

Measured on the EN root: a town SAV at the stash point is 13154 bytes (limit
8 MiB). On the new-game route a town SAV round trip keeps every party
member's JSON, the gold and the progress; it changes representation only
(see Open debt).

## Open debt

- A town SAV round trip changes three representations with no player effect:
  a fresh town's chapter records become the restored campaign projection, a
  consumed grant is recorded in the `AddHero` array instead of the consumed
  chapter list, `OriginalCity` provenance appears and party city objects gain
  zero human tails. The digest reads the pending grants from the SAV for this
  reason. On a party loaded from an original SAV (owner `game0007.sav`) the
  same round trip also rewrites `NativeRecord.Token.Reference` of the hero's
  carried items (62 paths); a town SAVE and cold LOAD without the mod does
  the same, so the guest route adds nothing to it.
- A mod map that does not decode is refused only when the guest mission
  opens, after mission 20's win has stashed the party. `data/missions.toml`
  is not restricted to the first game.
- The guest mission's world map has no marker; the briefing panel and event
  texts come from the overlay only.
- Only `carry = "join"` exists; one guest, a party of one.
