# Verification — the owner a script hands over

Toolchain `go 1.26.1`, Windows, `go test -trimpath`. Two commits, one per task,
each trailered; this file and its commit carry none.

## The gate

Run in the orchestrator's own worktree at the tip of the branch, and read by
**exit code** rather than by output:

```
build=0
vet=0
gofmt=0 (and printed nothing)
test=0
assets=0
budget=0
audit=1
```

`check-sdd-audit.sh` is the one failure and it is **inherited, not this story's**.
Its only FAIL line names a commit this branch did not make:

```
FAIL 86fd668: Co-Authored-By trailer
FAIL 0071-unit-owner: every task in tasks.md has landed and there is no verification.md
```

`86fd668` is master's own tip — the research pin bump this branch was cut from —
and it carries the `Co-Authored-By` trailer `AGENTS.md` forbids. It fails on any
branch built on that commit and cannot be corrected from here without rewriting
master's history. The second line is this file's absence and is closed by this
file. With both accounted for the story's own audit rows are clean.

The deletion check, which a green gate cannot make:

```
$ git diff --diff-filter=D --name-only 86fd668 HEAD
$
```

Empty. No file left the tree.

## SC-1 — the owner reaches the entity (FR-1, FR-2, AC-1, AC-2, DD-1)

```
--- PASS: TestUnitOwnerSlotIsDecodedAtItsOwnWidth
--- PASS: TestUnitOwnerSlotCostsTheDocumentNothing
--- PASS: TestEveryEntityCarriesItsRecordsOwner
--- PASS: TestAWorldBuiltFromNoMapCarriesNoOwner
```

The decode is asked with owner words `1, 9, 0, 0xffffffff` — the first roster
slot, the highest any shipped map authors, the value that names none, and the
top of the field, which no narrower read could carry. The words are written
straight into the record at `+0x14` rather than through the fixture helper, so
the offset is stated by the test; the same case asserts `DefID` at `+0x10` is
zero on every record, because a read one word out in either direction would
otherwise pass on the zero case alone.

AC-1's round trip holds **by construction and is still measured**: the map
document keeps its own payload bytes, so reading one more word off a record
changes no byte of it. `roundTrip` fails inside itself on any inequality
(DD-1).

AC-2's second half is the negative FR-2 states — *no path silently gives an
entity the first roster entry*. Three routes to an ownerless entity are asked at
once: a map whose records name none, no map at all, and every entity this tree
builds outside `FromALM`. All zero, none 1.

## SC-2 — the instant's three references (FR-3, AC-3, DD-4)

```
--- PASS: TestTheThreeReferencesAnInstantNamesReachTheCompiledInstant
--- PASS: TestPlayerZeroNamedIsNotNoPlayer
--- PASS: TestAnInstantsReferencesMoveNoPlainParameter
--- PASS: TestAnUnresolvableActionUnitReferenceIsReportedAndAbsent
```

A node carrying all three compiles to three present references; one carrying
none to three absent. The unit reference goes in as the map's word `21` and
comes out as entity id `3`, which is what says it was **resolved** and not
copied.

DD-4's own claim — that the three do not join the plain-parameter packing — is
measured as a **difference between two compiles**, not as an assertion about
one. A node interleaving a literal, a unit, an integer, a group, an integer and
a player is compiled beside the same node with its three references struck out,
and the two `Args` arrays must be equal. A binder that packed any reference as a
plain parameter shifts the three integers and this fails.

AC-3's last clause is the hero band with no party: the reference is reported
exactly as before, the compiled reference is absent, and the **player beside it
still resolves** — so an unresolved unit is one field and not the whole block.
Supplying a hero makes the same node resolve and the report go silent.

## SC-3 — the byte form (FR-6, AC-7, P-3, DD-2, DD-3, DD-5)

```
--- PASS: TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth
--- PASS: TestMarshalledBytesArePinned
--- PASS: TestThePinnedBytesDecodeBackToThePinnedWorld
--- PASS: TestHashIsPinned
--- PASS: TestThePinnedDigestIsFNV1aOfThePinnedBytes
--- PASS: TestUnmarshalRefusesEveryVersionButTwelve
--- PASS: TestTheInstantRecordCarriesItsThreeReferences
--- PASS: TestADeadUnitKeepsItsOwnerThroughTheForm
--- PASS: TestTheCanonicalWorldsFieldSetsArePinned
--- PASS: TestTheScriptSectionRefusesTheBytesNoTickCanLeave/an_instant's_unit-presence_byte
--- PASS: TestTheScriptSectionRefusesTheBytesNoTickCanLeave/an_instant's_group-presence_byte
--- PASS: TestTheScriptSectionRefusesTheBytesNoTickCanLeave/an_instant's_player-presence_byte
```

Version **12**, entity record 87 → 91, instant record 44 → 59, both at their
tails so no earlier offset moves (DD-5). The offset table in `binary_test.go` is
a partition of the whole form and is checked as one, so the appended word could
not slide in unnoticed.

**The two derivations are the evidence that matters**, because a pin and a
digest computed from that pin agree by construction:

```
--- PASS: TestThePinIsThePreStoryPinPlusTheOwnerWord
--- PASS: TestTheRoutedPinIsThePreStoryPinPlusTheOwnerWord
```

Each takes this story's hand-transcribed form, removes exactly the four bytes
each record grew by, puts the version byte back to 11, and requires the result
to hash to a literal that **predates the story** — `0xb56c622a377738cf` and
`0x5e1a6c6e7ea13679`. Both reproduce. The second fixture carries stored route
cells after its records, so a word written four bytes wide in the wrong place
would leave the first derivation intact and break that one.

The same method runs three deep in `pkg/mapload`, where the chain is owner →
group → the eight combat fields, against three literals from three different
stories:

```
--- PASS: TestTheDerivedPlaneIsCarriedHashedAndReadBack
--- PASS: TestTheSingleArgumentEntryPointIsUnmoved
--- PASS: TestTheNewPinsAreTheOldPinsPlusTheEight
--- PASS: TestStructurePassLeavesATablelessWorldWhereItWas
```

The new digests were not recorded from a run. `pinDigest` and `rtfDigest` were
recomputed from the **hand-written transcriptions** with a third FNV-1a outside
this tree, checked against the published vectors first; `rlxTick1Digest` and
`hybTick1Digest` were assembled outside the tree from their own byte-level
descriptions and hashed there. Each then agreed with the encoder.

DD-2 is witnessed as a digest **difference**, which matters more here than
anywhere: nothing in this build reads an owner, so every behavioural test in the
tree would still pass with the field dropped between the constructor and the
encoder. `TestTwoWorldsDifferingOnlyInAnOwnerHashDifferently` is what would
notice, and it also asks that a world of unowned units and one owned by roster
slot 1 are **different worlds** — the distinction FR-2 turns on.

DD-3 and P-3: the constructor's not-alive block grew no clause and the decoder's
refusal family grew none. `TestADeadUnitKeepsItsOwnerThroughTheForm` asserts
that as behaviour rather than trusting the omission; the pinned world's dead
record carries owner `0xffffffff` through the form. Had a sixth refusal been
added by pattern-matching the other five, both arms would produce worlds this
package cannot read back.

The three new presence bytes are refused outside `{0,1}` rather than read as
truthy (AC-7), through the same one function the six existing flags use.

## SC-4 — the state landed before the arms did (FR-7, DD-10)

T1 landed with both arms still unimplemented, and said so in a test written to
have exactly that shelf life: `TestTheTwoHandOverArmsAreStillUnimplemented`
asserted the report named both opcodes and that **no owner changed when a
trigger carrying them fired**. It passed at `f50a0c2` and failed on the first
build of T2, which is the transition it existed to mark. T2 replaced it with
`TestTheReportStopsNamingTheTwoHandOverArms`.

DD-10: no reader was added. Every mention of the field in shipped code across
the whole tree:

```
$ grep -rn "\.Owner" --include=*.go pkg/ | grep -v _test.go
pkg/mapload/fromalm.go:272:  Group: u.GroupID, Owner: u.Owner,
pkg/sim/binary.go:266:       binary.LittleEndian.PutUint32(b[o+87:o+91], e.Owner)
pkg/sim/script.go:736:       w.entities[i].Owner = in.Player
pkg/sim/script.go:751:       w.entities[i].Owner = in.Player
pkg/sim/step.go:428:         // where the state is (Entity.Owner), and the scaled rule
```

Two writes by the arms, one marshal, one load, one comment. **No branch.** The
far-search budget's arm is unchanged; its prose, which asserted "no entity here
carries an owner", was corrected — that premise is now false and the arm stands
on the half that never depended on it, since the term asks whether a *human
participant* owns the mover and a roster slot does not answer that.

## SC-5, SC-6, SC-7 — the arms (FR-4, FR-5, AC-4, AC-5, AC-6, P-1, P-2, DD-6, DD-7)

```
--- PASS: TestTheGroupArmWritesEveryMemberAndNobodyElse
--- PASS: TestAGroupNoEntityCarriesLeavesEveryOwnerAlone
--- PASS: TestTwoGroupArmsInOneTriggerEachWriteTheirOwn
--- PASS: TestTheGroupArmDoesNotDependOnStorageOrder
--- PASS: TestTheUnitArmWritesItsOneEntity
--- PASS: TestTheUnitArmOnAnEntityTheWorldNoLongerHolds
--- PASS: TestTheUnitArmWritesAFelledEntity
--- PASS: TestNeitherArmPanicsOverAnEmptyWorld
--- PASS: TestAnArmNamingNothingWritesNothing/the_group_arm_with_no_player
--- PASS: TestAnArmNamingNothingWritesNothing/the_unit_arm_with_no_player
--- PASS: TestAnArmNamingNothingWritesNothing/the_group_arm_with_no_group
--- PASS: TestAnArmNamingNothingWritesNothing/the_unit_arm_with_no_unit
```

The fixture is five entities in two groups with **one member of the named group
felled**, at five distinct starting owners none of which is the player any arm
hands to — so "was written" and "was left alone" are never the same observation.
AC-4 comes out `{1:9, 2:9, 3:9, 4:4, 5:5}`, the corpse included.

FR-4's dead clause is asked **separately of each arm**, because the two find
their entity by different means: a lookup that filtered the dead would be
invisible to the group arm's cases.

P-1's storage-order independence (DD-6) is measured as a **digest equality**
between one world built from ascending entities and one from the same entities
reversed, after the arm has run — not as an inspection of the loop.

AC-6 asks each of FR-5's cases over a world whose owners a successful arm *would*
have overwritten, and adds the clause that makes it FR-5 rather than a weaker
claim: no owner may come back **zero**. Writing zero would be indistinguishable
from handing a unit to nobody, and this build separates "named nothing" from
"named a zero" everywhere else.

## SC-8 — no trigger was armed, and the escort runs (AC-8, AC-9, DD-8, DD-9)

```
--- PASS: TestImplementingTheseArmsArmedNoTrigger
--- PASS: TestTheEscortChoreographyRunsEndToEnd
--- PASS: TestTheReportStopsNamingTheTwoHandOverArms
```

DD-9's differential: **one script shape built twice** over identical triggers,
once with these two opcodes and once with an opcode this build still does not
run, then the inert set, the latch vector and the outcome compared. The fixture
deliberately carries all three trigger shapes — one live and firing, one genuinely
**inert** (it reads a register an unimplemented check owns), one live that does
not hold — so the comparison has a non-empty inert set to agree about rather
than two empty ones. Both builds report inert set `[1]`, latches `1,0,0`, and
outcome undecided; the arms demonstrably ran in the first build and demonstrably
did not in the second. No trigger became live, inert, evaluated or unevaluated.

DD-8: the supported-arm table is still the single answer, so the report and the
dispatch cannot disagree. FR-7's other half — the rule is otherwise unchanged —
is asked with opcode 2, `Send message`, the campaign's most authored instant:
the report names it and only it.

AC-9's choreography is built as a world and a hand-built script, not from a map.
A group of three goes to player 1; a second trigger, **gated on a register the
first trigger's own instant writes**, later hands one of those three to player 3.
The gate is what puts them in different passes, so nothing depends on instant
order within a trigger. Owners after pass one: `{1:1, 2:1, 3:1, 4:4, 5:5}`.
After the second: `{1:1, 2:3, 3:1, 4:4, 5:5}`. A further 128 ticks change
nothing — both triggers are one-shot, and a group arm that re-fired would put the
escortee back on player 1, which is the failure the ordering exists to expose.
`Script.Unsupported()` is empty and no trigger is inert, so no arm outside this
build's supported set was needed.

Resumption is covered by `TestAScriptHoldingBothArmsCrossesTheFormAndStepsOnAlike`:
a world the arms have already run on is cut, decoded, and both are stepped 64
ticks with digests compared every tick. A round-trip comparison alone would miss
a wrong latch vector, because re-running a spent one-shot arm against a world it
has already written changes nothing observable.

## What this story does not claim

**No progress on any mission's win condition**, and none is delivered. That is
the contract's own statement rather than a limitation discovered here; AC-8
exists to make a build that changed a trigger's state fail, and it passes.

**A research dependency is disclosed rather than closed.** The
instruction-level body of arms 19 and 22 is not published — their effect is
known from a whole-map catalogue rendering and from the parameter grammar, not
from a read of either arm. What remains unknown is whether the original re-keys
group membership, what it does when two owners share a group identifier, and
what it writes besides the owner pointer. The spec authorises the only reading a
tree with no group object can express and names the divergence; this build
implements that and nothing more. Research still owes a published claim for the
instant table's arm-by-arm effect, and neither arm is in the four the published
row names exactly nor the five it grades Medium.

**Two documentation errors were found and corrected rather than followed.**
`binary_test.go`'s fixture comment named second-record offsets `114, 115, 116,
124, 133` — a 44-byte record's — while the cases below already used a later
width's. A comment that names offsets no case uses cannot fail, which is how it
survived three widenings. And `step.go`'s far-search premise contained a
sentence this story makes false. Both are called out in the T1 commit body.

**One test was renamed rather than deleted.**
`TestATableBuiltWorldRoundTripsAtTheUnmovedVersion` asserted a version that no
longer holds still; the round trip it measures is unchanged and the
version-is-unmoved half belonged to a story that added no field.
