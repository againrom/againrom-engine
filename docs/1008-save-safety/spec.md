# 1008 — save safety specification

## Terms

**Final save** is the unique `save-YYYYMMDD-HHMMSS[-N].ags` path returned to the player.

**Temporary save** is an exclusively created sibling file used only while one final save is being
published. Its contents are complete, synchronized and closed before publication starts.

**Candidate** is a decoded snapshot plus every town or mission object required to enter it. A
candidate has passed the envelope, gob, simulation version, simulation invariant, map, asset and
mission-construction checks that can fail during load.

**Commit** is the first operation allowed to replace the active front-end state. A prepared commit
has no remaining file, decode or mission-open operation. One prepared opener performs its commit
at most once across serial or concurrent calls.

## FR-1 — atomic save publication

`SaveStore.Write` keeps the current directory rule and unique final-name rule. It creates the save
directory lazily, creates a temporary file in that directory, writes all bytes, synchronizes the
file and closes it. It then publishes the temporary file through an operating-system operation
that fails when the final path already exists. The final name does not exist before publication.

Create, write, sync, close and publish are production operations behind one testable file seam. A
failure before publication commit returns an error, attempts to close and remove the temporary
path and publishes no final save. Cleanup also runs when a write reports a partial count. A close
or remove failure is joined to, and does not hide, the operation that refused the save.

Publication has an explicit committed result separate from its operation error. Once a final path
is committed, `Write` returns its name successfully. A post-commit failure to remove a private
source name is carried as cleanup state, never as a publication failure, because failure beside a
visible valid row would contradict the filesystem. Retirement removes the path from the active
registry and, under the same lock, runs one recovery pass across that physical directory. Thus the
last of several concurrent writers sees all aliases earlier writers retired, removes every one the
operating system now permits and cannot touch a writer still active.

Temporary names carry the creating process identifier. Creation and an in-process active-path
registry share one lock, so recovery cannot remove a concurrent caller's live temporary file. The
registry compares the directory through operating-system physical-file identity and the filename
with platform case semantics. Callers using relative, absolute, case, symlink, junction or volume
aliases therefore name the same active path where the operating system says they name the same
directory. Symlink evaluation supplies a stable publication spelling when permitted; physical
identity remains authoritative when Windows denies that metadata query.
Before creating a new temporary file, a writer retries removal of closed residue owned by the same
process. A repeated refusal returns before creation and cannot add another staging file or reach
publication. Multiple writers already active when post-commit removals begin can each leave one
alias only while the operating system continuously refuses both retirement recovery passes. The
last retirement retries the whole inactive set; a later writer retries it again before creation.
Another process's namespace is never recovered because that path may still be live. If the creating
process exits first, hidden residue may remain on disk. It is not an `.ags` row and cannot replace a
final save. This contract does not claim that an operating system which continuously refuses
deletion nevertheless deleted the aliases.

Earlier `.ags` files are not opened for writing and are not removed. The writer does not use
`Stat` to reserve a name. It tries the base name and then numbered suffixes through the exclusive
publish operation. A collision leaves the complete temporary file in place and tries the next
suffix. This rule holds across threads and processes. Read still accepts only a bare filename, so
publication does not weaken traversal safety.

## FR-2 — durable contents

The temporary file is synchronized before close. On Windows, publication calls `MoveFileExW`
without `MOVEFILE_REPLACE_EXISTING` and with `MOVEFILE_WRITE_THROUGH`. Omitting the replacement
flag makes destination creation exclusive. The write-through flag asks Windows not to return until
the same-volume move has completed. The contract does not claim survival from storage hardware
that acknowledges a flush and later loses it.

On non-Windows systems, publication creates the final path as an exclusive hard link to the closed
sibling file. That exclusive link is the commit point. Removal of the temporary name is
post-commit cleanup: a refusal leaves a hidden hard-link alias for next-save recovery, returns a
distinct committed-with-cleanup result internally and does not remove the final name. The content
was synchronized before link creation. This implementation does not synchronize the containing
directory and therefore does not claim that the final filename survives a sudden power loss beyond
the operating system's hard-link guarantees.

## FR-3 — prepare before commit

The UI load seam asks the game tier to prepare a candidate while the load window and its backing
screen remain active. For an `.ags` mission save, preparation performs all of the following before
the seam reports success:

1. read and decode the envelope and gob payload;
2. restore candidate campaign, party and offer values without writing the active front end;
3. resolve and read the mission map from the current install;
4. decode map and presentation assets needed by the mission screen;
5. construct the mission and unmarshal the saved world through simulation form 53;
6. clone the mutable unit-body cache, load candidate-only bodies into the clone, apply residue and
   build the complete viewer and runtime seams against that clone;
7. retain one commit operation that adopts the prepared campaign, unit-body cache and live driver.

A town candidate completes steps 1 and 2 and prepares its town state. Commit resets the town UI to
the square and regenerates non-serialized stock through the existing arrival rule.

The original `.sav` path may use the same candidate/commit structure, but this story does not widen
what that format restores.

## FR-4 — failed load is non-destructive

A preparation error returns its exact refusal and does not call the commit operation. It leaves all
of these values unchanged:

- the UI screen, backing screen, selection and load row;
- the current viewer, command mode and all map seams;
- the shared unit set and every cached body entry used by the running map;
- `FrontEnd.Town`, `Carried`, `Offered`, live mission number, live party and live driver;
- the running world's bytes and hash.

The load window remains open and displays the refusal. Its row remains choosable.

After successful preparation, the UI may release the old map and then invokes the returned commit
opener. The commit returns already prepared viewer and runtime seams and cannot discover another
decode or open failure. A concurrency-safe once gate makes simultaneous invocations wait for and
share the same commit. A town restore has no opener: its complete candidate is committed before the
successful return, after which the UI releases the old map and shows the town at the square.

## FR-5 — released-form compatibility baseline

A synthetic, committed byte string represents one valid mission `.ags` file written with envelope
version 1 and simulation form 53. The fixture contains no game asset. A test decodes it through the
production envelope and simulation readers and proves the expected mission, town fields and world
hash.

Envelope version 1 has one exact length. The bytes after the label contain an eight-byte checksum
and length prefix followed by exactly the declared payload. A shorter body is truncated. A longer
body has trailing data. Both are refused before gob decoding and produce no partial snapshot. The
declared and checksummed payload contains exactly one complete gob snapshot. A second gob value or
any unread byte inside that boundary is also trailing data and produces no partial snapshot.

The complete envelope, including header, label, checksum, length and payload, has an authored safe
maximum of 16 MiB. Encoding and direct store writes refuse larger values before publication. The
decoder rejects both a larger byte slice and a declared size which would exceed the ceiling before
gob decoding. The load list reads only the bounded prefix and shows a row only when the file's
physical size exactly equals its safe declared envelope size; oversized, truncated, trailing and
future-version files are absent.

The byte ceiling does not trust gob's internal container counts. Before `encoding/gob` constructs
the destination, a non-allocating wire preflight parses the public delimited-message grammar,
records at most 1,024 type definitions, limits nesting to 128 and charges every wire-field, array,
slice and map count against one aggregate maximum of 65,536. Raw byte and string values remain
bounded by the already-held 16 MiB payload. The preflight refuses an over-budget count before
iterating its claimed entries or allocating the destination collection. The encoder runs the same
preflight, so this build never writes an envelope it would reject on that policy.

`SaveStore.Read` opens one handle, checks that handle's physical size and then reads through a
16-MiB-plus-one-byte limiter. It never allocates from the envelope's declared length. The stream
limit remains authoritative if the file grows after the size check. A load refusal after a row was
already listed takes the same non-destructive path as every other candidate refusal.

The current encoder must reproduce the exact fixture from the same synthetic snapshot. This pins
the released form and makes an accidental gob or envelope change visible. The test names the policy:
this build reads exactly simulation form 53. Changing the form requires an explicit compatibility
decision and fixture update; the fixture does not make older forms readable.

Preflight does not rewrite the gob or introduce an envelope version. The committed
envelope-1/form-53 fixture and ordinary current saves pass through unchanged before the standard
decoder remains the value-compatibility authority. The safety policy deliberately refuses an
otherwise small customized envelope-1 save if its aggregate container/type-field population
exceeds 65,536.

## FR-6 — disclosures

The divergence ledger states these outcomes in player terms:

- an `.ags` from another simulation form is refused; there is no migration chain;
- a complete `.ags` above 16 MiB or above 65,536 aggregate gob container/type-field entries is
  refused and this build has no streaming large-save reader;
- `.ags` campaign definitions and presentation assets come from the currently selected install,
  not from the save, and no content fingerprint enforces a match;
- an original between-mission `.sav` restores the supported party and purse but starts with fresh
  completed missions, available missions and consumed offers;
- `DIV-026` remains the single row for original-save state that is decoded or present but not
  applied;
- the original menu and againrom menu differ where typed naming, overwrite, delete and autosave are
  absent.

The pre-story direct write and destructive load ordering are recorded as closed defects in their
rows after the fixes land.

## Acceptance criteria

- **AC-1** Injected pre-commit failures at create, write, sync, close and publish each return their
  own error, leave no final `.ags`, attempt cleanup and preserve a pre-existing save byte for byte.
  An injected cleanup-close or removal refusal is returned together with the primary failure. A
  successful removal leaves no temporary file. An injected non-Windows unlink refusal after the
  exclusive link returns committed success, leaves the final row readable, never attempts to
  unlink that final row and makes the private name eligible for next-save recovery.
- **AC-2** A normal write appears under one unique final name only after the production sequence
  write, sync, close, publish. The file decodes and same-second concurrent processes keep every
  save under a distinct name.
- **AC-3** A stale simulation version accepted by the envelope but refused by `sim.UnmarshalBinary`
  is chosen through the real UI load path over a running mission. Screen, viewer, map seams,
  front-end campaign state and live-world hash remain exact.
- **AC-4** Independent mission-map read, map decode/world invariant and late mission-construction
  failures use the same path and leave the same state exact. A candidate carrying an uncached body
  and a corrupt world does not add that body to the running map's unit set.
- **AC-5** A valid mission load is fully prepared, commits once, releases the old map and enters the
  saved tick. A valid town load commits once and enters the town square.
- **AC-6** Mutating the load path back to `leaveMap` before the failing opener makes the
  non-destructive-load witness fail.
- **AC-7** The committed synthetic envelope-1/form-53 fixture decodes to its literal expected state,
  and the current encoder reproduces the committed bytes exactly. A byte outside the declared
  envelope, a second gob value inside a rechecksummed envelope and one unread byte inside such an
  envelope are each refused as trailing data.
- **AC-8** Default save directory selection, `-saves` override, bare-name refusal, original-save
  listing and current UI labels remain covered by their existing tests.
- **AC-9** Headless production loads through the UI/game seams on both EN and RU installs. A refused
  candidate retains the running mission; a valid candidate resumes it.
- **AC-10** A pre-commit removal refusal leaves one hidden process-owned staging file and returns
  the save failure with its cleanup failure. A post-commit non-Windows private-name refusal leaves
  the same hidden shape but returns the committed final name successfully. Two concurrent commits
  whose private unlinks both fail run serialized retirement recovery; after one injected recovery
  refusal, the last writer reaps both inactive aliases. A continuously denied set remains hidden
  and makes the next save refuse before creation. Concurrent callers and processes do not remove
  one another's live staging files.
- **AC-11** A relative-path writer paused after close and an absolute-path writer targeting the
  same physical directory both publish successfully. Recovery recognises the paused staging file
  as active. Foreign-process namespaces remain untouched.
- **AC-12** A sparse 64 MiB file, a declared envelope above the safe maximum, a trailing file and a
  future-version file are absent from the list. The sparse file is refused from its same-handle
  size without reading its contents, and a stream that grows after that check stops at the bounded
  extra byte. A valid-length, valid-CRC payload claiming a 1,048,576-entry residue map is refused
  before allocating more than 4 MiB, while removing preflight makes the same focused test allocate
  over 37 MiB. The released fixture and ordinary saves remain readable, and the encoder refuses to
  produce an over-budget collection.
- **AC-13** Two simultaneous calls to one prepared opener execute the front-end commit exactly once
  and both receive the prepared seams.

## Non-goals

No streaming reader or writer above 16 MiB and no container population above the authored 65,536
aggregate ceiling. No simulation-form migration. No original-world completion. No original
difficulty, omitted spellbook/session materialization or original-save writer. No content
fingerprint enforcement. No new save-name, overwrite, delete or autosave UI. No simulation
byte-form change.
