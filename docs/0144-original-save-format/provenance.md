# 0144 — provenance

## The research pin, and a bump inside this story

**The submodule was bumped mid-story TWICE: `20dcf51` to `e2516a3`, then `e2516a3` to `445202e`.**
The first was authorised as the only one; the second is the second, and it happened because this
seat opened a further research round (`EXP-0146`) and named the hash, not because this lane moved
anything on its own initiative. Both bumps are on the same owner ruling of 2026-08-12 — a story
that needs research data waits for it and lands without holes. SDD S-5 freezes the pin mid-story; this story is the case that rule does not cover, because it
needed two research results produced during its own life. `EXP-0145` read eleven `Serialize` bodies
and the image's class descriptors; `EXP-0146` resolved the eight embedded objects those bodies
serialize and corrected one of them.

**A fact this tree held was retracted by the second round, and it was load-bearing.**
`SAV-MEMBER-036` said a `Human` record is a `Unit` record plus 24 raw bytes. `SAV-HUMAN-043` shows
`Humanoid::Serialize` is a twelfth body which writes those 24 bytes **and thirteen object
references**. The old reading is not merely incomplete: a walk built on it desynchronises eight
bytes into the first `Human` of a save and never recovers, and a writer that believed it emits a
record the game's own loader reads twenty-six bytes short. This package carried the old reading in
`Human`'s row and no longer does; `TestHumanIsHumanoidAndHumanoidIsWhereTheBytesAre` is what stops
it coming back, and it refuses the value 24 by name.

Rows below citing `SAV-…-033` upward are from the pins bumped to; the rest were true at `20dcf51`
and are unchanged except where a row says otherwise.

## What the contract rests on

| Contract | Claim | Confidence | Note |
|---|---|---|---|
| FR-1 header, blob extent, the reader's own checks | `SAV-HDR-001`, `SAV-VER-002`, `SAV-FRAME-021` | High | `blobEnd` is patched in after the write, which is why this tree recomputes it |
| FR-1 label region and the embedded state store | `SAV-PTR-003`, `SAV-EMB-004` | High / Medium | the tail is carried verbatim; its inner `&YA1` is not opened here |
| FR-2 the run/literal word codec, both directions | `SAV-PACK-007`, `SAV-EXT-009`, `SAV-CODEC-022` | High | the encoder rule is read off `rom.exe`, not inferred from output |
| FR-3 the campaign head, field for field | `SAV-HEAD-025`, `SAV-MAP-005`, `SAV-STREAM-010` | High | the eleven dwords' store order is not ascending; this tree carries them as a block |
| FR-3 mission number 0 with a stale map name | `SAV-CITY-030` | High | the only reason the spec forbids reading the name to decide |
| FR-4 the seventeen-field record and its two obfuscated dwords | `SAV-PLAYER-028`, `SAV-OBF-029` | High | the saturation at 32767 is the same claim |
| FR-4 class introduction and the shared index counter | `SAV-STREAM-010`, `SAV-STREAM-013` | High | `Player` is first-use-first, so its index is 1 in every corpus file |
| FR-5 the outcome latch, and what is not the latch | `SAV-FLAG-027` | High | the win counter lives in the world half and is gone when it is needed |
| FR-6 the block-plane record array and its packing | `SAV-BLOCK-011`, `TERR-PASS-053` | High | |
| FR-6 the array is a delta over the ingest | `SAV-BLOCK-012` | Medium | whether a load re-runs the ingest is argued, not read — which is why this tree reports the delta rather than applying it (`spec.md` L-3) |
| FR-6 the 54-byte cell-record table and its key law | `SAV-CELLREC-017`, `SAV-CELLREC-032` | High | the key law is what turns the scan into a location |
| FR-6 the 4374-byte session block, field by field | `SAV-SESS-031` | High | supersedes the 4382 an earlier round measured |
| FR-6 / L-4 the shape byte and where it sits | `SAV-SHAPE-023` | High | in the stream after the roster, which is why L-4 derives it |
| FR-4 / L-2 the roster is three levels and is in both shapes | `SAV-ROSTER-024`, `PARTY-ROSTER-002` | High | the group record's actor list is what makes a sequential walk unavailable |
| FR-7 the head is 37 bytes and one routine | `SAV-TOKEN-034` | High | amends `SAV-OBJ-014`, whose field positions were right and whose extent was a prefix |
| FR-7 the packing, the fine bytes, the state word | `SAV-OBJ-014` | High / Medium (fine bytes) | `cellA == cellB` in 413/413 is the scan's own discriminator; the block they live in belongs to an object nobody has identified |
| FR-7 identity keys and cross-object references | `SAV-PTRMAP-035` | High | what a writer must mint, and the silent null on an absent key |
| FR-12 the eleven programmes, field by field | `SAV-MEMBER-036` | High per sequence | and its own Unknown: no total length for `Unit`, `Human`, `Diary` or a group record |
| FR-12 twenty-eight classes, not eleven | `SAV-CLASS-033` | High | amends `SAV-STREAM-010`, which was true of four files |
| FR-12 the obfuscation and the clamp are `Player`'s alone | `SAV-OBFCEN-038` | **Medium**, and cited at that grade | blind to a field already held transformed in memory — which a corpus cannot see either |
| FR-13 a `Building` record is 77 bytes, chained and terminated | `SAV-BLDG-037` | High | two independent derivations meeting exactly, at two map populations |
| FR-12 the eight embedded objects, and the class at each | `SAV-EMBED-039` | High | `Group+0x20` is an eighth site the earlier claim did not carry, and it is first in a group record |
| FR-12 `Humanoid` is a twelfth body writing 24 bytes and thirteen references | `SAV-HUMAN-043` | High (Medium for reading the twelve as equipment) | **retracts** `SAV-MEMBER-036`'s `Human` clause, which this tree had built on |
| FR-12 a `Unit` has NO length; floor `609 + L` | `SAV-UNITLEN-045` | High (Medium for `609+L` reproducing 621) | four counted lists, a `CString`, three references, two presence flags |
| FR-13 an `Effect` is 44 bytes and is a LIST ELEMENT, not consecutive | `SAV-EFFCHAIN-046` | High | closes this tree's own open question; reading #2 of the three it enumerated |
| FR-7 runtime id, and 0 meaning dead | `SAV-ID-015`, `SAV-OBJ-016` | High | |
| FR-7 / L-2 member extents, before the programmes | `SAV-OBJ-016` | High | superseded for `Building` (77 confirmed) and contradicted for the pair `Human 528 < Unit 621`, which `SAV-MEMBER-036` makes impossible |
| FR-11 the map is re-opened by name when the mission number is non-zero | `SAV-HEAD-025` | High | this tree opens it through its own mission-to-map table instead, and the two agree on the corpus |

30 of the 30 active `sav` claims were cited nowhere in this tree before this story.

**Nothing about the format now blocks this story.** All eight embedded sites are resolved and every
programme a `.sav` needs is published. What is owed is *implementation*, not research, and it is
named in `verification.md` under what is not verified.

## Ours by choice, not by evidence

- **Locating by validated scan.** The research spec locates these structures; it does not say how a
  consumer should. The scan and its acceptance test are this tree's, and the acceptance test is
  built out of `SAV-CELLREC-032`'s key law precisely so that it can fail loudly.
- **Recomputing `blobEnd` and `blobBytes` on write** rather than carrying them. The original patches
  them after the fact; a writer that carried them would be right only by accident.
- **Reproducing the 32767 saturation.** `spec.md` L-5.
- **Reporting the block deltas and the session latches rather than applying them.** `spec.md` L-3.
  Applying them needs a door into state this tree hashes, and that is a different contract.

## Observations of this consumer, not published facts

These are things this tree measured while exercising the decode. **None of them is published as a
claim here** — implementation does not author facts about the original. They are recorded so they
can be routed to research.

- **The encoder rule reproduces the shipped bytes exactly.** Re-compressing the decoded body with
  the greedy rule of `SAV-CODEC-022` yields the original blob byte for byte on **23 of 23** files —
  the four at each install root and the fifteen owner-produced ones, re-measured at the landing over
  a corpus three files larger than the one this bullet was written on. The published claim states the
  rule; this is the first time the rule has been run backwards over a whole corpus, and it says the
  encoder has no tie-breaking freedom left unstated.
- **The label region is 8-bit text, not ASCII.** One owner-typed label in the corpus begins with the
  byte `0xE0`, which is a Cyrillic letter in the game's own single-byte page. The format spec calls
  the region a string and does not say what encodes it.
- **The actor head's packing is the `.alm` unit record's own fixed point.** `SAV-OBJ-014` gives the
  head's cell as `(row << 8) | col` and the fine bytes as a separate pair; the map's unit record
  stores position as a 24.8 fixed-point dword. The two are the same value: cell in the high byte,
  fine in the low, so a saved position transfers into a map unit record with no arithmetic. The
  spec relates the head's cell to the map record's cell but not its fine bytes to the map record's
  low byte.
- **The state dword's published value set is not closed.** `SAV-OBJ-014` names three values
  observed at `+0x08` over four saves. The twelve owner-produced saves hold **three more** —
  `0x0612a020` on 34 objects of one 10.alm session, `0x0256d020` on 57 and `0x02563020` on 40 of a
  20.alm session — and the shipped restart save holds only `0x06735020`. Every value seen, old and
  new, has the low twelve bits `0x020`, which is the shape of a heap address. The claim's own
  reading, that this is state and not format, is confirmed rather than contradicted; what is new is
  that a consumer must not enumerate the values. This tree calibrates on them instead.

  > **Not re-measured at the landing, unlike the bullets around it.** No `savtool` verb prints the
  > raw dword — it prints the classification read off it — so the three saves preserved since have
  > not been scanned for new values. The count *"three more"* therefore reaches twelve files, not
  > fifteen. This is stated rather than quietly extended, because the whole point of the bullet is
  > that the set is open: a corpus that grew without being re-scanned is exactly the condition under
  > which someone would conclude it had closed.
- **A between-mission save carries the outcome latch of the mission just finished.** The owner
  labelled two saves as finished; a third file, the one taken in the city with no world half,
  carries the same latch value 1 and the same purse as the finished 20.alm save. That is
  `SAV-FLAG-027`'s stated *reason* — the latch is in the campaign half precisely so it survives the
  mission's end — observed for the first time on a file where the mission is over.
- **Two saves introduce `Effect` before `Item`.** Class first-use order differs file to file, as the
  published framing says; over the fifteen owner saves the orders fall into **nine** shapes, and two
  place `Effect` ahead of `Item`: `game0007.sav` at positions 6 and 7, and the later `game9999.sav`
  at 4 and 6 — the second separating them by two positions and introducing `Spell` in the gap.

  > Re-measured at the landing. This bullet read *five shapes over twelve saves, and `game0007`
  > alone*. Both halves were wrong: the twelve already held seven shapes, and the second
  > `Effect`-before-`Item` file was preserved after the bullet was written. **The correction makes
  > the thread stronger, not weaker** — a lone file is as easily a damaged file as a signal, and a
  > second one taken from a different session is the control the first did not have.

  This is recorded because `EXP-0146` reports that on `game0007` its programme walk and its tag
  scan agree on only 12 of 50 objects without establishing which is wrong, and an `Effect` is an
  element of an `Item`'s own list — so a file where the first `Effect` precedes any `Item` is where
  an assumption about which object owns one would come apart first. **It is a hypothesis about
  where to look, not a claim about the format**, and this tree's own instruments find `game0007`
  unremarkable on every measure they make: 30 buildings chained, 30 distinct identity keys,
  creation order 2..62, all 30 re-encoding identically, and the world half locating with its
  cell-record key-set agreement intact — the same as its three 20.alm neighbours.
- **The head's two counters have a fixed ratio outside the restart save.** The first two dwords of
  the body satisfy `second == first >> 4` on the corpus, which is the published relation, and the
  first is 0 on the one save with no world half. Consistent with the spec; recorded because it is
  now measured on **23 files rather than 4**, the three latest of which the decoder had never seen
  when the relation was written — `152/9`, `489/30` and `1/0`, and it holds on all three.

## What a green round trip cannot witness

`SAV-HUMAN-043` retracted `SAV-MEMBER-036`'s "a `Human` is a `Unit` plus 24 bytes" **after** this
tree had built on it. **The round trip stayed green across the correction, and that is not evidence
the table was ever right.** Everything past a head is carried opaque here, so a byte-carrying reader
reproduces the file whether the row is right or wrong; the row was wrong and only the opacity kept
it harmless.

The load-bearing part for whoever comes next is that **the opacity expires**. A reader that decodes
`Unit` stops carrying those bytes, and the same wrong row that costs nothing today desynchronises
eight bytes into the first `Human` of every save. The round trip is this story's central evidence
and it is strong — about the container and the codec. It says nothing whatsoever about a layout it
never decodes, and any future round that reads it as *"the table is correct"* has misread the
instrument.

## Open

- The class member bodies. `spec.md` L-1 is the whole consequence, and a research lane is opening
  that question in parallel; nothing here pre-empts it.
- The `&YA1` state store in the tail. Carried verbatim, never opened. Its keys are named by
  `SAV-EMB-004` and its values are the registry format's business.
- Whether a load re-runs the terrain ingest before applying the block records (`SAV-BLOCK-012`).
  This tree's resume ingests and then reports the disagreement, which is the measurement that
  question wants and is recorded in `verification.md`.
