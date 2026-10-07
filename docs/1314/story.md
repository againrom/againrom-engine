# Timed current-state autosave

## Intent and authority

Game Options controls wall-time autosave independently of simulation speed.
The seat chose reversible working defaults: enabled, five minutes, three timed
slots, and modal deferral. The optional owner preference remains unanswered.
DIV-2171 records this addition; it asserts no ROM1 timed-save policy.

## Behaviour

Game Options edits enabled and positive minutes in its local draft. OK persists
both preferences together; Cancel and Escape discard them. Missing or malformed
old keys use the working defaults. Left and Right adjust the focused minute
control; the two pointer halves decrease and increase it. Enter adds one minute.
The maximum is the largest complete
minute interval representable by the process clock duration.

An injected process-local clock owns the deadline. Accepted map entries, LOAD
and NEW GAME reset it. Menus, notices, dialogue, movies and other modal surfaces
defer an expired deadline. The first admitted map or town frame writes current
state through Snapshot and the ordinary SAV producer. Large clock jumps produce
one write. Failure reports a visible message and retries after one minute.

Three separate `timed-autosave-N.sav` files rotate by successful sequence. Each
has a `.timed-owner` companion containing kind, slot, sequence and the exact SAV
hash. Replacement requires a matching pair and a valid SAV document. Unowned,
incomplete or corrupt targets are retained. The shared named-save publisher
checks directory and file identity, stages complete bytes and restores changed
targets after a publication failure. Manual and mission-start files use their
existing independent names. A successful manual SAVE changes the active timed
destination. Clock and rotation values never enter simulation state or SAV.

Rotation scans at most three targets in sequence order. Protected targets are
skipped; two healthy slots continue alternating when the third is blocked.
If all three are protected, the visible failure and one-minute retry remain
bounded. Ordinary LOAD/SAVE Delete removes a companion only for a verified
kind/slot/sequence/hash pair with a valid SAV. Both files are rechecked after
confirmation. A companion deletion failure restores the removed SAV without
overwriting a file that appeared during the operation.

## Proof

The clock-expiry regression failed on version commit `3e6ff190`: an accepted
App mission created its start save but no timed SAV after five minutes.
Evidence: `review/story1314-timed-autosave/red-clock.log`.

The sole review returned F1: deleting a timed SAV left its companion, and a
protected next slot stopped the other two. The correction RED uses ordinary
App LOAD Delete and foreign, corrupt, orphan and mismatched slot controls on
unchanged production bytes from `503f9265`.
Evidence: `review/story1314-timed-autosave/correction-01/red-recovery.log`.
Correction EN/RU invocations each pass 23 top-level tests and 25 subtests without
skips or failures. Four installed families cover timed SAV, mission-start
routes/successor and scheduled light. Eleven fresh-process cold LOADs per
install compare independently sampled source state and the same next action.
Map samples cover tick0/16, tick16/32 and blocked-slot recovery at tick32/48.
Ordinary chooser Delete then removes the verified pair and timed writing
resumes at current tick48. SAV, World/hash, view and campaign comparisons retain
late-state and missing-write loss controls. Installed Game Options captions
and controls fit the 640 and 1024 routes.
The independent review's town control is retained: gold changes from 600 to
677 before expiry, then cold LOAD and the same pointer unequip preserve the
sampled party, gold, chapter and campaign history. Fixtures cover seven-write
cycling through slots 1/3, protected bytes, changed-file confirmation, delete
rollback and bounded all-blocked retry. Deadline/reset, modal/dialogue,
settings OK/Cancel/Escape, manual/start preservation and publication rollback
controls remain. Ordinary game and the affected story, population, divergence
and architecture guards pass. FrontEnd remains 82 fields and 17 coordinators;
no ratchet rises. All correction receipts are under
`review/story1314-timed-autosave/correction-01/`.
The seat owns canonical full Go, release and milestone gates on the correction
merge. The correction does not claim those gates are green.

## Unknowns

Original executable listing and loading of the emitted SAV remain separate,
unproved acceptance. Timed files inherit the current SAV producer's represented
state and remaining ledger debt. A process termination between publishing the SAV and its
ownership companion may leave an incomplete pair; the next write preserves it
and continues through healthy slots. If all slots are protected, it reports a
bounded failure. Crash atomicity of paired deletion is also unproved; retained
recovery files are named if rollback fails. No native window was driven.
