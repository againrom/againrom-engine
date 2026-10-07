# Script command17 reaches Group Roam

Script sub-command17 now executes on native and imported Groups. Compilation
reports it as supported, the fired instant reaches its setter, and native
SAVE/LOAD preserves subsequent Group evaluation. Catalogue18 stays inert.
AI-GROUPCMD-020 and AI-ROAM-025 count zero command17 nodes in the measured
shipped maps; this closes an authoring gap rather than a campaign blocker.

The setter runs the STOP projection, writes Group order0x11 and seeds the
Group cell from its first member. AI-PATROL-018 separates that helper from
Patrol's later two-node-ring constructor: command17 retains the prior actor
state and ring. Imported orders also receive the literal helper writes to
range, order38/50/60, mover7c and actor action. Native target, route, attack
cycle and group speed clear. Roam takes ownership of group evaluation, so a
retained native Patrol or Escort cannot also decide. A new player order
creates its own Group and takes priority.

AI-ROAM-025's maximum member distance below10 or counter above50 triggers
an eight-direction20-cell reroll inside the inset rectangle. Acceptance
updates the Group cell and resets the counter; ordinary Swarm2 evaluation
then runs and the counter increments. AI-SWARM2GATE-107 and AI-MOVE-023 say
the Move fallback reads the actor's existing destination. This slice never
copies the generated Group cell into a member destination and does not
claim guaranteed wandering movement.

The existing saved SplitMix64 stream remains the RNG policy. Every rejected
direction consumes a draw, up to32 attempts; after32 rejections the first
valid direction is used. An impossible rectangle is refused before any
primary mutation or draw. DIV-789 records both differences from the original
unbounded rejection loop. DIV-1022 records native membership/STOP projection,
decision cadence and unproved composed helper chronology. DIV-386 closes
only the absent command authoring route.

Form87 appends sparse native Group counters, keyed by the exact owner/group
pair. Nonzero counters survive replacement orders; absent old fields default
to0. Imported counters already live in their retained AI bytes. The reader
rejects out-of-buffer spans, zero/repeated/descending records and missing
Groups before publishing state. Existing form86 inputs remain readable; an
older executable cannot read form87. Historical literal pins change only
their version tag and four zero footer bytes, with form86 hashes preserved.

## Proof

Focused tests exercise the compiled one-shot trigger and both Group stores,
current-list versus EntityID first-member selection, two native owners with
the same selector, all STOP field writes, preserved actor/ring state, player
replacement, missing/empty/ambiguous/unmaterialized selectors, exact10 and
50/51 thresholds, counter255, impossible rectangles and a literal32-reject
RNG seed. Cold native reload is compared before each of130 successor ticks.
The new footer has an independent literal layout and atomic corruption tests.
Historical inputs and older format digests remain frozen.

The sole review returned one native membership defect. The correction admits
Roam through the same moving-member gate as Swarm2. A compiled command followed
by automatic withdrawal keeps evaluating through a cold reload; a mixed Group
retains its moving member in the maximum-distance calculation.

The candidate runs full Go, gofmt and the asset guard before its sole review.
The landing runs paired EN/RU release, original-save acceptance and the
milestone script census. An ordinary executable witness must identify its
synthetically authored command17; no shipped node is claimed as its source.
