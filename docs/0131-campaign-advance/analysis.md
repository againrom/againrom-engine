# 0131 — the campaign advances: analysis

**Intensity: spec-anchored / static.** The mission-end path is a contract two
tiers consume and it will keep changing — the town, and eventually ROM2, arrive
through it. No watcher tool exists, so the synchronisation is discipline.

**Terrain: brownfield** for the mission-end path — `Campaign`, `FinishMission`,
the driver's notice seam, and the front-end's notice arm all have shipped
behaviour that must not silently break. **Greenfield** for the registry key's
own read and for the headless report.

## What we did not know

Winning a mission puts the player back on the map list holding a sentence, and
he has to find the next row himself. The owner asked for mission 10 to finish
and mission 20 to start, with the same character. Three questions stood in the
way, and each was settled by reading rather than by guessing.

## What was looked at

**Does anything already decide what follows a mission?** Two things do, and they
are not the same fact. `Campaign.NextMission` answers "the next `[Mission<n>]`
section whose number is a multiple of ten, ascending", and its own doc says that
order is ours — nobody has read the original's advance routine. The campaign's
own `AutoGetMission` key was read by nothing outside a cross-check test. On a
stock scenario the two agree at 10 -> 20, which is exactly why the difference is
invisible from the shipped data alone.

**Is the key a ladder or a one-off?** Dumped from both lawful roots. The parsed
`scenario.reg` trees are identical between them, although the containers are not
(`SCENARIO.RES` md5 `52ac6170...` en, `ba67cc75...` ru). Twenty-four
`[Mission<n>]` sections; `AutoGetMission` appears in exactly one of them,
`[Mission10] = 20`. `[Mission20]` carries `Mercenaries = 1` and nothing else. So
the key declares one hop, and what follows mission 20 is the town.

**What does an absent key mean?** This is the question that decides whether the
story can be written at all, and it is answered in the pin rather than by us —
see `provenance.md`. The reader's default and the fork it feeds are both read at
instruction level, so "absent" and "-1" are one state and the story does not
have to author a reading.

**What already carries the character?** `mapload.CarryParty` runs in
`FinishMission` and lands in `FrontEnd.Carried`; `FrontEnd.NextParty` reads it
and `MissionOpener` calls that. So every mission this front end opens **by
number** already starts with the party the last win left behind. The gap is not
the carry — it is that nothing opens the next mission.

## The implicit assumption that was nearly missed

The front-end's map-screen arm returns early after dismissing a notice, on the
test `screen != ScreenMap` — written when every dismissal that went anywhere
went to another screen. An advance that opens a mission leaves the screen on
`ScreenMap`, so that test answers false and the rest of the frame runs against
the newly opened mission with the dismissing press still in hand. Nothing in the
suite would have failed.
