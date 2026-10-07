# ROM2 dialogue and new trigger arms

## Intent and authority

The owner completed ROM2 mission 10 in the current engine: victory fires, but
none of its dialogue triggers produces a dialogue. Deliver the ordinary
dialogue path and all ROM2-only trigger arms while preserving that victory.
This report is engine observation, not native ROM2 evidence.

EXP-2008 investigates new script arms and their actual consumers. EXP-2009
investigates mission text selection and decoding. Implementation uses their
published claims after the engine pin adopts them; unread operations do not
inherit ROM1 semantics.

Public authority is k178 `fe7d269a`, with native script contracts
R2-ENGINE-041..048/053..060 and mission-text contracts049..052. The engine
worktree adopts that gitlink. NPC/activity producers 069..072 are published with their High/Medium/Unknown
boundaries. Native unknowns do not acquire a guessed classification.

## Proof

First reproduce a missing authored dialogue through the installed mission's
compiled announcement and production text reader. Then verify ordinary
script execution, visible text and dismissal on ROM2 EN/RU, including one-shot
and ordering controls, all new operation families, and existing victory.
Shared changes retain ROM1 EN/RU release and current-state SAV controls.

At base `b59b4699`, installed-root tests fail on EN and RU: mission 10 event 1
has no readable text; 300 ordinary UI steps set its first message latch
without opening a dialogue. The compiled scripts of all 46 campaign maps leave
192 new-arm nodes unsupported per root. These are regressions awaiting the
implementation, not completion evidence.

The production mission-section reader now opens mission 10's first ordinary
dialogue on EN and RU New Game. Separate red header witnesses distinguish
controlled-NPC presence from health, omit ROM1's bare speaker-class arms,
and bound the native speaker field to five characters. These pass after the
selection correction. An independent missing-briefing witness also fails on
the old reader and passes through ROM2's combined mission-text section.
DIV-2378 records the portable RU code-page choice, whose native system
configuration remains Unknown.

Actual execution reports retain ordered messages before victory. Both clocks
deliver the same transient stream. Source-bound EN/RU tests select and acknowledge
all seven mission 10 events: 15 pages on EN and 14 on RU, including event 7
with four and three parts. CPU compositions use the production notice renderer;
the map itself has no CPU frame seam. RU frame-piece fidelity remains DIV-2371.

The 46-map census has no unsupported new arms and no omitted new-arm reference
on either root. The single-hero entry omits 189 shared-arm references in later
maps, exclusively absent 10002/10003 companions; that campaign-party debt stays
visible in DIV-2355. Mission10 has no omissions. Focused synthetic execution
covers each new arm, actual effect attachment/expiry, carried-item removal,
group activity gates and explicit commands. Bank/group footer round trips and
corrupt-footer rejection preserve ROM1 byte compatibility. ROM2 player SAVE
remains unavailable. The sole review returned one failure-text slot defect:
missing reasons3/4 read decimal121/122 portrait captions instead of native
hex-base slots283/284. Installed EN/RU production failure-before-victory tests
reproduce that defect and pass after the correction to `0x118+reason`.
The seat records the corrected merge, final release gates and promotion in
its single review report and implementation journal.

## Open debt

The published authority leaves explicit producer/fidelity boundaries in
DIV-2354/2355 and DIV-2378..2382. Campaign driving, ROM2 SAV, full mission
layout and movie playback remain in the track plan. This slice does not claim
that the full ROM2 campaign is playable.
