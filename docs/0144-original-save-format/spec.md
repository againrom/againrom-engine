# 0144 — the original game's save file

The contract for reading, re-emitting and editing a `game####.sav` written by the original game,
and for resuming a mission from one. Self-contained: the research claim ids behind every fact
below are in `provenance.md` and are not needed to implement this.

This story owns the **file format** and nothing that draws. Our own save format, and every screen
and menu that would reach one, belong to a different contract.

## Why

Three things are wanted of save/load: our own save, reading the original's, and writing one the
original can load. The second and third are one contract — a writer is only trustworthy if it is
the reader run backwards over the same bytes — and neither is reachable from our own format. The
format is already decoded and published; nothing in this tree consumes it, so the decode has never
been exercised by a consumer, and a decode no consumer has run is a decode with untested edges.

## Scope

In: the container, the transport codec, the campaign half's head and its `Player` records, the
world half's block-plane records, cell-record table and session block, the label region, the
uncompressed tail, and a runnable resume of a real mission from a real save.

Out: anything that draws; our own byte-form (`pkg/sim`'s serialization is untouched).

**Two research rounds have landed and this contract has absorbed both.** Every programme a `.sav`
needs is now published: the eight embedded sites are resolved, and `Unit` is established to have
**no constant length at all** — that is the answer, not a missing measurement, because its
programme holds four counted lists, a `CString`, three object references and two presence flags.

So **nothing about the format blocks this story any more**, and what L-1 and L-3 now record is
implementation that is not built rather than knowledge that is missing. The distinction matters
because it changes who closes them. `Building` is decoded, walked and re-emitted from its decoded
form, which is what demonstrates FR-12's table is a mechanism and not a promise; the remaining
classes need two member kinds and a stream walker that this contract names and this revision does
not yet ship.

## Functional requirements

### FR-1 — the container

`Open` takes the whole file and answers a value or an error. It requires magic `"Asg&"` at 0, a
version dword at `+0x08` **at least** `0x0BAD0002`, `20 <= blobEnd <= len` and
`blobBytes == blobEnd - 16`; each failure names which. The blob is `[0x10, blobEnd)`, the label
region is `[blobEnd, blobEnd+0x100)`, the tail is `[blobEnd+0x100, EOF)`.

`Marshal` answers a whole file: header, the body re-compressed by FR-2, the label region, the
tail. `blobEnd` and `blobBytes` are **recomputed**, never carried; the version and the magic are
written as read.

### FR-2 — the transport is a run/literal code over 16-bit **words**

`Decompress` reads the blob's leading `u32 outWords`, then opcodes from `blob+4`: `n < 0x80` is
`n` literal words (`2n` bytes copied); `n >= 0x80` is the next word emitted `n & 0x7f` times. The
loop is bounded by the **source** span, never by the output count, and must then have emitted
exactly `outWords` words — a decode that ends short, long, or overruns is an error naming which.

`Compress` emits a run wherever `word[i] == word[i+1]` and a literal otherwise, greedily, run
length capped at `0x7f` and literal length at `126`; it pads an odd-length input to an even byte
length and writes `outWords` first.

The repeated unit is a word. A byte-wise reading keeps the length and corrupts the content, so it
is not a variant to fall back to.

### FR-3 — the campaign head

Decoded in one straight run from body offset 0: two counters, the map name as a length-prefixed
string, eleven dwords, the mission number, the difficulty, one dword whose meaning is not ours,
and the player count. Offsets after the map name shift with its length and are computed, never
tabled.

**The mission number, not the map name, says whether a mission is in progress.** A save taken
between missions carries mission number 0 while the map-name field still holds the previous
mission's name; a consumer that reads the name to decide is wrong on exactly that file.

### FR-4 — the `Player` records

The first class the object stream introduces is `Player`, so its class index is 1, its first
instance follows its class record immediately, and every later instance is introduced by the u16
tag `0x8001`. A record is seventeen fields in one straight run: a length-prefixed name, the
1-based slot id twice (u16 then u32), eight undecoded bytes, and then the fields FR-5 names.

**Two traps.** Money and the dword beside it are stored **XOR `0x5c073f4d`** — an involution, so
the same operation applies in both directions and a reader that skips it sees about 1.54 billion
gold. Two further fields are silently saturated at 32767 by the original's writer; this tree
saturates them the same way on write and says so rather than round-tripping a larger value it
knows will not survive.

A record is accepted only where the u16 slot id and the u32 slot id agree, the slot is 1..16, the
human-participant flag is 0 or 1 and the outcome latch is 0, 1 or 2. The count of accepted records
must equal FR-3's player count or `Open` fails.

### FR-5 — the mission-outcome latch and the money

The outcome latch is one byte of the `Player` record and therefore lives in the **campaign half**
and is present in **every** save, including one taken between missions: 0 in progress, 1 complete,
2 failed. The win counter in the session block is not the latch and must not be read as one — it
is gone the moment a mission ends, which is when a campaign needs the answer.

Money is a dword of the same record, after the involution.

### FR-6 — the world half

A save carries the world half only sometimes; the file says which. Where it is present this tree
decodes three structures and locates them by **anchored scan with forward validation**, because
the class member bodies are not decoded and a sequential walk of the object stream is therefore
not available (**L-2**):

- **the block-plane record array** — a u16 count then that many dwords `(cell << 16) | (dyn << 8) |
  static`, cells strictly increasing and inside the serializer's sweep window;
- **the cell-record table**, immediately after — a u16 count then that many 54-byte records each
  beginning with a packed cell key;
- **the session block**, four unattributed bytes after that — 4374 bytes holding 100 trigger
  result slots, 1000 fire-once trigger latches, a 50x50 diplomacy matrix, and the win and lose
  counters.

A candidate location is accepted only if all three parse inside the body **and** the cell-record
table's key set is exactly the set of block cells whose `static` byte carries bit 5. That
agreement is 185 to 197 keys wide on this corpus and is what makes the scan a location rather than
a guess. The first accepted candidate is the answer; where none is accepted, the save has no world
half.

**The block array is a delta over the map's own terrain, not a plane.** It omits every cell whose
only block bits are the map's own. A consumer must ingest the `.alm` first and apply these records
over what the ingest derived; applying them to a zero plane yields a map with no terrain blocking
at all.

### FR-7 — the actor heads

Every placeable class begins with the same head, and it is **one routine and 37 bytes**. Its first
twelve bytes are a raw copy out of a *separate object the token points at*, and the cell packed
`(row << 8) | col`, the same cell again, the two fine-position bytes, a word and the state dword all
live inside that block — so those positions are unchanged and their owner is not the token. Past the
block: the runtime id, a byte, a word, a dword whose low half is the map's own unit id, a word, a
dword, **the object's own address as an identity key**, and **another object's address as a
reference**. The last two are one mechanism: the load arm binds old-address to new-object as it
reads the first and resolves the second through that same map, or to null, silently, when the key is
absent. This tree locates heads
by scanning for the two agreeing cell words and then **calibrating on the file's own state
dwords**: candidates are grouped by that dword and a group is taken for a class run only where it
has enough members and its runtime ids are small and, where non-zero, all different. The state
dword's value set is *not closed* — it is state, not format — so a tabled set of values would be
wrong on the next save anyone makes. The scan exposes cell, fine position, runtime id and map unit
id. A head whose runtime
id is 0 is a **dead** object and is reported as such.

The head's packing is the same fixed-point the `.alm` unit record uses: cell in the high byte,
fine position in the low. So an actor's saved position transfers to a map unit record as one value
and needs no conversion.

### FR-8 — round trip, byte for byte

`Marshal(Open(b)) == b` for every file in the corpus, with the body re-compressed rather than the
original blob carried. This is the story's central evidence: if every byte that was not decoded
comes back where it was, the decoded parts are the only parts that can be wrong.

### FR-9 — read, modify, write

Editing is **in place into the decoded body**, never a re-serialization: a change writes the bytes
of the one field it names and every other byte of the body, the label region and the tail is the
source file's own. The editable set is exactly what FR-3 to FR-7 decode — an actor's cell and fine
position, money, the outcome latch, a block-plane record's two bytes, a trigger latch, a
diplomacy entry, the win and lose counters, and the label.

The label region is a NUL-terminated string in a fixed 0x100 buffer that the original **overwrites
in place and never clears**, so a label is followed by the tail of whatever was longer before it.
This tree reads to the first NUL and, on write, does the same: it writes the new bytes and the NUL
and leaves the remainder of the region exactly as the source file had it. Label text is 8-bit and
is not ASCII — one owner-typed label in the corpus carries a Cyrillic byte — so it is handled as
bytes.

### FR-10 — a tool

`cmd/savtool`, beside the other thirteen, with verbs over a file named on the command line:
`info` (container, head, players, world-half extents, label), `blocks` (the delta records),
`actors` (the heads), `session` (latches, diplomacy, counters), `verify` (round trip, reporting
identical or the first differing offset) and `set` (FR-9's edits, writing to a path the caller
names). It reads game files and writes only where told.

### FR-12 — a class layout is a table, not a hole

Every class the object stream introduces is a row of one table: its name, whether its record opens
with the placeable head, its members **in file order**, and its extent. A member carries its own
arithmetic — the involution and the saturation are properties of the field and not of the caller —
so a decode and an edit go through the same row.

A derived class's programme is its base's followed by its own, so "a `Human` record is a `Unit`
record plus 24 bytes" is a row and not a comment somebody keeps true.

**The obfuscation and the clamp are `Player`'s alone** and the table must not let them spread: over
the eleven read bodies and every sub-serializer they call, both occur only inside
`Player::Serialize`. That census is graded *Medium* by the round that made it and names its own
blindness — it read the writers of each record, not of each field, so it cannot see a value already
held transformed in memory.

Length is one of three things, and the distinction is the contract's, not an implementation note:
**fixed** (a constant, so the record can be stepped over without decoding); **computable** (the
programme is complete but the length depends on a counted list, a string or a presence flag, so the
record can be walked but not tabled); **unread** (the programme names an embedded object of a class
nobody has read, so the record cannot be walked at all). A class this tree has no row for degrades
to *unread* rather than being refused: twenty-eight classes exist, eleven have been read, and
meeting an unknown one is the ordinary case.

### FR-13 — walk one class, and re-emit it from its decoded form

Where a class's record is a fixed length and its instances are written consecutively, this tree
**walks** them: it finds the class introduction, steps the record length, learns the instance tag
from the first step and then requires it, and stops on the null object reference that terminates the
run. `Building` is that class.

Each walked record is then **re-encoded from its decoded values** and compared to the bytes it was
read from. This is the check FR-8 cannot make: a byte-carrying round trip proves nothing was
disturbed, and only a re-encode proves what was read was *understood* — a field read at the wrong
width or into the wrong variable survives the first and fails the second.

### FR-11 — a resumed mission

`ResumeMission` takes a save's bytes and answers a started mission plus a report of what was
carried across. It reads the save, opens the map the save's mission number names by the tree's own
existing mission-to-map table, applies each matched actor's saved cell and fine position to that
map's unit record, and starts the mission through the same path a fresh start uses. `missionrun`
gains `-sav FILE`, which resumes instead of starting and then drives exactly as it already does.

The report carries: the mission and map, the save's label and outcome latch, how many actor heads
were located, how many joined a map unit id, how many were dead, and how many of the block-plane
delta records disagree with what the map's own ingest derived.

## Acceptance

- **AC-1** Every file in the corpus opens, and `Marshal` reproduces it byte for byte.
- **AC-2** The two saves the owner labelled as finished carry outcome latch 1. The labelled set is
  external evidence and this is the only place it can be spent. Any *other* file carrying 1 must be
  explained rather than explained away.
- **AC-3** The one save with no world half opens, yields its whole roster, reports mission number
  0 with a map name still set, and offers no block records, no cell records and no session block.
- **AC-4** A save whose money is set to a value and re-emitted reads that value back, and every
  other byte of the file is unchanged.
- **AC-5** A resumed mission starts, places at least one actor at a cell the map alone does not
  place it at, and can be driven for ticks without error. **And a resume from a save the game wrote
  at the very start of a mission drives identically to a fresh start of that mission**, because
  such a save says nothing a fresh start does not — a resume that diverged there would be a
  transfer that had invented something.
- **AC-6** A truncated file, a bad magic, an older version, a blob that decodes short and a
  block-array candidate that fails the cell-record agreement each produce a named error and no
  panic.
- **AC-7** Every `Building` instance of every corpus save walks, and every one re-encodes from its
  decoded form to the bytes it was read from. Their identity keys are distinct within a save and
  their creation-order ids form a run.
- **AC-8** A save introducing a class this tree has no programme for still opens, still round-trips,
  and reports that class by name. The one corpus save that introduces `Spell` is the witness.

## Properties

- **P-1** `pkg/formats/sav` is a leaf: standard library only, no import of any package of this
  tree. Its tests build every fixture in test code from the contract above and read no install.
- **P-2** Nothing here changes what any existing test asserts about our own byte-form.
- **P-3** No `.sav` and no byte of one enters the repository. Evidence is counts, offsets and
  digests recorded in `verification.md`.
- **P-4** Decode is total over the corpus or it is an error: there is no arm that returns a
  partial value and a nil error.

## Authored, and on what grounds

- **L-1 — a save this tree writes is still one whose lineage began at a save the game wrote, and
  that is now an UNBUILT capability rather than an unknown one.** Every programme needed to author
  a record from nothing is published. What this tree does not yet have is the two member kinds an
  object reference and a counted list need, and a stream walker that carries the archive's shared
  index counter so that a reference can name an object by the index it took. `Unit` must be written
  by its programme and never by a length: it has none, and its floor of `609 + L` is a floor and
  not a size. When those exist, a writer mints a unique non-zero key per object and writes every
  reference as its target's key — and a reference to a key no object claims becomes **null
  silently**, which is the first thing such a writer must be tested against rather than trusted
  about.
- **L-2 — while extents vary with content, object bodies are located and not walked.** A
  sequential walk of the object stream needs every class's extent, and four of the eleven have none
  yet, so every structure past the roster is found by the validated scan FR-6 and FR-7 describe.
  The validation is what carries the weight, and where it fails this tree reports no world half
  rather than a guessed one. Two consequences worth stating: the state dword FR-7's scan calibrates
  on is **state and not format**, so its value set is learned from the file rather than tabled — a
  table would have gone stale on this very corpus; and a save holding fewer than a class run's
  worth of objects yields **no** located heads, which is what a between-mission save does. The walk
  FR-12's table is built for replaces all of this.
- **L-3 — for now, a resumed mission is a resumed *position*, not a resumed *state*.** Statistics, health
  and mana pools, inventories, timers, group orders and script register state are not decoded, so
  they come from the map's own placement rather than from the file. A resumed actor therefore
  stands where it stood and is otherwise as the map places it — at its placed health, not its
  saved health. The block-plane deltas and the session block's latches are decoded but are
  **reported, not applied**: this tree's world is built by its own ingest and has no door that
  takes a foreign block plane or a foreign latch set, and inventing one would reach state that is
  hashed. The report is what makes the divergence measurable instead of silent. Like L-1 this is
  where the story stands part way through: the statistics and pools a resume does not carry are
  exactly what the member layouts carry, and the report is the list of what has to shrink.
- **L-4 — the shape is derived, not read.** The byte that says whether a world half follows sits
  after the roster, whose extent L-2 makes unavailable, so this tree answers "has a world half"
  from whether FR-6's validated location was found. The two agree on every file of the corpus, and
  the derived answer is the one a consumer can act on.
- **L-5 — saturation is reproduced, not corrected.** The original silently clamps two `Player`
  fields at 32767 on the way out. This tree clamps them the same way: a writer that preserved a
  larger value would write a file that says something the original cannot mean.
