# Story 1293: Command and selection voices

## Intent

Known defect B12. A unit's spoken reply to a map gesture follows the original's
speaker chooser and voice readers instead of the authored hero priority,
first-eligible speaker and three-recording cycle. Rows DIV-1293 (command voices)
and DIV-1457 (selection voices) are narrowed; DIV-1489 is untouched, because
the wounds still use the same one stamp.

## Authority

Pin k128. `VIDEO-067`, `VIDEO-068`, `VIDEO-069`, `VIDEO-070`, `ANIM-119`,
`ANIM-120`, plus `ANIM-094`, `ANIM-096`, `HERO-APPEAR-055` for the bank. B1
holds: the questions below carry no expected answer.

## As-built behaviour

- `pkg/ui` asks one command reply per gesture, after the order, with a
  `VoiceGesture` and the units the gesture reached: move, attack, swarm, patrol,
  town, guard, stand ground, defend, retreat and pickup. A cast asks none.
  Guard, stand ground and retreat reach the reply from the stance and retreat
  routes; before this story they spoke nothing.
- The selection reply is asked only by a plain click and a plain marquee
  (`noteSelected` runs with Shift up). The E key and group recall no longer ask.
- `pkg/game/commandvoice.go`: `speakerOf` takes the first non-empty tier
  (hero-shaped, armed human, unarmed human) of living members with a voice bank
  and picks index `draw*n/32767`; a draw of 32767 picks nobody (DIV-2040).
  `commandRecording` draws `rand>>13` for the command reader (`command1..3`,
  `defend` for a non-peasant bank) and uses `defend`, `retreat` or `idle` with
  no draw for the fixed readers. The selection reader draws `rand>>14`.
  Thresholds are 3000 ms (command, defend, retreat, idle) and 2000 ms
  (selection) on the viewer's one stamp (`ClaimVoice`), shared with wounds.
  The reader's draw is taken only after the stamp admits.
- Move and swarm are silent when the speaker's last drawn frame showed the
  move action (`mapWorld.drawnMoving`, DIV-2042); the speaker is drawn first and
  no other unit is tried.
- The generator is viewer-owned (`newVoiceDraw`), so a reply never takes a
  value from the simulation's stream: the World digest is the same with
  Acknowledgement on or off (DIV-2038). Stamps are viewer state and are not
  saved.

## Rows

- DIV-1293: narrowed to five named differences and three research questions;
  stays OPEN, FIDELITY-DEBT.
- DIV-1457: caller set, Shift gate and reader are now claims; stays OPEN,
  UNKNOWN on the summary bit meanings.
- DIV-2038 generator, DIV-2039 clock, DIV-2040 one-past slot, DIV-2041 member
  set, session mode and primary object, DIV-2042 moving test. DIV-2043 is
  returned unused.

## Proof

- `pkg/game/commandvoice_test.go`: tier order and per-member tests, the index
  formula at its draws, no draw when nobody qualifies; recording and draw count
  per gesture; the 3000 and 2000 ms boundaries on the shared stamp; refusal of a
  speaker on cooldown with no second speaker and no reader draw; a missing
  recording keeps the stamp; the moving gate; the selection primary gate; the
  generator range.
- `pkg/ui/voicegesture_test.go` and `selectionvoice_test.go`: each gesture
  through the App asks one reply with its gesture; a cast asks none; E, group
  recall and every Shift form ask no selection reply.
- `TestSelectionRepliesThroughTheMapInput`: a fixture mission through the
  ordinary App; the World tick and hash equal a run with the option off.
- `TestReleaseUnitRepliesPlayInstalledRecordings` and
  `TestReleaseCommandAcknowledgments1188InstalledGestureAndCheckbox` on the EN
  and RU installs: every bank's eight reader recordings are nonzero; click,
  stamp, E, retreat and move through the map input; the command witness
  compares the World digest with Acknowledgement on and off.
- Loss controls: each rule reverted alone fails a named test (see the return).

## Open debt

- The claim's executed list places the three-member boundary at draws
  21845|21846; the index formula `draw*3/32767` places it at 21844|21845.
  Question for research: at n = 3, which member does draw 21845 select?
- The five unresolved indirect receivers, the one-past slot, the sample
  service's refusals and the summary bit meanings (DIV-1293, DIV-1457,
  DIV-2040, DIV-2041).
- E and group recall were authored selection forms before this story; they are
  silent now because `VIDEO-069` names three map-click callers only (absence of
  other callers Medium).
