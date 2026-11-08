# One inn for both games

## Intent

Build both games' talker rosters, ordered seats, selected NPCs, seat actions and
talk starts through one builder per kind. Keep existing behavior except that
ROM2 hire-seat Talk opens that person's about dialogue. Preserve ROM1's disabled
Hire button when the purse is short and record the owner-directed difference.

## Authority

The owner requires the second game's shape with the first game expressed as
profile data. Knowledge advances separately from k435 to k437, commit
`5d3739691c07d4b32b23324242885d4c2c06536a`.

`R2-ENGINE-496` specifies `npc<ID>about` on a hire-seat Talk button and the RU
negative-selection guard. `R2-ENGINE-246` specifies its Talk target.
`TAVERN-BUTTON-020`, `TAVERN-LINES-022` and `TAVERN-CLICK-019` establish the
original ROM1 refusal route. The owner keeps the disabled button because it
shows the refusal before the click: DIV-4649, DEVIATION/ACCEPTED. DIV-3112 closes
and moves to `docs/DIVERGENCES-CLOSED.md`.

The neutral baseline also reads `TAVERN-ORDER-038`, `TAVERN-STAY-039`,
`TAVERN-ACCEPT-041`, `R2-ENGINE-967`, `R2-ENGINE-968`, `R2-ENGINE-969` and
`R2-ENGINE-973`. Existing differences from these claims stay outside this change.

## As-built behaviour

`pkg/game/inn.go` owns the five kinds. The old host names adapt inputs or project
outputs; they contain no separate roster filter, NPC resolver or talk start.

| Kind | Builder | Profile data, ROM1 / ROM2 |
|---|---|---|
| Talker roster | `innRoster`, registry overlay `innRegistry` | keepSpeakers true/false; uniqueSpeakers false/true |
| Ordered seats | `innSeats` | hideCompleted true/false; hires precede talkers in both |
| Selected NPC | `innActor` | tokenActors true/false; identityActors false/true; keyedActors false/true; hire tavernHires/bankHires |
| Seat availability and action | `innButtons`, `innAction`, `innControl` | disablePoorHire true/false; squadHire true/false; leaveScene false/true; soundBeforeTalk false/true; hireTalkKey empty/npc%dabout |
| Talk start and quest effect | `startInnTalk` | sectionText false/true; joinOnOpen true/false; applyTalk false/true; queueOffers true/false |

The shared seat carries the campaign word and a chapter index. ROM1 projections
keep their stable candidate keys. ROM2 projections keep their word order and
copied high bits. NPC resolution still delegates hired-member construction to
`hireMembers` and controlled-member construction to the existing character
producer. ROM1 token and stock matches retain their original order and fallback.

Hire-seat Talk enters the common dispatch with an explicit Talk verb. The UI
passes a room-button control separately from its default hire row; it changes
neither the row list nor page geometry. The profile supplies `npc%dabout`.
`startInnTalk` opens it without `TalkTo`, a hire transaction or a pending join.
An absent selection has no Talk action. Existing talk seats retain their row
route. ROM1 double-clicks retain the hire refusal path despite the disabled
button.

Both games' row-list `Choose` builds the seat action through `innAction`. ROM2
inn row i+1 is seat i through `innAct`. ROM1 tavern row i is talker seat i of
`innSeats` with the Talk verb. This row path is used by headless drivers and
tools. It now raises the tavern's Talk room event, as the Talk button does; a
driver without a sound bank plays nothing. A tavern page without a town lists
no candidate.

Room construction and painting, `innOffers`, `hireOffers`, `hireMembers`,
`hireDeal.run`, membership and `joinMember`, `talkCursor.read`, the pager,
`snapshotTown`/`restoreTown` and the SAV producer and reader remain the underlying
builders. No persisted field or RNG operation is added.

## Proof

`TestInnRecord` and `testdata/inn_record.txt` were committed before production
changed. Synthetic fixtures record both games at entry, after Talk, Hire,
dismissal and snapshot-town LOAD. They record seat order, each button's
availability and action, dialogue identity and text, quest/party state at open
and close, and SAV lengths and SHA-256 digests. Both join moments are exercised.
This LOAD is a town snapshot restore, not a native-runtime reload.

The 1,052-line baseline becomes 1,058 lines. Only six ROM2 hire-seat Talk blocks
change: five NPC1 blocks and the NPC2 hire exposed after the stage-30 talk.
Every SAV record and every other line is equal. A planted ROM1 seat swap fails
the record at line 3. Existing focused assertions are unchanged except
`TestSecondInnHireRouting`, whose former no-hit expectation now checks the about
dialogue and unchanged bank. The record adapts the new button-control call.

`TestTavernRowChooseIsTheSeatTalk` compares ROM1 tavern `Choose(i)` with a cell
selection and the Talk button on the same seat. Room, NPC, offer, dialogue text,
queue, party and SAV digest are equal, and both request the Talk room event.
Rows -1 and past the end do nothing. `TestTavernWithoutTownHasNoCandidates`
covers the page without a town.

`CheckInnBuilders` is a cheap AST scan with 14 planted source shapes. It guards
builder ownership, roster filtering, seat concatenation and literals, selected
NPC literals, direct dispatch, a tavern seat's talk opened through
`openOfferDialogue` or `openTownDialogue` outside `inn.go`, talk effects/cursor
starts, and decisions added to host adapters. Its named non-visit entries are the hire transaction, the
reachability instrument and the speech instrument, each limited to its own
call. It does not recognize every possible equivalent program. The existing
offer guard admits only the two new current/retained roster projections.

The focused game, town and UI suites, source guards, build and vet pass during
editing. `TestTownSurfaceInnButtonUsesItsAction` checks the UI consumer.
`TestReleaseSecondInnHireSeatAbout` checks installed text through headless
pointer input, with and without a selected hire, and compares SAV bytes before
and after the conversation. Final acceptance runs the fast gate once and the
brief's selected release tests on ROM1 EN/RU and ROM2 EN/RU, serialized. The lane
return records their exact results, skips and any sandbox refusal.

## Open debt

The baseline's fitted stock ordering omits the bucket wrap described by
`TAVERN-ORDER-038`. ROM1 keeps accepted speakers (DIV-1529), whereas
`TAVERN-ACCEPT-041` removes the pair on exit. ROM2 deduplicates NPC keys
(DIV-2635), whereas `R2-ENGINE-968` describes repeated vector elements. Missing
ROM2 sections still skip an empty modal (`R2-ENGINE-969`, DIV-2635). These are
unchanged. Native modal-return scheduling remains Unknown under
`R2-ENGINE-967`; this story makes no new original-runtime claim.

Two legacy tavern release witnesses require output in a foreign notes
directory. The lane leaves their output environment unset and reports their
skips. Push, reconciliation and the sole adversarial review belong to the seat.
DIV-4650 through DIV-4652 are unused.
