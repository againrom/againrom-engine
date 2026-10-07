# Story `1044` — original-save campaign progress

This is the canonical as-built specification. `contract.md` records the dispatched result and
research boundary. Base `d5b76f4c`; merged masters `75fe35f9`, `220bf4c2`, `d3049be7`,
`1f62579c` and `b4e119ef`; research pin `23daf74f`. Simulation form 61 remains current.

## 1. Typed import

**B1.** `sav.File.Campaign` exposes a detached `CampaignProjection` only when the tail contains a
valid embedded state store followed by one complete campaign record. It covers the main and child
records, both fifteen-type mercenary-count arrays, hire flags, six candidate and unlock arrays,
documents, selected/automatic/last missions, the first-MapPoint flag, mission time and marker
cache. The two unnamed top-level scalars are consumed only to frame the record and have no field in
the projection. The marker record's still-unnamed dwords remain typed values with no game meaning.

**B2.** The parser bounds every count, validates the announcement, hire and first-MapPoint
booleans and every document kind, requires one non-empty NUL-terminated marker name with no
embedded NUL and consumes the tail exactly. A malformed record returns no partial projection.
Every returned slice is copied, including nested record arrays.

## 2. Campaign replacement

**B3.** `FrontEnd.RestoreOriginal` parses and validates the whole projection against the already
loaded `scenario.reg` campaign before mutating front-end state. The main mission must be a declared
main section; children must be distinct retained side sections; selected must name the main or one
child; every inn, school and shop candidate must name one of those live records; mercenary types
must be valid; and the paired arrays must agree. A mid-mission save's active mission must be that
selected live record. A town-only save is valid only at or after `Campaign.TownBegins`. Validation
precedes every front-end mutation on both original and againrom restore paths.

A successful import constructs a restored `Town` for both lawful town-only and mid-mission saves.
Its open latch is derived from the restored main mission and `Campaign.TownBegins`: restored main
10 and 20 remain pre-town, while main 30 and later expose the town. It replaces fresh defaults; no
chapter-30 state is unioned into it. A pre-town defeat still returns to the main menu, an off-map
save remains unavailable, and completing main 20 opens the town through the ordinary campaign
boundary.

The live restored state owns the main and retained children, record-local announcement latches,
working/pristine/hired mercenary state, shelf and permanent unlocks, paired inn candidates, school
and shop candidates, documents, selected mission, MapPoint relation, mission time and markers.
Town offers, availability, payment, shop bounds, mercenary rows, documents and world-map position
read this state.

## 3. Progression

**B4.** A lower requested main mission is complete for presentation and is refused before map I/O.
No name or companion-specific rule participates.

A side win drains its `EnableMercenary` values into permanent unlocks, removes that child, leaves
the other children unchanged and selects the current main. A main win drains its unlocks, advances
by the researched ten-mission stride, ages retained children, removes age 2, reloads the next main's
record and four building arrays from `scenario.reg`, and appends the newly configured side
candidates at age 0. The announcement-derived availability set is rebuilt from the records that
survive. A final main stays selected.

**B5.** Town acceptance removes the persisted candidate before setting its record announcement
latch. Inn acceptance removes the same index from both paired arrays; school and shop acceptance
remove their one candidate. Selection updates the selected mission. `AddHero` is consumed once at
town activation. The existing NPC-22 grant remains separate from Brian's mission-40 map handover.
Documents append in text-then-picture order and deduplicate by `(value, kind)` without applying
againrom's fresh-campaign monotone document guard to a restored side mission.

## 4. World map and persistence

**B6.** The restored first-MapPoint flag chooses the first campaign point when true and the selected
mission's point when false. A non-empty imported marker cache seeds that selected mission's marker
presentation without assigning a meaning to the record's unnamed dwords. Selecting a mission after
a zero-marker import sets the same presentation latch and clears first-MapPoint. Both facts survive
an againrom round trip and reach `enterWorldMap` and `WorldMapView`; `DIV-418` records the remaining
original-record mapping uncertainty.

**B7.** An againrom snapshot carries `CampaignState`, the detached typed projection and
`CampaignMarkerSelected` as additive gob fields. The separate boolean persists againrom's
selection history without manufacturing a marker record in the original format. Restore validates
the projection, active location and town boundary before installing a candidate. Legacy snapshots
leave the fields zero and retain their existing fresh-campaign path. No original tail bytes enter
canonical state. The authored gob location is `DIV-417`. The outer fixture hash changes;
`pkg/sim` binary form, upgrade paths and hash remain form 61.

**B8.** Every original-save load row and the headless `savecheck` seam disclose that party,
positions, supported statistics and items, explored map and campaign load, while the mission world
restarts. A between-mission restore reports the restored main and selected missions when a campaign
record exists; absence of that record is reported as fresh campaign state.

## Design decisions

**DD1.** Original bytes remain authoritative in `pkg/formats/sav`; the game layer receives copied
values and owns mutation.

**DD2.** Validation builds a complete candidate and callers assign it once. Candidate arrays and
the active mission are joined to the restored record set before construction. Neither original-save
nor `.ags` restore can leave a partially replaced town.

**DD3.** `Campaign` stays the immutable install definition. `campaignProgress` is one player's
mutable position through it, so a modded install supplies its own declared missions and chapters.

**DD4.** Main reload derives new record fields and candidates from `scenario.reg`; it never copies
fresh chapter 30 or infers progress from the open map number.

**DD5.** Lower-main refusal sits at the mission opener before archive reads, so mission 40 cannot
reach Brian's handover once restored main is 50.

**DD6.** Record-local announcement and global building candidates remain separate state. Acceptance
orders the candidate mutation before the latch mutation.

**DD7.** Unknown top-level and marker dwords are parsed for framing or preservation only. They are
never named as progress, availability, identity or position.

**DD8.** The evidence command completes campaign state through the production `FinishMission` seam.
It does not manufacture a world-half outcome and does not launch the GUI.

## Bounds

This story does not import world-half script state, actors, cells, sacks, structures, terrain,
area effects, orders or casts. Concurrent Token lifecycle, cell rebind and dead-actor work is not
consumed. Non-empty original marker-record semantics remain `DIV-418`; no research id or SAV claim
id is allocated here.
