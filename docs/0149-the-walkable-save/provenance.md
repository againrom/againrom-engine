# 0149 — provenance

Research pin: `eafab79`. Read a row with `cd research && go run ./tools/claim <ID>`.

## Claims this story stands on

| Claim | Confidence | What it supplies |
|---|---|---|
| `SAV-PLDIARY-054` | High | `Player::Serialize` has a third tail call, a `Diary` at `Player+0x40`, dispatched at `L07961` through the object's own vtable. `Diary::Serialize` at `L08081` is a `CDWordArray`, a `CWordArray` and a `u32` resolved through the identity map on load. This is the record FR-1 adds. |
| `SAV-DOC-053` | High | The whole top-level document `Serialize`, `R0414`, enumerated. The `Player` list is a `u32` count and that many `ar << CObject*`, and its first element begins at decoded offset 75 on 18 of 18 saves. This is what says the stream continues past record one, and what bounds FR-1's extent test. |
| `SAV-HERO-059` | High | `Player+0x34` names the human participant's own starting character. It is written at `L08350`, read at `L08351` and resolved through the identity map by `L08352`. It equals the identity key of exactly one actor in that participant's own groups on 18 of 18 distinct saves. This is the decoded field FR-3 orders the party by. |
| `SAV-GRPORD-058` | High | Nothing orders the actor list. The store arm of `R1341` is a head-to-tail `CObList` walk with no comparison, and one actor set appears in three orders across one session. This is why an ordering rule is needed at all, and why the file's own order carries no ranking. |
| `SAV-ID-015` | High on this corpus, Medium as a law | The runtime creation-order id, and the allocator behind it. It carries a Medium clause: a corpse decay frees a bitmap bit and the next spawn reuses the lowest free one, so a save with mid-session churn could carry no id 1 or two. It is demoted here from the ordering rule to its fallback (DD-2). |
| `SAV-OBJ-016` | High | The doubled slot id at `+0x04`/`+0x08`, and the untagged group record between sibling actors. Unchanged by this story and still the anchor the walk's `Player` programme is built on. |
| `SAV-DIARY-042` | High for the shape, Unknown for the meaning | The two runtime arrays of a `Diary`. What they hold is Unknown, which is why FR-1 reads the record and applies nothing from it. |

## What was retracted, and what this story does about it

`SAV-TOPLVL-052`'s **headline** is REFUTED by `SAV-PLDIARY-054`. The verbatim retracted text is
*"the stream does not chain above the first record: a walk reaches one `Player` subtree and a
terminator, and everything else needs a pattern scan"*, measured 23/23 and graded **High**. The
grade was earned on a walk that tiled and stopped at the same word on every file; the stop was the
record programme running out, not a frame in the file.

The rest of the row is untouched and is still cited where this tree cites it: the tag-scan error
rate, the two zero-run artefacts, and the `SAV-OBJ-016` anchor that rejects them.

Two further rows moved with the pin and are named here because a reader of the older documents will
meet them. `SAV-CELLREC-032` is SUPERSEDED in its "unattributed" label only — the four bytes are the
terrain object's identity key — and its `2 + 54 × count + 4` measurement is exact and unchanged, so
`pkg/formats/sav`'s world-half locator is unaffected. `PARTY-ROSTER-002` is SUPERSEDED in its
`+0x1c` clause only: `Group+0x1c` is an authored id for map-supplied groups and uninitialised memory
for runtime ones. This tree reads `G1C` and consumes nothing from it, so nothing changes.

## Ours by choice

- **The fallback ordering rule, and only the fallback (DD-2).** `SAV-HERO-059` decides the leader.
  Where a file resolves no starting character, or more than one, this tree keeps the runtime-id rule
  `0148` authored, and where that does not resolve either it keeps the file's order. Both fallbacks
  are authored and both are reported.
- **Reading the `Player`'s own `Diary` and applying nothing from it.** The record is read so the
  programme is complete. What its two arrays hold is Unknown (`SAV-DIARY-042`), and this tree has no
  journal.
- **Not extending the walk past the first `Player` (DD-4).** `SAV-DOC-053` reaches the whole stream
  on 7 of 18 saves and desynchronises inside a `Unit` subtree on the other 11. A consumer built on
  that would be right on 39 per cent of the corpus.

## Landed documents are not rewritten

`docs/0147-party-from-the-save/` and `docs/0148-the-resumed-mission/` are left exactly as they were
published, including the rows that cite the retracted headline as live High evidence. This follows
the owner's 2026-08-11 ruling that a published record stays as published and a correction is carried
by new work.

The one edit considered and rejected was a superseded marker on `0147/provenance.md`'s
`SAV-TOPLVL-052` row. A story's `provenance.md` is the evidence ledger for **that story** — the
evidence it stood on when it landed — and the live retraction state is owned by
`research/claims/retracted.md`, which `tools/claim` cross-reads on every lookup at about 1 KB. A
second copy of the retraction state in a per-story ledger would be a copy nobody greps and one more
place to go stale.

What is living, and is therefore corrected here: every documentation comment in `pkg/formats/sav`,
`pkg/game` and `cmd/savtool`, and every sentence the build prints to the owner. Those describe the
tree as it is now, not as it was reasoned about once.

## Open, and not answered here

- **Whether the whole stream is walkable.** `SAV-DOC-053` tiles 7 of 18 saves and desynchronises
  inside a `Unit` subtree on the other 11. Research carries this as a live thread.
- **What the actor list's order means.** `SAV-GRPORD-058` answers it as far as it can be answered:
  the order is the order of appends and is runtime state. No consumer should read position as
  identity.
- **What a `Diary`'s two arrays hold.** Unknown (`SAV-DIARY-042`).
- **What a save records of what the participant has seen.** Under active re-derivation in research
  at the time this story was written. No claim about it is cited here and no statement about it is
  made anywhere in this story's scope.
- **More than one human participant.** `SAV-HERO-059` grades this Unknown; every save in the corpus
  has one.
