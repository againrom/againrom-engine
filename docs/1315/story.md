# Quicksave and Quickload

## Intent and authority

The owner requests three ordinary SAV quick slots. F4 saves the current state;
F9 loads the newest complete owned slot. These are working keys; the optional
preference remains unanswered. Pinned k149 AI-KEY-125 establishes F4 and F9 as
original default-map no-ops, High for the mapping. DIV-2173 records the addition.

## Behaviour

Focused campaign maps and eligible town surfaces admit one F4/F9 press edge.
Children, text, notices, dialogue, menus and cutscenes retain precedence. A
quick action consumes its frame and pending pointer gesture. F4/F9 replace the
engine's former fog-reveal and lighting-step diagnostic bindings.

F4 uses Snapshot and the single current-state player SAV producer. Three
`quick-save-N.sav` files have separate `.quick-owner` companions containing the
kind, slot, nonzero sequence and exact SAV hash. A valid document and matching
companion establish ownership. Both files missing means a free slot. Partial,
foreign, corrupt and nonregular pairs remain protected. Selection fills the
first free slot, then replaces the smallest valid sequence. F9 takes the largest
valid sequence and skips protected pairs. Ties use the lowest slot index in both
directions. Sequence exhaustion and all-protected writes fail visibly and
boundedly. An explicit subsequent F4 retries from a fresh disk scan.

Quick publication reuses the named SAV transaction, directory/profile fences,
identity and hash rechecks, staging and rollback. Ordinary chooser Delete removes
both files only for a verified pair and rechecks them after confirmation. Manual
SAVE changes the active quick destination only after successful publication.
Timed, mission-start and other files retain their independent names.

F9 restores the exact captured validated bytes through RestoreOriginal. It
shares the ordinary App LOAD transaction: replacement preparation and the map
opener succeed before the old map is released. Success resets the timed deadline.
An F9 frame suppresses the deferred timed poll, so failed LOAD leaves an already
due deadline and session unchanged. The next ordinary eligible frame can poll it.
Slot bookkeeping and wall time do not enter hashed simulation state.

## Proof

On version commit `0b3b13b1`, TestQuickKeysReachPhysicalVocabulary fails because
App rejects both F4 and F9. The RED is retained under
`review/story1315-quick-save/red-key-vocabulary.log`.

Focused controls cover `[10,1,100]` oldest selection, ties, free-first selection,
sequence exhaustion, seven-write cycling around protected slots, all-blocked
failure, new/overwrite rollback, verified pair deletion, foreign companion and
confirmation changes, captured bytes, input precedence and due-timer failure.
The installed witness uses actual App F4/F9, five distinct independently sampled
live states at each of 640 and 1024 widths, ordinary fresh-process chooser LOAD,
the same subsequent action, cold F9 ordering and town gold/next pointer action.
Private receipts and exact counts belong in the artifact manifest. The seat owns
the sole pushed-candidate review and canonical full gates on the merge.

On reviewed candidate `41c54bd1`, the durable pointer regressions fail for F4
and refused F9; the independent no-quick control passes. Quick ownership now
cancels the pending Viewer inventory, doll, map, minimap and pane gesture.
Existing suppression flags consume held button levels and their releases.
No item request, world tick, selection or saved application state changes in
the quick frame. Release, the following ordinary frame and fresh pack/map
gestures are checked. Completed item requests retain their existing drain.

Correction focused checks pass 14 top-level tests and 63 subtests with no
skips/failures. Ordinary UI passes 1575 top-level tests and 1399 subtests, with
nine asset-free top-level skips. Architecture, story, population and divergence
guards pass another 47 top-level tests and 115 subtests. One private
release-shards invocation runs quick, timed and function-key families on EN
and RU: each has three top-level passes and nine subtest passes, no skips or
failures, in 43 seconds. Each quick family includes thirteen fresh-process
proofs. The town states distinguish newest selection and +77 current gold;
cold LOAD plus the same pointer unequip retain the source. All 1455 game Go
files remain byte-identical to `41c54bd1`; its ordinary game receipt is retained.
Assets and formatting pass. FrontEnd remains 82 fields and 17 coordinators;
flow remains 80 fields and CommentBytes remains 8568910. Exact source hashes,
RED and correction receipts are under
`review/story1315-quick-save/correction-01/`. Focused evidence is not a final
release verdict.

## Unknowns

Unprocessed paused ordinary commands remain inherited producer debt for the
following active-pause result. No simulation tick is executed to hide that debt.
Quick saves inherit the current producer's other represented-state limits.
Original executable listing/LOAD remains unproved. Process termination between
paired publication or deletion remains Unknown; incomplete pairs stay protected
and healthy slots continue. Whole-slot-set and interprocess total ordering are
not proved. Adversarial directory replacement during rollback remains unproved.
