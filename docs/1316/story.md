# Active pause

## Intent and authority

The owner requests real player pause on bare Space and a zero Game Options
speed choice. Selection, camera and commands remain usable while simulation
time stops. Pinned k149 AI-KEY-125 establishes the original default key surface;
MENU-073 establishes the nine-position speed slider. DIV-2175 records the
owner-directed replacement. This story makes no claim of original active pause.

## Behaviour

Bare Space and Ctrl+Space toggle the same player stop. Alt/Shift Space do not.
I/backquote opens the pack; B/Q opens the book. Focus loss, text, child screens,
notices, dialogue and cutscenes retain precedence. A notice dismissal and Space
in one frame leave the preexisting pause intent unchanged. The persistent EN/RU
pause caption uses the existing message HUD and installed font conversion.

Game Options adds pause before nine positive shipped speeds. The local draft
changes no cadence until successful OK; Cancel drops it. Zero preserves the
exact positive rung and unpaced mode. A selected positive speed resumes at that
rung. Unchanged OK preserves rungs outside the slider and unpaced mode. Outside
an active mission, zero clamps to the first positive choice; pause is not a new
global preference. Fresh mission entry defaults to nonpaused. Existing partial
flag/profile writes are not made into a general atomic Options transaction.

Desired player pause is distinct from incidental modal stop. The existing
application supplement carries pause, exact period and unpaced mode, including
a cadence change and pause before a simulation tick. Its existing raw-speed
provenance guard also governs pause. Old supplements default to nonpaused;
old GameSpeed0 remains positive. The map clock consumes paused wall time without
advancing the World or repaying it after resume.

The ordinary producer captures the complete ordered pending command stream
without a simulation tick or resource debit. SnapshotResidue and
AgainromActions distinguish an absent legacy queue from a new empty queue.
Legacy Options replay only when the new queue is absent. Actor issuers and
actor/structure targets have typed archive bindings; actor0 is valid. Cells,
item/equipment slots, spell IDs, Group tags and Player roster slots retain their
domains. Document relocation moves only archive references. Ordered batches,
the group tag and live commanded schedule exclusions survive LOAD.

Missing endpoints keep opaque commands marked ignored. LOAD neither reserves
IDs nor creates tombstones for these queue bindings. They cannot attach to a
later actor and drain once on resume. Missing commanded bindings are ignored.
Malformed commands, incomplete bindings and excessive populations fail
boundedly. Ordinary LOAD prepares the replacement before committing it, so a
failure leaves the current session, queue and World intact. DIV-2176 records
the supplement and missing-endpoint policy. Manual, timed and quick SAVE use
the same current-state producer.

## Proof

The version commit is 9412709e. RED UI controls show bare Space operating the
old panel pair and zero retaining positive speed. The independent source queue
RED shows move, attack, move disappearing at cold LOAD. Private RED receipts
remain under review/story1316-active-pause/.

Focused tests cover exact positive rungs and unpaced resume, draft/failure/focus/
text/modal controls, persistent HUD drawing, source World preservation, first
resume and later ticks, queue removal and permutation, and loss of issuer,
target, cell, spell, item and group operands. Typed domains, actor0, remint/
reindex, missing endpoints, legacy absent/new empty queues, bounded malformed
atomic LOAD, cast/scroll payment and potion slot order have separate controls.
Synthetic continuation uses deterministic simulation steps independently of
the fixture's party recomputation. Installed proof uses the actual App driver.

The installed witness independently records source World bytes/hash, ordered
queue, commanded set and application controls before each SAVE. Actual App
selection, camera, movement, attack, cast, potion, scroll and group input queue
eight commands at paused tick0. Manual F2, timed expiry and F4 retain that state.
Fresh processes enter the ordinary chooser and compare before any tick, on the
first Space resume and after 24 later App frames. A fourth process removes only
the queue: pre-resume World/View stay exact and the first action differs.
Installed pointer controls exercise zero Cancel/OK and persisted positive
resume. A second SAVE has no pending commands. The caption is measured at 640
and 1024 widths. The manifest owns source hashes, counts and private receipts.
Scenarios 1005, 1087, 1089 and 1119 pass on both installs. The former Space panel
steps use I/B; the quick-spell scenario retains the pause restored by LOAD.
The sole review passed on caaef73f. Its merge chain exposed ten stale release
fixtures: nine resumed an already paused LOAD with unconditional Ctrl+Space;
one used a fixed SAVE index after panel input became I/B. The correction uses
actual Ctrl+Space only when player pause is absent and checks unchanged source
World bytes/hash, ordered commands, commanded set and cadence. Cold paused
processes apply it twice and preserve the independent sample. The town fixture
locates its unique mission-30 gate and SAVE followed by LOAD from authored input.
Cast charge, first-action lockstep, portraits, sheets, equipment loss and later
tick assertions remain intact. Production source is unchanged. The correction
manifest holds the exact ten-family EN/RU RED, 16-family EN/RU GREEN, source
hashes and ordinary checks. The seat owns final gates on the corrected merge.

## Unknowns

Original executable listing or LOAD of engine supplements remains unproved.
The current producer retains its other documented representation limits.
Interrupted paired autosave publication remains inherited Unknown. An ignored
positive-speed profile write error and partial Options flag persistence remain
inherited limits. Focused and installed receipts are not final release readiness.
