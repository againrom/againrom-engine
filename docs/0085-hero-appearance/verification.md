# Verification — the party member is drawn as what he wears

## The gate

Each command run on its own and branched on **its own** exit code, not on a grep of its output.

```
go build ./...                                  EXIT=0
go vet ./...                                    EXIT=0
gofmt -l $(git ls-files '*.go')                 EXIT=0   (no output)
go test -count=1 -trimpath ./...                EXIT=0   31 packages ok, 0 FAIL
bash scripts/check-no-game-assets.sh            EXIT=0   clean (tree scan)
bash scripts/check-no-game-assets.sh --history  EXIT=0   clean (history scan)
bash scripts/check-doc-budget.sh                EXIT=0   every artifact and both chain relations ok
bash scripts/check-sdd-audit.sh                 run in the orchestrator seat after this file lands
```

`-trimpath` is used because Windows Defender quarantines one test binary otherwise.

`check-sdd-audit.sh` is reported and not pasted for a reason worth stating: run from a worktree its
note and warning **counts** are meaningless — the `builds/` check is guarded by `[ -d builds ]` and a
worktree has no `builds/`, that directory being untracked rather than ignored. Only the FAIL set is
enforced and only it is comparable. Run here before this file existed it reported exactly one FAIL,
`0085-hero-appearance: every task in tasks.md has landed and there is no verification.md`, which is
this file's own absence.

## The deletion set, and what the diff does not touch

```
git diff --diff-filter=D --name-only c49d5b8 HEAD   -> empty
git diff --name-only c49d5b8 HEAD | grep pkg/sim    -> 0 files
```

The second is FR-7's evidence and it is stronger than a test: **`pkg/sim` does not appear in this
story's diff at all**, so no state field moved, no encoder changed and `formatVersion` was not
touched. A test pinning the version literal was written and then removed — see *Mutation testing*.

Trailers: `T1`, `T2`, `T3`, `T4` exactly once each, on the four code commits and nowhere else; the
two evidence-stage commits carry none. No `Co-Authored-By`. The submodule sat frozen at `b2876c0a`
with no leading character throughout.

## AC evidence

| | Witness |
|---|---|
| **AC-1** | `TestEveryPublishedNameAnswersItsClass` — the seventeen names against a list transcribed independently of the table, plus the distinct-key count |
| **AC-2** | `TestAnUnmatchedNameFallsBackOnTheForcedClass` — six unmatched names including two the shield suffix can actually compose (`swordsman2h_`, `archer_`), each answering the fallback with the flag clear |
| **AC-3** | `TestTheNameIsComposedFromWhatIsWorn`, the shield rows |
| **AC-4** | `TestTheNameIsComposedFromWhatIsWorn`, the mage and dying rows — including that the mage substitution does **not** fire on a name other than `unarmed`, and that a dying mage reaches the staff body from either side |
| **AC-5** | `TestTheDirectoryIsTheThreeWay` — all sixteen blocks against an independent transcription, both special arms asserted at every one of them, and the out-of-range refusal |
| **AC-6** | `TestTheSheetAddressIsComposed` — the base, the sheet, the sibling, and the empty-argument answers |
| **AC-7** | `TestThePartyMemberIsNoLongerDrawnAsAnUnarmedMan` and `TestTheMissionPartyIsOneMemberAtTheStartCell` — the placed member's key is the law's answer for the authored body and is not the fallback |
| **AC-8** | `TestAHeroBodyDrawsTheComposedSheetOnTheRecordsGeometry` — the two fixture sheets differ in frame count and in every frame's size, so which half came from where is read off the frames |
| **AC-9** | `TestAnUnresolvableHeroBodyIsASkip` — five causes: no entry, the sheet under the other directory, a name resolving to no record, a sheet of no frames, an empty name |
| **AC-10** | `TestOnlyTheEntityAnArtEntryNamesDrawsTheBody` — two entities of **one** class key, one with an entry and one without |
| **AC-11** | `TestABodyNameReachesNoWorldByte` — two parties differing in nothing but the body name, identical bytes and identical digest; and the diff evidence above |
| **AC-12** | `TestTheAuthoredBodyAndTheAuthoredSkillAreOneChoice` — the pair, and that the authored slot still trains the weapon the rule reads |

## Success criteria

**SC-1**, **SC-2**, **SC-3**, **SC-4** are the four `pkg/data` tests above, run green.
**SC-5** and **SC-10** are AC-7's and AC-12's witnesses. **SC-6** and **SC-7** are AC-8's and AC-9's.
**SC-8** is AC-10's witness plus `TestPartyArtPairsTheStartsIdsWithTheBundlesBodies` and
`TestAStartedMissionDrawsItsMemberFromTheResolvedBody`, which close the join between the lookup
builder and the driver. **SC-9** is AC-11's. **SC-11** is the gate block above.

## Properties

**P-1** — `grep` over the tree finds no class-key literal on the party's path: `MissionParty` is the
one expression that produces one, and it produces it by calling the law. The retired constant is
gone and its site carries a tombstone saying what replaced it and why a number did not.

**P-2** — `pkg/render/terrain` gained one field and no import; its own tests and the import-graph
check are green. The bundle's new map is keyed by a plain string precisely because that tier may not
import the package that owns the body-name type.

**P-3** — nothing in the tree maps a weapon, a shape name, a skill slot or a statistic onto a body
name. The authored name is a literal at one site, `PartyBody`, reached from nothing;
`TestTheAuthoredBodyAndTheAuthoredSkillAreOneChoice` is what makes the *pairing* a checked claim
rather than a comment.

**P-4** — `TestNoOverrideLeavesEveryEntityOnItsOwnClass` over three shapes of empty lookup, and the
map-placed half of `TestAStartedMissionDrawsItsMemberFromTheResolvedBody`. The whole existing suite
is green with no test's expectation weakened.

## Mutation testing

Seventeen mutations, each applied, compiled, run against the targeted package and reverted. A
mutation the compiler rejects is not evidence; none of these was rejected.

Killed on the first pass (15): the arm `swordsman -> 3` changed to the fallback; the fallback
constant changed; material block 14 flipped to the other directory; the sprite prefix dropped from
the composed base; the two name substitutions applied in the other order; the mage and no-armour
directory arms swapped; the shield suffix never appended; the party's class forced back to the
fallback; the member's body name emptied; the authored body moved without the skill slot; the
party's directory hard-coded to the other one; the body keeping the record's frames; the composite
losing the record's geometry; an unresolvable body installed anyway; the composed path pointed at
another body's name.

Killed on the driver (5, run after the survivor below was fixed): the push's override test inverted;
`partyArt` keyed by party index instead of the start's minted id; `partyArt` installing unresolved
bodies; `openMission` passing no lookup; and `NewFrontEnd` dropping the call that resolves the
party's body.

**Two survivors, both fixed, both re-run and killed.** They are the same shape and it is worth
naming: *a fixture that is empty exactly where the code under test looks makes a passing test
indistinguishable from an absent one.*

- **Clearing the body's tier slices survived.** The loader fixture's class declared no tier, so
  clearing a nil field was a no-op and deleting the line changed nothing. The fixture now declares
  one and ships its colour table, the record resolves a real tier slice of its own two frames, and
  the assertion has something to refuse.
- **The one line in `NewFrontEnd` that resolves the party's body was not covered at all.** The
  install fixture's registry declared no classes, so the resolution succeeded at doing nothing. The
  fixture now carries the class the authored body resolves to and a sheet at the address the law
  composes — taken *from* the law rather than written out, so it cannot drift from what startup
  reads — and `TestUnitBundle` asserts the body arrived.

**One test removed rather than kept.** An assertion pinning the byte form's version literal at 14
was written into `TestABodyNameReachesNoWorldByte` and then deleted. "The form is still the one 0085
was written beside" is a claim this story has no standing to make and a tax on every later bump —
0084 deleted exactly such a pin after 0081 landed a version underneath it, and repeating that two
stories later would be worse than making it. The diff evidence above replaces it and is stronger.

## One correction to the source

`HERO-APPEAR-042` summarises its seventeen arms as giving **fifteen** distinct class keys. They give
**sixteen**: the keys are `1 2 3 4 5 7 8 9 10 11 12 13 14 14 15 24 23`, and `archer`/`bowman` is the
only pair that shares one. The claim's own enumeration and the raw-byte reproduction beside it carry
the same seventeen rows and agree with each other; only the adjective disagrees, and the enumeration
is the evidence. No individual name-to-key pair moves, so nothing built here rests on the difference.
Reported back rather than worked around; `provenance.md` carries it too.

## What this tree now has of the published law, and what it does not

Has: the seventeen-name mapping and its total fallback; the shield suffix; both substitutions; the
directory three-way with its sixteen material values; the composed sheet address and its sibling; the
class-record-supplies-the-geometry split; and the fact that a placed actor and a player's character
are drawn by two different rules.

Does not, each disclosed at the site that would otherwise imply otherwise:

- **The ordered name list.** Slot 0's five-bit field indexes a list built at run time; the corpus
  pins the set of names and not the order. This tree has no such slot either, so the step from what
  a character carries to which body he wears is missing on both sides. It is **AUTHORED** at
  `PartyBody` — absence established positively, disclosed, named, one function.
- **The recompute.** The original rebuilds the appearance on ordinary per-actor state messages. This
  tree derives it once, where the party is built, because nothing here can change what a member
  wears.
- **The dying body.** The original forces a dying character into one of two named bodies. A fallen
  member here falls through the composite's own corpse link, which is the placed unit's rule.
- **The equipment channel.** No slot array, no shield, no armour. The shield suffix and the material
  arm of the directory are held and reached by nothing this tree runs.

## Limitations

The agreement between a hero sheet's frame count and its class record's predicted total is a fact
about a shipped install and is asserted **nowhere here**; what is asserted is that the loader carries
the record's descriptor and the sheet's frames across unchanged. Every fixture in this story is
synthetic and no test reads a game install.

Nothing was run against a lawful install from this seat, so what the owner will see is not measured
here — only that the figure now resolves through the game's own chain to a class that is not the
bare-handed body, and draws from the sheet the composed path names.
