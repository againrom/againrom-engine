# Story 1228 — SAV round-trip gate

## Intent

Owner direction, `pipeline/SAV-ENDGAME.md` execution order item 8, first part:
one gate runs GAME -> SAV -> LOAD -> GAME over the save points this project
reaches without a window and compares the hashed World, what the player sees
and, for a town, the town state. Earlier round-trip tests compared only the
simulation World for single fixtures, so a hero redrawn as an NPC after our
SAVE and LOAD passed them.

## As built

Two tests in `pkg/game/savroundtripgate_corpus_test.go`, build tag
`sessioncorpusaudit`, run by `scripts/check-milestone2-acceptance.sh`'s
runner, one process per test and root under `AGAINROM_M2_JOBS`:

- `TestSAVRoundTripGateNewGameKits`: `newgame`, `kit resave`, `kit played`,
  `kit cycled`.
- `TestSAVRoundTripGateCorpus`: `corpus town`, `corpus resave`,
  `corpus played`, `corpus dying`.

`AGAINROM_M2_SAVGATE=skip` leaves the gate out of the milestone-2 run;
`scripts/check-sav-roundtrip-gate.sh` runs it alone, so a chain can run it as
its own leg. Unset, the milestone-2 run includes it. Each test runs its cases
on four workers, every front end its own.

| population | route per case |
|---|---|
| `newgame` | missions 10 and 20: tick 0; 300 ticks and a ground drop; five kills and full decay; the same sparing entity id 0 |
| `corpus town` | every corpus file with no World (a between-mission save): restored, saved through the SAVE dialog off the map (`OnMap: false`), restored on a fresh front end |
| `corpus resave` | every corpus mission file, saved as loaded; a file `sav.Open` cannot read is one unreadable case here |
| `corpus played` | the decay sample of missions: 30 ticks, a ground drop by the first party member, five kills, full decay |
| `corpus dying` | every other corpus mission: the same play, saved 2400 ticks after the kills with the bodies mid-decay |
| `kit resave`, `kit played` | files named by `AGAINROM_SAV_ROUNDTRIP_KITS`; played is the full-decay route; a pattern matching nothing fails |
| `kit cycled` | each kit, no drop: kill 12, full decay, SAVE, LOAD; the loaded copy kills 12 more and repeats, three rounds; each round compared (`cycle.*` keys) |

Full decay waits until every killed body has left the world. A killed actor
the world still holds alive and not decaying did not die; it is logged and not
waited for (entity 0 on `kjill.sav` and `saver.sav`).

The decay sample is one corpus file in four by FNV-32a of its relative path,
so adding a file moves no other file between populations. Today it holds 14
missions. The dying route proves drop, kills, dying bodies and decay stages;
the fully decayed state is proved on the sample, the New Game kill cases and
every kit that completes a round.

### Comparisons

1. Mission: `World.Hash`. When the hashes differ, every World field is walked,
   unexported ones included. A field that differs is copied from the saved
   copy into a shallow copy of the loaded World; if that moves the loaded
   hash, it is hashed state, and each differing leaf path under it is a key
   (`world.originalDead.OriginalDeadActor.Source.ContainerPresent`). Slice and
   map indices are dropped from paths. A second loss in a World that already
   differs is therefore its own key. `world.other` names a remainder no field
   explains.
2. Visible state: each unit's drawn art, composed figure record, portrait
   (the picture production draws, by entity class) and hover picture pixel
   hashes; each scene entry's cell, art, frame pixels and draw state; each
   sack's frame index; camera X, Y, zoom; the CPU-composed character pane.
3. Both copies advance 120 common ticks. When LOAD agreed on the hash, the
   hashes must agree on every tick (`advance.hash`). When it did not, the
   hashed field paths are named again after the advance (`advanced.world.*`).
   The visible comparison is repeated with no exemptions (`advanced.*`).
4. Town: `currentRoundTripSnapshot(f, false)` of both front ends, every
   exported field walked as above (`town.Gold`, `town.Party.Weapon.Name`).
   `OriginalCity` and `CityObjects` are not compared: they carry the loaded
   document and its object topology, which our writer rebuilds by design.

Exemption at LOAD only: a unit caught mid-step draws from the view's pre-step
cell, which no save carries (`TRANSIENT`). A refusal in a later cycle keeps the
earlier rounds' mismatches.

### Baseline

`pkg/game/testdata/savroundtripgate-newgamekits.txt` and
`savroundtripgate-corpus.txt` record each case's population, outcome and
every key it carries. Written with `AGAINROM_SAV_ROUNDTRIP_RECORD=<dir>`.

- A recorded case fails on a key it did not carry, or when its population or
  unreadable state changes (a mission that restores as a town is a refusal).
  A recorded key it no longer carries is logged `FIXED`.
- A case not in the baseline fails on any key its population does not carry in
  some recorded case. A new corpus file that shows only named defects passes.
- A recorded case that does not run fails (`MISSING`), so an emptied or
  shrunk corpus fails. `AGAINROM_SAV_ROUNDTRIP_ONLY` skips this check.
- Every recorded key needs a cause in `savGateCauses`, by prefix.

## Census

Engine main a53c140 merged (f22db1e). EN and RU give the same outcome and
keys for every case.

| population | discovered | accepted | unreadable | refused | mismatched |
|---|---|---|---|---|---|
| newgame | 8 | 1 | 0 | 0 | 7 |
| corpus town | 40 | 5 | 0 | 0 | 35 |
| corpus resave | 66 | 59 | 1 | 4 | 2 |
| corpus played | 14 | 0 | 0 | 0 | 14 |
| corpus dying | 51 | 2 | 0 | 1 | 48 |
| kit resave | 5 | 3 | 0 | 2 | 0 |
| kit played | 5 | 0 | 0 | 4 | 1 |
| kit cycled | 5 | 0 | 0 | 3 | 2 |

Before this pass the 40 towns ended as uncompared `town` outcomes in each of
the resave, played and dying populations (80 corpus cases); they are now 40
compared town cases. Kits: `owner-sav-story1226/game9255.sav`,
`owner-sav-story1225/game9252.sav`, `game9253.sav`,
`owner-saver-terminal86/kjill.sav`, `saver.sav`. The unreadable file is
`EXP-0261-owner-runs/game9000.sav` (magic `Bsg&`).

## Sensitivity

Each production change built in a scratch copy of the tree and discarded;
both roots.

| change | gate |
|---|---|
| town gold written one short (`currentsave.go`, `Money` from `s.Gold-1`) | fails: `town.Gold` on 40 of 40 towns |
| retired dead-actor record `Health` one lower (`worldsave.go`, Stage 5, no reference, container or terrain key) | fails: `world.originalDead...State.HP`, `world.originalDead.terminal.HP` and their `advanced.` forms on 15 cases (14 played, `game9255.sav cycled`) |
| player purse written one high (`currentplayerslot.go`, `currentworldbuild.go`, `savplayerpurses.go`) | fails: `world.purses` on 140 cases, including every case whose World already differed at LOAD |
| corpus plus a byte-identical copy of `2026-08-02/game0007.sav` under `added/` | passes; the copy's resave is accepted and its dying case carries only named keys |
| empty corpus | fails: 171 recorded cases `MISSING` |

Earlier, on the gate before this pass: disabling the entry-party skip in
`Snapshot` and the party guard in `missionAppearanceArt` failed on
`visible.unit.body`; disabling the `rootDeadActors` call failed a played kit
at SAVE.

## Timing

`AGAINROM_M2_JOBS=4`, both roots in one invocation, `AGAINROM_GOCACHE` the
seat cache, 20-core seat machine with no other chain running, on be91c3e:

| run | wall |
|---|---|
| milestone-2 run with the gate | 212 s |
| milestone-2 run with `AGAINROM_M2_SAVGATE=skip` | 144 s |
| `check-sav-roundtrip-gate.sh` beside the skip run, both at once | 111 s and 197 s |

The gate tests took 50 s (New Game and kits) and 81 s (corpus) per root
inside the full run. The seat's chain on e73c8bd ran 370-372 s without the
gate, with the milestone-2 leg at 321-331 s; the chain time with either
arrangement is Unknown.

## Findings

Fixed on main since the first census: SAVE refused after a New Game kill, the
`S3C` panic after a New Game drop, the terminal-actor collision at LOAD
(`saver.sav`, `kjill.sav cycled`), and `SavedGroups` keys on played and dying
cases (a body's order is written from current state).

Open, player-visible:

1. After kills, creatures that were not killed gain a composed human figure,
   portrait and hover after LOAD: 4 New Game, 10 played, 40 dying. Inferred:
   `entityFigures` keys by map-unit record index.
2. A killed human mid-decay draws as its class before SAVE and as Unarmed
   Fighter or Unarmed Mage after LOAD; a party member that dropped a worn item
   draws unarmed after LOAD (`game0018.sav` twice, `game0076.sav`): 15 dying.
3. Unit draw frames differ after LOAD: the view scene clock
   (`mapWorld.scene`) restarts at 0 in the loaded copy (traced by the review).
   6 New Game, 12 played, 48 dying, 4 kit cases. Queued as a hotfix.
4. SAVE refused after a ground drop on kits `kjill.sav` and `saver.sav`:
   `new Sack N lacks current cell planes`.
5. SAVE refused in round two of `game9255.sav cycled`: `current actor
   manifest has a repeated or absent source actor`.
6. Our own resave cannot be loaded: `unsupported saved motion lacks its
   explicit issue`, 4 corpus files.
7. `2027-09-07/game0033.sav dying`: our reader refuses our file, `decay stage
   1 on a unit ... which is alive`.
8. Mission 20 camera moves one cell down after LOAD (`viewOriginFloor`): 4
   New Game cases.
9. Kits `game9252.sav` and `game9253.sav` are refused before play: `saved
   structures: roster/source count mismatch`.
10. Town: a party member's weapon name gains `Common` after LOAD (`Iron Mace`
    to `Common Iron Mace`, `town.Party.Weapon.Name`): 5 towns.

Open, hashed or campaign state:

11. `savedMotion`: raw keys re-minted by the writer (`Position.TerrainKey`,
    `Cell`, `Mover`, `ActorAction`, routes): 13 played, 8 dying, 2 resave, 2
    kits.
12. `originalDead`: a fully decayed original-bound actor gains a loot
    container record after LOAD (`ContainerPresent`, `ContainerTail`): 13
    played, 2 kits.
13. `savedSpellGraph`, `savedWorldEffects`: retired nodes and drivers dropped
    by LOAD: 2-3 dying.
14. `script`: on mission 10 after kills, script checks and instants bound to a
    killed, fully decayed unit lost their unit binding after LOAD. Fixed by the
    dead-unit script hotfix (engine a53c140); the baseline no longer records
    it.
15. `entities.ActorLoad.Source.MoverSpeed`: the dying actor's load record on
    `game0017.sav` and its copy.
16. `savedGroups.Orders.Raw` on two resaves (`game0002-bigsack.sav`,
    `game0017-victory.sav`).
17. Town: `Party.Saved.Cell` goes from 0,0 to 16,12 after LOAD on 30 towns;
    `Campaign.Main.MapObject` from 0 to 9 on 11 towns. Player effect Unknown.

## Open debt

- Every finding above; the seat decides which become hotfixes.
- A town SAVE after a mission resumed from a mission SAV is not covered.
  Without a window, completing a resumed mission (`LiveCompleteCampaign`)
  opens no town, and the town SAVE refuses with "there is no game to save
  yet".
- Unit names and character sheets (`mapWorld.actorNames`, `chars`) are not
  compared; the review saw unit 29 of `game0007.sav` renamed `Axeman` to
  `Swordsman` after LOAD. Queued as a hotfix.
- Visible keys are per field, not per unit: a second frame or figure loss in a
  case that already carries that key adds no key.
- A town is compared on its snapshot; no town screen is rendered.
- The sack-frame comparison has no demonstrated failing case.
- The chain time with the gate is Unknown.
