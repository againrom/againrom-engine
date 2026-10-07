# Mission SAV with current application settings

## Intent

A mission SAVE is not refused because its camera lies between cells, its speed
does not exactly match an original index, or its local panels and settings have
no original application record. This is SAV-ENDGAME M4, following its Rules for
every story on this track.

## As built

The application writer rounds the current camera to the nearest integer cell,
with halves away from zero. It chooses the original speed whose period is
nearest to the current period, with ties choosing the lower index. Unchanged
original values keep their wider signed domains. Zoom, the character,
equipment and minimap panels, and unpaced mode have no SAV leaf; they no longer
refuse SAVE. The projection leaves the live application unchanged.

A local-only checkpoint supplies its three recorded presentation booleans and
current formation and retreat mode. Its separately recorded camera wins over
the document. Other unrecorded application fields retain the document's values.
During live capture, an absent or local-only checkpoint with an admitted document records
the full current viewer and clock before projection. This includes selection
and speed changed after loading a legacy AGS without an application checkpoint,
without requiring an options toggle. An empty current selection also wins over
the document's old selection. Direct historical projection uses the document
only for fields the checkpoint never recorded.
Malformed state and missing exact bindings remain errors. There is no new
refusal or AGS selection. DIV-1184 records these encoding choices and the
boundary of the published application evidence.

The corpus comparator now observes the current mission purse and holdings.
Snapshot.Gold is the town purse and Snapshot.Party is the entry roster needed
to reconstruct entity IDs; comparing those as live holdings falsely reported
a loss on the first mission AGS this change exports. An independent World
observation found the same purse and all five current member records on both
sides. A synthetic pickup control proves the observer detects changed purse
and pack without changing the World or entry roster. Town comparisons retain
their existing source and no disclosure ceiling is raised.

## Proof

Focused EN and RU witnesses pass. An actual pointer drag leaves a fractional
camera, the game-options menu selects 250 ms, and panel changes precede ordinary
SAVE. The output is SAV, its camera and speed are projected as specified, and
a child process loads it, changes a setting and writes the next SAV. Explicit
AGS still preserves the fractional camera and complete application state.
The sole adversarial review returned the legacy LOAD followed by an options
toggle: capture still discarded its new speed and selection. The one correction
promotes that live capture. The same EN/RU witness runs original and legacy
input routes, including a legacy route without an options toggle, each with
child-process SAV LOAD and the next SAVE. A final-gate follow-up updates the
structure witness to compare SAV semantics across reminted address keys
instead of a native World hash.

The discovered 114-file AGS corpus exports 16 files: the previous 15 towns and
one mission. The comparison reports one exact round trip, 15 files with their
existing disclosures, 98 refusals and zero mismatches on both installs.
Fifteen local-only files reach the existing source-free-world refusal instead;
the remaining refusal counts are 94 source-free worlds and four Group graphs.
The original corpus remains 94 exact round trips, eight refusals and no
disclosures. The owner's mission20 first input exports; the second reaches
the existing typed-engagement-target refusal, not a speed refusal.

Final full Go, EN/RU release, milestone-2, asset and divergence gates and the
landed-tip executable census follow the sole adversarial review.

## Open debt

Engine-only panel and timing choices have no original wire representation;
camera and speed projections are owner-directed approximations, not
original-game behaviour claims. M6 and M7 still own their current-world and
source-free construction refusals. Original-game acceptance belongs to M10.
