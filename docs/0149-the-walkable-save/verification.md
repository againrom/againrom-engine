# 0149 — verification

Research pin `eafab79`. Base `6aca1b9`. No `tasks.md`: this story was implemented in one lane
context, so no executor was briefed and no commit carries a task trailer.

## Corpus and instruments

The corpus is 23 preserved save files, 18 distinct by MD5:
`gameversions/en` and `gameversions/ru` (4 each, identical pairwise),
`gameversions/saves/2026-08-02` (12) and `gameversions/saves/2026-08-12` (3). Read-only.

Two instruments, both committed:

```
savtool lead FILE...     # which character each ordering rule selects, per file
savtool party -n 0 FILE  # the walk, its extent, and the Player's own Diary
```

One instrument is not committed and is named here: a throwaway `cmd/x0149dump` printed every
decoded value a save's party produces — class, offset, name, cell, fine position, runtime id, map
unit id, definition row, death stage, all fourteen statistic words, the spellbook and journal
counts, the declared item count, and every worn and carried piece with its code, row and stack. It
used only API that exists on both sides of Step 1, so the same binary source compiles against
`6aca1b9` and against this branch. It was run on this branch, then with `git stash push -- pkg`
applied, then restored. It is not committed because `internal/archtest`'s allow-map is fail-closed
and a package whose only purpose is comparing against pre-0149 code is dead weight after landing.

## FR-1 — the `Player` record programme reads its `Diary`

**Our programme was short by the record.** Before this story `pkg/formats/sav`'s `Player`
programme was `playerMembers`, the group set, 32 raw bytes, and nothing else. `SAV-PLDIARY-054`
states a third tail call at `L07961` dispatching `*(Player+0x40)`, a `Diary`. The programme now
carries it as a `stpInline` step.

The measurement, over all 23 files:

```
en/game0000.sav   record 1: Player @0x000057..0x000ad8 (2689 bytes), then 01 80 09 56
                  its own Diary @0x000806..0x000ad8: 119 dwords, 119 words,
                  reference 0x02c7df10 (its own key is 0x02c7df10)
```

Every file reads the same shape: 119 dwords, 119 words, and a trailing reference. The record is
722 bytes long on every file, which is `2 + 4×119 + 2 + 2×119 + 4`. Before the change the walk
ended 722 bytes earlier, at the `CDWordArray` count.

**What the old extent test was reading.** The four bytes at the old end are `77 00 00 00` on every
file. `0x0077` is 119, the `CDWordArray`'s own count, which the old test read as a back-reference
tag; the next two bytes are the first two of the 476-byte dword array, which it read as the `0000`
terminator. The retracted `SAV-TOPLVL-052` headline was measured on exactly those bytes.

## FR-1, AC-9, SC-4 — nothing decoded changed

478 lines of decoded values over the 18 distinct files, dumped before and after Step 1 and
compared with `diff`. **The difference set is empty.** No statistic, pool, position, worn piece,
carried item, spellbook count, journal count, definition row, death stage, runtime id or map unit
id moved, and no file gained or lost a walk error.

That is the expected result and it is why it was measured rather than assumed: the `Diary` is the
**last** construct in the `Player` record, so a programme short by it reads every earlier field at
the right offset and simply stops early. The party data this tree has been restoring since 0147 was
sound. A `Diary` anywhere but last would have desynchronised everything after it.

## AC-3, SC-3 — the walk reads every `Player` without error

`savtool party -n 0` over all 23 files: no `WALK STOPPED` line on any file.

The record's own internal check holds on 23 of 23: `Diary::Serialize`'s trailing dword equals the
enclosing `Player`'s identity key on every file (`0x02c7df10`, `0x02d0d100`, `0x02c7dd60`,
`0x02abf4a0`, `0x02b6d4a0`, `0x02cd0b20`, `0x02d008d0`, `0x02f4f930`). A programme with a field at
the wrong width lands those four bytes at an arbitrary offset, and an arbitrary dword does not
equal a value already read from the same record on 18 distinct files.

## AC-4 — the record ends where the next list element begins

The word at `Record.End`, over all 23 files: `0x8001` on 22, and `0x0000` on
`saves/2026-08-02/game0010.sav`. Class index 1 is `Player` — `readPlayers` fails the file outright
if the first class the stream introduces is not `Player` — so `0x8001` is an instance tag naming
`Player`. Both values are legal elements of the top-level `Player` list (`SAV-DOC-053`). By
distinct file that is 17 of 18 and 1 of 18.

## FR-2, AC-5 — the starting character is a decoded value

`playerMembers`' `F34` is renamed `Hero` and documented against `SAV-HERO-059`. The offset, width
and order are untouched; the name was referenced in exactly one place in the tree, which the
compiler found.

`sav.Character` gains `Key` (the record's own identity key) and `Hero` (the enclosing `Player`'s
field names it). `PartyWalk` sets `Hero`, being the one place holding both the `Player` record and
its characters. A key of zero on either side names nobody.

`TestThePlayerNamesTheParticipantsOwnCharacter` covers both arms on a synthetic save whose actor
list is written with the mercenary first, so a rule that took the head of the list answers 0 where
the correct answer is 1.

## FR-3, DD-2, AC-6, AC-7, AC-8, SC-2 — the ordering rule

**The two rules agree on 23 of 23 files, 18 of 18 distinct.** `savtool lead` over the corpus:

```
23 file(s): 23 agree, 0 disagree, 0 unresolved
```

On every file exactly one character is named by `Player+0x34` and exactly one carries runtime id 1,
and they are the same character. The selected position varies across the corpus — 0, 1, 3 and 4 all
occur — so the agreement is not an artefact of both rules answering 0. There is no file on which
they disagree.

`leadFirst` now applies three rules in order: the file's own field, then the runtime id, then the
file's order unchanged. `RestoredParty.LeadRule` carries which one applied and the resume report
prints it (AC-8), because two of the three are this tree's own choice.

`TestTheParticipantsOwnCharacterLeadsTheRestoredParty` is table-driven over seven cases including
the one the corpus cannot supply: a file where the field names one character and the runtime id
another. The field wins.

Deleting `leadFirst`'s first arm — the decoded-field rule — fails **three** of the seven cases and
leaves four passing. Two of the three fail on the reported rule alone, which is FR-3's disclosure
half. Exactly one fails on the party order itself: the case where the two rules name different
characters ends `[Witch Danath]` instead of `[Danath Witch]`. So the arm is witnessed, and the test
distinguishes an ordering change from a reporting change rather than lumping them.

**This reaches hashed simulation state.** The leader is party index 0, which reaches placement, so
the corpus agreement above is a necessary check and not a sufficient one. The sufficient one is the
resumed world itself.

`cmd/savecheck` was built from master `6aca1b9` and from this branch and each run against the nine
mid-mission saves the corpus holds, comparing the resumed world's own hash and entity count:

```
game0000.sav SAME hash aeab8d78a6c51578, 57 entities   game0005.sav SAME hash d96e7e21dacf2f16, 57
game0001.sav SAME hash fc91065e89859a3d, 57 entities   game0007.sav SAME hash dc96a70884a4a093, 57
game0002.sav SAME hash f9235ce2e75f1617, 36 entities   game0008.sav SAME hash c9bdfef632b8d8ce, 57
game0003.sav SAME hash b7fd15b7a08d5754, 36 entities   game0009.sav SAME hash 8e2e433d9e24acd8, 57
game0004.sav SAME hash 1c70e4f1860af686, 36 entities
```

Nine of nine identical. The other saves in the corpus are between-mission or restart files, which
this build refuses to resume and which therefore build no world to hash.

The report the driver prints, on `game0007.sav`:

```
party: 5 characters restored, 5 with statistics and pools, 5 off their own Humans row
led from position 3 by the character the file names as the player's own
```

Position 3 is what `savtool lead` reports for that file under both rules.

## FR-4, DD-4, AC-10 — the living sentences

`SAV-TOPLVL-052` was cited as live reasoning in six places. Five are living text and were
rewritten; one is a test whose whole subject was the refuted measurement:

- `pkg/formats/sav/program.go` — the file comment and the tag-constant comment.
- `pkg/formats/sav/party.go` — `Character`'s comment and `PartyWalk`'s.
- `pkg/game/originalsave.go` — `OriginalSaveResume`'s comment and the `NOT CARRIED` line the
  resume report prints, which is text the owner reads.
- `cmd/savtool/party.go` — the extent line.
- `pkg/formats/sav/program_test.go` — `TestTheWalkEndsOnTheTerminator` became
  `TestTheWalkEndsWhereTheNextItemBegins`, and its doc block records that its old form could not
  have caught the short programme: its expected value was produced by the bytes of the record it
  was checking.

The replacement statement everywhere is the same: what the walk reads is the human participant's
own subtree, and the rest of the file is later items of the same document (`SAV-DOC-053`). That is
a property of the document rather than of where a walk stopped, so it does not go stale when the
walk changes.

`grep -rn "SAV-TOPLVL-052"` over the tree outside `docs/` and `research/` now returns nothing.

**DD-4 — landed documents.** `docs/0147-party-from-the-save/` and `docs/0148-the-resumed-mission/`
are unchanged, including `0147/provenance.md`'s row grading the retracted headline High and
`0147/spec.md`'s AC-4. `provenance.md` states the reasoning and names the edit considered and
rejected. `git diff --stat 6aca1b9..HEAD -- docs/0147* docs/0148*` is empty.

**DD-5 — out of scope.** What a save records of what the participant has seen was in this story's
brief and was withdrawn from it during the work: the finding it rested on is under re-derivation in
research. No statement about it is made in this story's documents, in the source, or in any text
the build shows. `0148` L-3's disclosure stands unchanged and the build behaves exactly as it did.

## FR-5, AC-11 — the tool and the widths

`savtool lead` is the committed instrument for every figure above. It reports per file which index
each rule selects and how many characters each matched, prints the full character list only where a
file does not agree, and exits non-zero if any file disagrees.

`savtool party` marks the participant's own character with `*` and prints the `Player`'s `Diary`
beside its extent.

`OriginalSaveNote` is unchanged — 100 of the 104 columns `pkg/ui` can draw, still naming what does
not load. `TestTheOriginalSaveCaveatIsSayableOnOneLine` still passes.
`TestARestoredPartyReportsEveryAxisItDidNotApply` covers all three lead-rule wordings.

## DD-1, AC-1, AC-2 — the inline step

`stpInline` is a step of its own beside `stpClass` (inheritance, members merge into one record) and
`stpObjRef` (a tag is read, an index is taken, a back-reference can name it). A vtable dispatch on
a member pointer is neither.

`TestThePlayerCarriesItsOwnDiary` is AC-1 and AC-2 together. It builds a synthetic save whose
`Player` carries a `Diary` of 5 dwords and 3 words — deliberately different counts, because the
corpus writes 119 for both and a programme reading one count for both would tile that and not this
— then asserts the walk filed one `Diary` on the `Player`, both array lengths, the trailing
reference against the `Player`'s own key, and that the `Diary`'s end and the `Player`'s agree,
which is the extent half of AC-1.

AC-2 is the index: the record took archive index 0. The walker's counter starts at 1, so 0 is a
value no tagged object holds. The assertion is direct rather than observed through a consequence,
and that is a real limit of it: the `Diary` is the last construct in the `Player`, so within the
one record this walk reads there is no later tag for a wrong index to shift. The consequence lands
on a consumer that walks past this record, which is DD-3's out-of-scope work.

## DD-3 — the walk still reads one top-level record

`Walk` is unchanged in extent: it reads the first top-level record and returns. The corrected
programme now reaches the second `Player`'s tag and reads it as evidence only. `SAV-DOC-053` tiles
the whole decoded stream on 7 of 18 saves and desynchronises inside a `Unit` subtree on the other
11, so nothing is built on it here.

## SC-1 — the gate

Run in `wt-0149` at the head commit, clean tree.

```
go build ./...                          clean
go vet ./...                            clean
gofmt -l $(git ls-files '*.go')         prints nothing
go test -trimpath -count=1 ./...        all packages ok
bash scripts/check-no-game-assets.sh    clean (tree scan)
bash scripts/check-doc-budget.sh        0149 spec 10061, plan 5368, provenance 6350; chain ok
bash scripts/check-sdd-audit.sh         ok, no enforced failure
bash scripts/check-hotfix-ledger.sh     ok
git log --format='%h %(trailers:key=Co-Authored-By)' 6aca1b9..HEAD   no trailer on any commit
```

`check-sdd-audit`'s note and warning count is not comparable from a worktree, which has no
`builds/`. Only its FAIL set is, and it is empty.

## SC-5 — the milestone census

`pipeline/check-milestone.sh` drives `implementation/builds/current/missionrun.exe`, which is
rebuilt from master at a landing, so a lane cannot run the gate itself against its own branch. Two
readings were taken instead.

**Before**, the gate as it stands, driving `builds/current` at master `6aca1b9`: exit 0, output
identical to `pipeline/milestone-baseline.txt`.

**After**, the same script with `drive` pointed at this branch's own `missionrun`: exit 0, output
identical to the same baseline. Directly:

```
AGAINROM_ASSETS=gameversions/en missionrun -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   17
AGAINROM_ASSETS=gameversions/en missionrun -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   13
```

Both match master's baseline: mission 10 is `1 + 2 + 13 + 1`, mission 20 is `1 + 11 + 1`. **The
census did not move, and this story was never going to move it.** It changes no script node, no
mission behaviour and no value any save produces. Its result is the corpus measurement above:
the two ordering rules agree on 18 of 18 distinct saves, and the record programme was short by a
record whose absence changed nothing.
