# 0149 — the walkable save

## Contract

This tree's reading of an original save says what the current claims say, and its one authored
guess about who leads the party becomes a cited fact.

Three things follow. The `Player` record programme gains the record it was short by. The party's
leader is selected by a decoded field instead of an inferred id. Every sentence this tree writes
about a save — in source, in the developer tool and on the owner's screen — stops reasoning from a
refuted claim.

No simulation behaviour changes on any file in the preserved corpus. That is a measured result, not
an assumption, and `verification.md` carries the measurement.

## Background, self-contained

An original save is `game####.sav` at an install root. `pkg/formats/sav` decompresses it and walks
the object stream: it starts at the first top-level record, which is the human participant's own
`Player`, and follows that class's serialization programme through its groups, each group's actor
list, each actor's record and the items those reference. `pkg/game` turns the characters the walk
reached into this tree's party when a mission is resumed from such a file.

Three facts about that stream are load-bearing here.

**The `Player` record ends with a `Diary`.** `Player::Serialize` has three tail calls, not two:
after the group set and 32 raw bytes it dispatches an object at `Player+0x40` through that object's
own vtable. The object is a `Diary` — a counted array of dwords, a counted array of words, and one
dword resolved through the identity map on load. It is written **inline**: no reference tag
introduces it and it takes no index in the archive's shared counter. A programme without it stops
short by that record's length and reads whatever follows as the next construct.

**The top-level list continues past the first record.** The document writes a `u32` count and that
many object references, of which the first is the participant's `Player`. A walk that reads one
`Player` and stops has read one element of a list, not the whole file.

**One field on the `Player` names the participant's own starting character.** `Player+0x34` holds
an identity key that is resolved through the identity map on load. It equals the identity key of
exactly one actor in that participant's own groups on 18 of 18 distinct saves. Nothing else in the
record ranks the actors: the actor list is a `CObList` written head to tail with no comparison in
the store arm, and one actor set has been observed in three different orders across one session.

Before this story, `pkg/game` chose the party leader by a different route: the actor carrying
runtime creation-order id 1. That id is assigned at tick-list insert as the lowest free bit of a
bitmap with id 0 pre-marked, so the participant's own character holds 1 while nothing has been
freed. It is an inference from the allocator, disclosed as an authored rule, and it can in
principle fail on a save with mid-session churn.

## Functional requirements

**FR-1 — The `Player` record programme reads its `Diary`.** `pkg/formats/sav`'s walk reads the
inline `Diary` at the end of a `Player` record, with no tag and no archive index, and files what it
read on the `Player`. The walk's reported extent is the end of that record.

**FR-2 — The participant's starting character is a decoded value the package exposes.** A character
carries its own identity key, and carries whether the enclosing `Player`'s starting-character field
names it. A consumer needs neither the field's offset nor the record's internals to ask which
character is the participant's own.

**FR-3 — The restored party is ordered by that decoded value.** `pkg/game` puts the character the
file names at the head of the party. Where the file names no character, or names more than one, the
runtime-id rule is the fallback; where that does not resolve either, the file's order is kept
unchanged. Which rule applied is reported.

**FR-4 — Every living sentence about a save matches the current claims.** No documentation comment,
no developer-tool line and no text the build shows the owner reasons from the refuted headline —
that a walk reaches one subtree and a terminator and everything else needs a pattern scan. The
resume report states what the file does not carry in terms of what is outside the participant's own
subtree, which is a property of the document rather than of where a walk stopped.

**FR-5 — The corpus measurement is reproducible from a committed command.** `cmd/savtool` answers,
over any set of save files, which character each of the two ordering rules selects and whether they
agree.

## Acceptance criteria

**AC-1** A walk over a synthetic save whose `Player` carries a `Diary` reads that `Diary`, reports
the `Player`'s extent as ending after it, and files the two array lengths on the `Player`.

**AC-2** The `Diary` takes no index in the archive's shared counter: an object reference written
after the `Player` record still resolves to the object the archive's own numbering names.

**AC-3** Over the 23 preserved save files, 18 distinct by content, the walk reads every `Player`
record without error and the record's own trailing dword equals that `Player`'s identity key on all
18. That is an internal consistency check no programme with a field at the wrong width satisfies.

**AC-4** Over the same 18 files, the word immediately after the `Player` record is a legal element
of the top-level `Player` list — an instance tag naming `Player`, or the null reference.

**AC-5** A character built from a save reports its own identity key, and reports itself as the
participant's starting character exactly when the enclosing `Player`'s field names it.

**AC-6** Over the same 18 files, exactly one character per file is named by that field, and it is
the same character the runtime-id rule selects. Any file where the two disagree is named.

**AC-7** A restored party whose file names a starting character is led by that character. A party
whose file names none is led by the character carrying runtime id 1. A party where neither resolves
keeps the file's order.

**AC-8** The resume report names which of the three rules ordered the party.

**AC-9** No decoded value any character carries changes on any of the 18 files. Statistics, pools,
positions, worn pieces, carried items, spellbook counts and journal counts are identical before and
after FR-1.

**AC-10** No source comment, tool line or owner-facing string in this tree asserts that the object
stream ends after the first record.

**AC-11** The resume report and the LOAD GAME caveat still fit the widths they had, and the caveat
still names what does not load.

## Design decisions

**DD-1 — The `Diary` is a step of its own, not a base class and not a reference.** The walk already
has a step that runs another class's programme in place, and it means inheritance: a derived class's
record begins with its base's. It also has a step for `ar << CObject*`, which reads a tag and takes
an index. A vtable dispatch on a member pointer is neither. It gets its own step so the two
mechanisms stay distinguishable in the programme table, which is the table a reader checks a
serializer against.

**DD-2 — The decoded field wins and the inferred id becomes the fallback.** `SAV-HERO-059` is a
named store, a named load-side resolution and 18 of 18 agreement with a rival excluded by value.
`SAV-ID-015` grades its own ordering use Medium as a law: a corpse decay frees a bitmap bit and the
next spawn reuses it. The weaker rule is kept because it is measured to agree on the whole corpus,
so it costs nothing and covers a file whose starting character does not resolve.

**DD-3 — The walk still reads one top-level record.** The corrected programme reaches the second
`Player`'s tag, and the enumerated document says how many elements follow. It is not taken here:
the whole decoded stream tiles on 7 of 18 saves and desynchronises inside a `Unit` subtree on the
other 11, so a consumer built on it would be right on 39 per cent of the corpus. The extent test
this story writes reads the next tag as evidence and does not step into it.

**DD-4 — Landed stories' documents are not rewritten.** `docs/0147-party-from-the-save/` and
`docs/0148-the-resumed-mission/` stay as published. What is corrected is what is living: the source
comments, the developer tool and the text the build shows. `provenance.md` states why, and what was
considered and rejected.

**DD-5 — What a save records of what the participant has seen is out of scope.** `0148` L-3
discloses that a resumed mission shows only what the party reveals after the load, and that is
still what the build does and still what it says. This story writes no statement about whether the
file carries such a record: the research is live and this tree states nothing it cannot cite.

## Limits

**L-1 — The ordering rule's fallbacks are still authored.** `SAV-HERO-059` decides the leader on
every file in the corpus. What to do with a file that names no starting character, or two, is this
tree's choice: the runtime-id rule, then the file's order. Neither case is observed here.

**L-2 — More than one human participant is not handled.** The walk reads the first top-level
`Player`. `SAV-HERO-059` grades the multi-participant case Unknown and every save in the corpus has
one participant.

**L-3 — The rest of the object stream is still not reached.** Ground loot, sacks, corpses, the
map's own units, the cell records and the session block are outside the participant's own subtree
and are not restored. This story does not change that; it changes only how the limit is described.

**L-4 — The `Diary` is read and nothing is applied from it.** What its two arrays hold is Unknown
and this tree has no journal.

**L-5 — No number in the milestone census moves.** This story changes no script node and no
mission behaviour. Its result is a corpus measurement and a set of corrected statements, and
`verification.md` reports the census before and after to show it did not move.
