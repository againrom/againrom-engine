# 0144 — plan

## Shape

One new leaf package, one new tool, one new entry point in the front-end tier, one flag on the
existing driver. No package outside this list is touched.

```
pkg/formats/sav        the format: container, codec, CLASS TABLE, object walk, edits  (leaf)
cmd/savtool            info / blocks / actors / session / verify / set
pkg/game/resume.go     ResumeMission: save + map -> a started mission + a report
cmd/missionrun         -sav FILE
```

`pkg/formats/sav` joins `internal/archtest`'s allow map with an **empty** set (P-1), beside the
other eight format leaves. `cmd/savtool` gets `{pkg/formats/sav}`. `pkg/game`'s row is already the
`pkg/` wildcard, so the resume adds no edge. **`cmd/missionrun`'s row does not change**: the
resume returns a started mission and a report, so the driver reads a file and calls `pkg/game`,
and never names the format tier. That is the reason the resume lives in `pkg/game` rather than in
the tool (**DD-1**).

## The other side of the seam, and what is left for the story that wires it

`0143` landed `pkg/game`'s own save while this branch was open, and its `Restore` states that it
names no file format and that a reader of some other format produces a `Snapshot` and reaches it.
That is this story's other side, so the boundary is worth stating rather than discovered later.

**This story does not reach `Snapshot`, deliberately.** It owns the original's format and the
mission driver; the window, the menu and the store are `0143`'s and stay untouched. Building the
join here would put a second producer of `Snapshot` in the tree before the format could fill one.

What a `.sav` can already fill: `Mission` from the head; `Gold` from the `Player` record's purse;
`World` from `ResumeOriginalSave`'s own `*Mission` through the same `MarshalBinary` the live path
uses. What it cannot: `Party`, because a member's statistics are inside `Unit`; and `Won`,
`Available` and `Taken`, which are campaign progression this tree has not located in the file at
all. `Residue` is front-end memory with an honest default — nothing explored, nothing commanded.

So the wiring story's cost is one function and two gaps, and both gaps are the same gap: `Unit`.

## Decisions

- **DD-1 — the resume belongs to `pkg/game`.** It needs the mission-to-map table, the archive, the
  definition table and the map decode, all of which that tier already owns, and it needs to modify
  the decoded map *before* the world is built. Putting it in the tool would either duplicate the
  start path or push four imports into a driver that has managed without them.
- **DD-2 — the body is bytes, and the parse is an index over them.** `File.Body` holds the whole
  decoded stream; the parse records offsets and lengths, not copies. An edit writes into `Body` at
  a recorded offset. This is what makes FR-8 and FR-9 the same mechanism rather than two: every
  byte not written is the source file's, by construction, not by a re-serializer remembering to
  carry it. A tree that rebuilt the body from a typed value would have to model every undecoded
  segment to get FR-8, and would be wrong the first time an extent varied.
- **DD-3 — `Marshal` always re-compresses.** Carrying the original blob would make FR-8 vacuous.
  Re-compressing makes the round trip a test of the encoder, which is the only test of the encoder
  available without a running original.
- **DD-4 — the world-half scan validates forward before it accepts.** A candidate is a u16 count at
  some offset whose records are strictly increasing in-window cells; it is accepted only when the
  cell-record table parses after it and its key set equals the block cells carrying static bit 5,
  and the session block fits after that. The key-set agreement is the discriminator; without it the
  scan is a plausibility argument and with it a wrong offset is caught by 180-odd independent
  equalities.
- **DD-5 — the scan starts after the campaign head and stops at the body end.** It cannot start at 0:
  the roster is full of small counts and a naive scan finds one. The first accepted candidate wins;
  a second is not looked for, because the validation makes a second implausible and looking would
  double the cost of the only expensive operation in the reader.
- **DD-11 — a class layout is a table the decoder is driven by** (FR-12). `class.go` holds one row
  per class: name, head, base class, members in file order, and how the length is determined. A
  member carries its own arithmetic, so the involution and the clamp cannot be forgotten by a second
  caller — and because they are members of `Player`'s row alone they cannot spread by accident
  either. `Player`, `Building` and `Effect` are decoded through it and by nothing else. The eleven
  read programmes became rows and the reader did not move, which is the test this decision was made
  to pass.
- **DD-13 — length is a three-way answer, not a number.** Fixed, computable, unread. Collapsing the
  last two into "unknown" would hide the only distinction that matters to a caller: a computable
  record is one this tree could walk with more code, an unread one is one no amount of code here can
  walk. That is exactly the line between "not done yet" and "blocked on research", and it is what
  `spec.md` L-1 is derived from rather than asserted alongside.
- **DD-14 — an unknown class name degrades to unread rather than being refused.** Seventeen of the
  twenty-eight classes have no programme, so an unread class is the ordinary case and a reader
  already handles one. Refusing an unknown NAME would be a second failure mode for the same
  situation, and it would fail on saves that already exist: one corpus file introduces `Spell`.
- **DD-15 — the instance tag is learned, not computed.** A class's index in the stream's shared
  counter depends on how many objects preceded it, which cannot be counted while most classes have
  no programme. Reading the tag once and then requiring it at every step is a stronger check than
  computing it: a wrong step length shows up immediately as a tag that does not match.
- **DD-12 — the state dword is CALIBRATED, never tabled.** Grouping candidates by that dword and
  accepting a group that behaves like a class run costs one pass and survives a value nobody has
  seen. A table would not have: this corpus holds three values no published set carries.
- **DD-6 — actor heads are located by their own agreement, not by the class tags.** `cellA == cellB`,
  no third equal word after them, and a state dword DD-12's calibration accepted. The scan reports the map unit id at `+0x13` for every head; a consumer joining to a
  map drops the ids the map does not carry, so a false positive costs a dropped row and not a wrong
  placement. This is the honest reading of L-2: the reader does not claim to have walked the stream.
- **DD-7 — bounds for the cell test come from the save, not from a map.** The scan accepts a cell
  whose row and column are each below 256 and whose packed value is inside the block plane's own
  sweep window where a world half exists. The package is a leaf and must not need a map to read a
  save.
- **DD-8 — the resume rewrites the decoded map's unit records.** The head's packing is the map
  record's own fixed point (FR-7), so the transfer is one assignment per matched unit and the
  existing start path does the rest: passability, placement, script compile and party drop all run
  exactly as they do for a fresh start. No new door into `pkg/sim` is opened and nothing hashed is
  reached — which is what keeps this story off that threshold.
- **DD-9 — the block deltas and the session latches are reported, not applied** (`spec.md` L-3). The
  report counts the records that disagree with the ingest's own plane, which is both the divergence
  disclosure and the measurement `SAV-BLOCK-012` wants.
- **DD-10 — `set` writes to a path the caller names and never in place.** The corpus is read-only
  preserved evidence; a tool that could default to overwriting its input is one command-line slip
  from destroying it.

## Work

1. **The codec and the container** — FR-1, FR-2. `Decompress`, `Compress`, `Open`, `Marshal`. Tests build a blob
   in test code from the contract: a literal-only stream, a run-only stream, a mixed one, the four
   opcode values the shipped encoder never emits (`0x00`, `0x7f`, `0x80`, `0x81` — all legal), an
   overrun, a short emit, a long emit, an odd-length input, and a header failing each of its four
   checks.
2. **The campaign head and the roster** — FR-3, FR-4, FR-5. Head decode with the computed offsets; `Player` records
   located from the class record and the `0x8001` tags, accepted on the four-way agreement, count
   checked against the head. The involution and the saturation on both paths.
3. **The world half** — FR-6. The validated scan, the three structures, the typed accessors — block record
   dyn/static, trigger latch, diplomacy entry, win and lose counters — and the derived shape.
4. **The actor heads** — FR-7 — and their scan, over the class table FR-12 defines. The head is 37
   bytes and one routine; the fields the corpus reading named are a prefix of it, inside a block
   copied out of another object, so the scan's offsets did not move.
5. **The edits** — FR-9, and FR-8 is the test of them. One method per editable field, each writing at a recorded offset; a test that an
   edit changes the intended bytes and nothing else, over a synthetic file built by the tests
   themselves.
6. **`cmd/savtool`** — FR-10 — with the six verbs, its argument parsing tested as pure functions.
7. **`pkg/game/resume.go`** and `missionrun -sav` — FR-11 — with the report.
8. **The `Building` walk and the re-encode** — FR-13 — plus the `objects` verb that runs both over a
   real file and reports the identity-key and creation-order checks.
8. Evidence over the real corpus into `verification.md`: the round-trip count, the labelled-set
   discrimination, the resume's own output, and the milestone census before and after.

## Risks

- **The encoder may not reproduce the shipped bytes.** Measured first, before anything was built on
  it: 16 of 16. Had it failed, FR-8 would have had to become "the body re-decodes identically"
  instead, which is a much weaker test — so this was the first thing checked and not the last.
- **The world-half scan may land on a false candidate.** DD-4's key-set agreement is the mitigation
  and it is checked on every file of the corpus in `verification.md`. A save with no world half must
  produce *no* candidate; the one such file in the corpus is what tests that arm.
- **A resumed mission may diverge from what the original would resume to.** It certainly does, and
  L-3 names how. The mitigation for now is disclosure and a counted report, not a claim of
  fidelity; the report's own numbers are the list that has to shrink when the member layouts land.
- **The calibrated scan is a heuristic and the walk is not.** It is the interim locator, and its
  known soft edge is that a file holding fewer objects than a class run yields none — which is what
  a between-mission save does. The walk retires it one class at a time, and `Building` is the first;
  the rest wait on the seventeen unread classes.
- **`Effect` has a fixed length and still does not chain.** Its instances are evidently not written
  consecutively, but this tree cannot separate that from an extent it has wrong using the corpus
  alone. It is recorded as an open question rather than worked around, and `Building` — which does
  chain, 18/18 and 30/30 — is what the walk is claimed on.
