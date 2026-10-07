# Enemy selection and Retreat continuation

## Intent and authority

Reacquisition follows on-map traversal rather than entity identity. Retreat
policy can write a pending flee during an older action. A current SAV must
restore those decisions and the next production action through cold LOAD.

Public knowledge `57ad4d61a8620f48ff86d69e72f6a2daf8627a16` is the authority.
`AI-360` establishes running-nearest admission before diplomacy and signed-HP
partition. `AI-361` establishes circular-byte turn cost alone, last-equal ties,
all 65,536 facing-byte pairs and 81 controlled centre directions, including
coincident output224. `AI-362` establishes selected unlink/tail append (High),
with session composition Medium. `AI-363` establishes selected LOAD
player/group/member reconstruction (Medium). `AI-364` distinguishes policy,
entered progress, completion-clear and entered-zero dispatch. `AI-365`
establishes the supplied active-failure tail that preserves physical phase.
Engine code and tests are not ROM1 evidence.

## As built

`pkg/sim` admits traversal candidates through the running minimum before
diplomacy and HP filtering. Earlier admitted farther candidates remain. Turn
cost chooses the last equal survivor. Self, reach, previous-target, mage,
visibility and living/corpse safeguards retain their existing boundaries.
Whole-cell direction matches the published 81-case population.

Removal unlinks traversal; return and actor creation append. Entity storage
remains sorted by identity. Selected original LOAD rebuilds traversal from
the retained player/group/member graph. Unmatched actors follow by identity.
Current SAV preserves live traversal explicitly rather than inventing a native
global-list field. Old binary input defaults to on-map identity order; an older
SAV supplement without traversal uses the selected LOAD rebuild.
Identity remapping moves traversal endpoints with actors. Admission preserves
survivors and appends new or returned actors. Typed current Actions reconcile
traversal membership with edited presence rows, retaining survivor order and
appending newly on-map actors by identity. Native binary and direct traversal
restoration still reject missing, duplicate, absent or off-map endpoints.

Retreat carries pending coordinates, progress, counter, completion and supplied
failure separately from physical attack phase. Policy can write while progress
runs. Completion-clear skips pending dispatch in that executor invocation. A
later active entered-zero invocation can dispatch despite a retained phase.
Inactive entered-zero returns before consuming typed or current-mover failure.
An invocation entered with nonzero progress can still reach that failure tail
after completion-clear. Logical progress-clear retains the physical victim,
phase and countdown until a later admitted dispatch retires that carrier.
The held carrier does not approach or strike between those invocations.
The supplied active-failure tail can install a new target and progress1/counter0
without resetting phase. A stale imported mover cannot supply that failure.
Existing simulation cadence remains unchanged.

`pkg/game` projects known current state/pending/progress/completion into existing
SAV actor fields. Loaded cast operands and action survive that projection.
The live loaded attack carrier retains its pending byte while its cycle runs;
the SAV producer projects current Retreat pending/progress separately.
Completion-clear writes progress0 with the retained physical action/phase and
victim. The existing binary and global attack validators remain unchanged.
The existing typed SAV continuation carries exact engine traversal and executor
state. Optional World byte form101 hashes these fields; old forms omit them.
SAV remains the sole producer. No original SAV field or byte identity is added.
The gated-test manifest adds one installed witness. Ledger rows close the former
selector and pursuit heuristic debts and name four bounded Unknowns.
Malformed tactical sections use the existing prefix-free decoder error route.
`pkg/ui` retains the save name and the load window's width/cut assertions.

## Proof

Six control groups cover admission permutations; score, ties and the published
direction population; unlink/reappend and selected LOAD order; policy versus
progress-clear/entered-zero activity; active/stale failure with retained phase;
and current SAV decoded state, bytes, cast operands and next-action loss.
Twenty new asset-free tests cover `pkg/sim` and `pkg/game`. Focused correction
checks also preserve actor identity, detached admission, cast/effect LOAD,
loaded attack ownership and reader/field instruments. `internal/archtest`
pins pending dispatch as the new destination originator. RED discriminates
the former admission, distance scorer and busy-policy behaviour. Owner artifacts
contain `focused-red.txt`, `focused-green.txt` and the three
`focused-correction-*.txt` receipts.
`focused-tactical-load-error.txt` covers four corrupt tactical inputs and the
unchanged full version-message and truncation controls; valid tactical bytes
still decode through the production upgrade route.

`TestReleaseRetreatPendingSAVColdLoadAndNextAction` opens installed mission41,
admits Retreat through App input, supplies a bounded loaded-cycle control,
writes SAV through App F2, returns to GAME, cold-loads it through the main menu
and compares three production session ticks. A second App F2 SAV is cut at
completion-clear, returns to GAME and cold-loads through the main menu before
the next two production ticks. Removing
only tactical continuation fields changes SAV bytes, decoded state and the
first attack/move action. EN/RU validation belongs to the release gate; focused
tests do not claim install or release readiness. Existing refused-flee, loaded
shot/cast, crossing, Hold and original-admission controls remain required.
Independent activity and completion-clear controls cover typed/current-mover
failure, active entered-zero and inactive entered-nonzero, retained phases1/4,
strict World decode and current SAV bytes/state/first-action loss at that cut.

## Open debt

`DIV-1631` names direction inputs outside the 81 controlled centres.
`DIV-1632` names initial chronology, external list writers and unmatched LOAD
ordering. `DIV-1633` names original raw state0x16 admission, controller/owner
mapping, counter construction and application-side failure prerequisites.
`TestSavedTactical1115RetreatDispatchAndOriginalUnknown` preserves that explicit
unsupported original-admission instrument and state22 continuation issue. `DIV-1634` names unestablished
visibility/terminal-target safety-filter placement. Native cadence and
whole-session cancellation remain `DIV-576`; generic refused orders remain
`DIV-1520`, and the seed-only route inference remains `DIV-1556`.
