# 0149 — plan

## Step 1 — the `Player` record programme reads its `Diary` (FR-1, DD-1, DD-3)

`pkg/formats/sav/program.go` gains one step op. It runs a named class's programme at the cursor as
a record of its own, without reading a tag and without taking an index in the archive's shared
counter, and files the result on the enclosing record under a site name. The `Player` programme
gains that step for `Diary` after its 32 raw bytes. The `Diary` programme already exists in the
table and is unchanged: `Humanoid`'s thirteenth reference reaches the same class through a tag.

The inline record is given index 0. The walker's counter starts at 1, so 0 is a value no tagged
object can hold and it reads as "this record was not numbered". It is not entered in the
back-reference table, which is what keeps a later tag resolving to the object the archive names.

`Walk` still reads one top-level record and returns. `class.go`'s comment on the `Player` field
list is corrected to name the `Diary` as part of what follows the straight run.

## Step 2 — the starting character is exposed and named (FR-2)

`class.go` renames the `Player` field currently carried as `F34` to `Hero` and documents it against
`SAV-HERO-059`. The name is the only change to the decode; the offset, the width and the order are
untouched.

`party.go`'s `Character` gains two fields: `Key`, the record's own identity key, and `Hero`, true
when the enclosing `Player`'s field names it. `PartyWalk` is where `Hero` is set, because that is
the one place holding both the `Player` record and the characters at once. A key of zero never
matches, so a record with no identity key cannot become the hero by accident.

## Step 3 — the party is ordered by the decoded field (FR-3, DD-2)

`pkg/game/originalparty.go`'s `leadFirst` takes the characters and answers the position it led from
and which rule applied. The order is: the file's own starting character where exactly one is named;
otherwise the character carrying runtime id 1 where exactly one carries it; otherwise the file's
order unchanged. `RestoredParty` gains a field naming the rule, and `heroRuntimeID` keeps its
comment as the fallback's own justification.

## Step 4 — the living sentences are corrected (FR-4, DD-4, DD-5)

Four places assert the refuted headline and are rewritten against `SAV-DOC-053` and
`SAV-PLDIARY-054`:

- `pkg/formats/sav/program.go` — the file's opening comment and the tag constants' comment.
- `pkg/formats/sav/party.go` — `Character`'s comment and `PartyWalk`'s.
- `pkg/game/originalsave.go` — `OriginalSaveResume`'s comment and the `NOT CARRIED` line of the
  report it prints.
- `cmd/savtool/party.go` — the extent line.

The replacement statement is the same everywhere: what the walk reads is the human participant's
own subtree, and the rest of the file is outside it. That is true of the file and does not depend
on where a walk happens to stop.

The LOAD GAME caveat is unchanged. It never cited the retracted headline, it fits its 104 columns
exactly as it is, and the counted report beside it is where every unapplied axis is named.

## Step 5 — the corpus measurement is committed (FR-5)

`cmd/savtool` gains a `lead` verb taking any number of files. Per file it prints the `Player`'s
starting-character key, each actor's identity key and runtime id, the index each rule selects and
whether they agree; then the totals. It is the tool that produces `verification.md`'s figures, so a
later reader re-runs one command instead of trusting a number.

## Success criteria

**SC-1** `go build ./...`, `go vet ./...`, `gofmt -l` and `go test -trimpath -count=1 ./...` are
clean, and the four repository scripts pass.

**SC-2** `savtool lead` over the corpus reports 18 distinct files, both rules resolving exactly one
character each, and the agreement count.

**SC-3** `savtool party` over the corpus reads every `Player` without a walk error, and the
`Player`'s trailing dword equals its own identity key on every file.

**SC-4** The character values every file produces are compared before and after Step 1, field by
field, and the difference set is reported.

**SC-5** `pipeline/check-milestone.sh` is run before and after, and both readings are recorded.

## Risks

**The `Diary`'s two counts could be a coincidence.** A `CDWordArray` count of 119 on every file
would parse as a plausible array whatever it really was. The trailing dword equalling the
`Player`'s own identity key is what rules that out: a wrong-width programme lands the last four
bytes somewhere arbitrary, and an arbitrary dword does not equal a value already read from the same
record on 18 files.

**Renaming a decoded field could silently change a consumer.** The old name is referenced in one
place in the tree, which the compiler finds.

**The ordering change could reach hashed simulation state.** It decides which member is party index
0, which reaches placement. It is mitigated by the measurement: the two rules select the same
character on every file in the corpus, so no world this tree can build from a preserved save
changes.

## Traceability

| FR | DD | Step | SC |
|---|---|---|---|
| FR-1 | DD-1, DD-3 | 1 | SC-1, SC-3, SC-4 |
| FR-2 | — | 2 | SC-1, SC-2 |
| FR-3 | DD-2 | 3 | SC-1, SC-2 |
| FR-4 | DD-4, DD-5 | 4 | SC-1 |
| FR-5 | — | 5 | SC-2, SC-3 |
