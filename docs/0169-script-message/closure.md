# 0169 — closure

Branch `0169-script-message`, base `2a09324`. Research pin `7d41fa4`.

Gate: `go build ./...`, `go vet ./...`, `gofmt -l` over tracked and untracked Go files, and
`go test -trimpath -count=1 ./...` all pass with no game install present.
`bash scripts/check-no-game-assets.sh` is clean. The deletion set against the base is empty.

## The twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | Event texts are read from `main.res` through vfs at the moment an announcement fires, never at map load. Both preserved roots resolve; RU decodes from CP866 correctly. |
| Runtime state | PASS | The announcer's rising-edge memory and the panel's open/part state. `TestTheAnnouncementPassWritesNothingToTheWorld`. |
| Simulation | PASS | `ScriptInstantMessage` is a named empty arm. 28 campaign digests are byte-identical before and after (below). |
| Player input | PASS | The panel's paging and dismissal are the existing ones and are unchanged. `TestPagingReAppliesTheSameAudience` shows a page cannot show a part the same audience was refused. |
| AI | N-A | No unit decision is reached. |
| UI / HUD | PASS | The dialogue panel opens with the part this hero receives, and does not open when no tag survives. `TestThePanelDoesNotOpenWhenNoTagSurvives`. |
| Triggers / scripts | PASS | Opcode 2 is in the supported set; the census fall is 254. |
| Inventory / equipment | N-A | Nothing is given or taken. |
| Persistence / save-load | PASS | This story adds no serialized field; `formatVersion` stays 50. The persistence-relevant state it does add is the announcer's rising-edge memory, and it survives by re-derivation: `NewAnnouncer` seeds `prev` from the world's own latch array, which is inside the sim's serialized bytes, so an already-latched trigger is not a rising edge on resume. `TestAnnouncerDoesNotReannounceAnAlreadyLatchedTrigger`. An **open panel** is not carried — `SnapshotResidue` holds commanded units, swing/phase, group tag and fog, and no notice state of any kind. That is pre-existing and applies equally to the outcome notice this build has shipped since 0121; this story neither introduces it nor widens it. It is disclosed as `spec.md` SC-6. |
| Campaign / session | PASS | The mission number selecting the directory is the campaign's own, per `DLG-MISSION-026`, never a value carried by the announcement. |
| Shipped content | PASS | All four shipped edge classes are closed below. |
| Interactions with existing mechanics | PASS | The drop-when-open rule, the outcome notice and the 255 collision are each exercised. |

No in-scope GAP remains.

## Integration witness

`cmd/againrom --headless` drives a real campaign mission with no window. Mission 10, 2000 ticks,
with a `report` step. The firing node is mission 10's own, not a fixture.

```
EN: {"message":15,"tick":7,"shipped":true,"shown":true,
     "parts":["\r\nWhat a weird way to begin our mission... I don't see my companions
               anywhere. ..."]}
RU: {"message":15,"tick":7,"shipped":true,"shown":true,
     "parts":["\r\nЭто задание начинается очень странно. Я не вижу своих спутников. ..."]}
```

The number, the tick, the resolution against `main\text\battle\m10\event15.txt` and the part text
all come from the install through vfs. A scenario file can assert the numbers raised
(`HeadlessWorldAssertion.Announced`), and the assertion states the whole list rather than a subset.

**The windowed path was not driven.** The machine is the owner's desktop and was in use, so no
window was opened and no synthetic input was sent. What that leaves unwitnessed by observation is
narrow: this story changes which part of an event text is selected and whether the opcode fires at
all, and it does not change how the panel is drawn, wrapped, portrait-fitted or dismissed — that
is 0066 and 0079 code, unmodified here. Panel opening, part selection, paging and the
no-surviving-tag case are witnessed by `pkg/game` tests; the end-to-end resolution from a real
map's node to install text is witnessed by the headless drive above. `cmd/paneldump` was
considered as a no-window renderer and does not apply: it dumps unit-information panels, not the
event panel.

## The script-gap census

`bash scripts/campaign-sweep.sh` over the EN root, 2000 ticks, 28 campaign maps. The before column
was measured in this worktree by reverting the one line that adds opcode 2 to
`scriptInstantSupported` and running the sweep again, so the two rows differ by that line alone.

```
                 unsupported   reached
before                   313      7396
after                     59      7375
```

The fall is **254**, exactly the instant-2 node count `TRIG-MSGCORPUS-049` measures over the same
28 maps. The two figures are independent: the claim walks the containers with `tools/msgtext`, and
this column counts what a drive stepped on.

`pipeline/milestone-baseline.txt` recorded `cannot run 13 x instant op 2` for mission 10 and
`cannot run 11 x instant op 2` for mission 20, on both roots. Driven now with
`missionrun -mission <m> -trace -ticks 1`, both print zero `UNSUPPORTED` lines. The baseline file
was not edited.

The 59 that remain are check opcodes 4, 9, 16, 17 and 21 and unresolved ops 6 and 7. Zero instant
opcodes remain on any of the 28 maps. No part of this story implements a check opcode.

### The digests did not move

All 28 `hash` values are identical before and after. Mission 10 reads `7577197458402901582` in
both sweeps; mission 151 reads `8641609061282433150` in both. This is the simulation aspect
measured over the shipped campaign rather than over a fixture: 254 nodes that previously did not
run now run, and no map's digest changed. `reached` falls by 21 because a node that runs is no
longer counted as reached-and-unsupported.

## Shipped cases

Measured with `cmd/restool list` over both preserved roots.

- **Message 0 on mission 20.** `text/battle/m20/event00.txt` ships on **neither** root. The two
  nodes raising 0 resolve to nothing and open no panel. `settleNotices` continues its loop after a
  raise that yields no text, so a later raise on the same step still gets its turn.
- **The 12 duplicates.** 254 nodes reduce to 242 distinct (map, number) pairs: eleven pairs
  authored more than once, one of them three times. Nothing de-duplicates them.
  `TestTwoNodesRaisingOneNumberProduceTwoAnnouncements`.
- **The two unraised files.** `m50/event11.txt` and `m61/event06.txt` ship on both roots and no
  node names either. They are never opened and nothing was added for them.
- **The three RU-only files.** `m100/event09.txt`, `m130/event07.txt` and `m150/event10.txt` ship
  on RU and not on EN. On EN they take the silence arm; on RU they resolve. The same code path
  serves both and no root-specific branch exists.
- **The 19 unresolved EN pairs** are the authored silence `DLG-ABSENT-003` describes, about one
  raise in thirteen, and are not a defect to report.

## Research reconciliation

- `TRIG-MSGCORPUS-049`'s 254 nodes per root is reproduced independently by the census fall of 254
  measured here with a different instrument. The claim stands.
- `TRIG-MSGCORPUS-049`'s three RU-only files and two unraised files were re-listed against both
  preserved roots during this story and both figures reproduce exactly.
- `DLG-MSGNUM-025`'s reserved 255 is implemented as the claim states. No shipped node raises it,
  so this arm is authored on an established absence rather than observed in play.
- `DLG-TAGARM-027`'s speaker gate is implemented as a **key** lookup. The `npc.reg` measurement in
  `contract.md` shows no shipped section carries such a key, so the four speaker arms are inert
  over shipped data. Whether the original's accessor would also answer to a `Flags` token is not
  established and is not claimed. This is the one place where a claim's reading and a plausible
  alternative reading disagree over shipped data, and it disagrees for exactly four sections.
- No claim was refuted by this story.

## Contract corrections made during the story

`AC-1` previously claimed the sweep's `unsupported` and `reached` columns reach zero on every
mission. They do not: the column also counts unimplemented check opcodes, of which 59 remain and
none is in scope here. The criterion was rewritten to scope to instants.

## Allocations returned

`formatVersion` 52 was allocated to this story and is **not used**. The byte form did not change
and `pkg/sim/binary.go` is untouched. 52 returns unallocated.
