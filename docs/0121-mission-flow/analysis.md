# 0121 — mission flow: analysis

Two defects that look unrelated and are one subsystem's gate: a player cannot start a campaign
mission from inside the game, and a hero who dies leaves the mission running.

## What was checked before anything was designed

The handed premises came from a subsystem census. Each was re-derived here, on `67d2e57`, and the
line numbers below are this tree's rather than the census's:

- `MissionOpener` (`pkg/game/frontend.go:631`) is complete and is reached from **one** call site,
  `cmd/againrom/main.go:179`, behind `-mission <n>`. Confirmed.
- The picker's rows are `f.Maps`, built by `BuildMapList` (`pkg/game/frontend.go:349`), and its
  archive half lists every `.alm` entry under the scenario identity — the 28 campaign maps.
  Confirmed.
- `f.loadMap` (`:472`) is the only loader `ui.NewApp` is given (`:452`), and it always ends at
  `openMapWorld`, which calls `mapload.FromALMWith` and compiles no script. Choosing mission 10
  from the list therefore gives scenery. Confirmed.
- The two closures return the same eight values, and differ in exactly one: `loadMap` hands back
  `nil` for `ui.MapAdvance`, `MissionOpener` hands back `mw.advanceNotice`. Confirmed — so the
  routing is a delegation, not a merge.
- `MissionOpener(n)` reads `scenarioPrefix + itoa(n) + ".alm"` (`MissionMap`,
  `pkg/game/mission.go:38`) and `mapBytes` reads `scenarioPrefix + e.Source` for an archive row.
  For a row named `10.alm` those are the same address, so a mission row can delegate to the
  existing door with nothing recomposed.
- `w.lost` has two writers, `pkg/sim/script.go:669` (check 18, the VIP rule) and `:831`
  (instant 5), and `scriptReport` (`:1042`) reads only the two counters. Nothing asks about a
  hero. Confirmed.
- **The stale comment is at `pkg/game/world.go:988`, not `:938`.** The census read `b4e47da`; the
  sentence is the one it names — *"A map opened from the picker advances under its own script and
  the AI alone"* — and `openMapWorld` compiles no script, so it is stale as reported.
- The campaign is **28** maps. `MISSION-ROOT-017` and `MISSION-WIN-003` both say 28 of
  `scenario.res`, and `pkg/game/maplist.go:102` says the same of a stock install. The pipeline
  ledger's 29 is wrong; no third answer was found.

## The question the design turned on

`MISSION-END-013` tests **the player's hero**, not the player's last unit. This tree has no hero
concept below `pkg/game`: `sim.Entity` carries `Owner` and nothing that distinguishes a party
member from a map-placed unit at the same roster slot, and `mapload.Start.IDs` — which does know —
lives two tiers up.

Three ways to give `pkg/sim` the hero were considered and two were rejected.

- **A hero-id field on `sim.World`.** Correct, and it bumps the byte form: the script program is
  itself serialised precisely because *"a world carries no map to re-derive it from"*
  (`pkg/sim/scriptbinary.go:19`), so anything the reporter reads must round-trip. No byte-form
  version is allocated to this story, so this is not available here.
- **"The roster slot `SelfSlot` has no living actor."** Needs no new state and is wrong on the maps
  that matter: 5 of 28 campaign maps place units for the player themselves
  (`MISSION-START-001`, cited at `pkg/game/frontend.go:751`), so on those the hero could die and
  the mission would run on — which is the defect being fixed.
- **The test in the front-end driver, where `Start.IDs` already is.** Taken. It costs the sim's own
  `Outcome()` staying blind to the hero, which is disclosed in `provenance.md` as the seam a
  byte-form story should close.

## What the routing change makes cheaper later

Nothing about `openDifficulty` or `startColumns` — both were left alone, as briefed. But both doors
into a mission now spell `openDifficulty` and `MissionParty(...)` at the same two call sites
(`frontend.go:647`, `:720`). A chargen story (MM3/MM4) that wants a chosen hero and a chosen
difficulty changes those two lines and no others; before this story it had to invent the door as
well.
