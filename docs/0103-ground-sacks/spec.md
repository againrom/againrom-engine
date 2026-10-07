# Spec — the map's authored loot reaches the simulation

**Intensity:** spec-anchored / static. **Terrain:** greenfield in three tiers, brownfield in one —
the script's check table ships and gains an arm. **Threshold: High** — it reaches hashed simulation
state and bumps the byte form.

## Terms

- **The loot section** is the `.alm` record whose typeId is **8**. This build stores its payload raw
  today and decodes none of it.
- **A loot record** is one entry of that section. **A ground record** is one whose owner word is 0;
  **a stock record** is one whose owner word is not 0, and it stocks an actor that already exists
  rather than putting anything on the ground.
- **An element** is one 10-byte item entry inside a record. **An item code** is the low 16 bits of
  an element's leading word.
- **A ground sack** is the simulation object a ground record makes: a cell, a purse and a list of
  item codes. It is not an entity — it does not tick, takes no order, blocks nothing and is in no
  entity list.
- **The sack query** is script check opcode **14**, a cell-to-sack existence test. This build
  reports it as unimplemented today.
- **The cell** of a loot record is each axis word, read as a signed 32-bit value, shifted right by 8.
- **Both roots** are the two lawful installs. A measurement made on one is not made.

## Why

Thirty of thirty-eight maps on one root carry a loot section, and between them the sections author
**137 ground sacks** and **43 stock records**. None of it exists in this build. The map's own script
already asks about it: **28 check-14 nodes ship on eleven maps**, reached by 33 trigger condition
slots, and every one of them is inert here because the check has no arm. A mission that gates a
trigger on "the sack is still there" cannot advance.

It matters beyond the 28 nodes, because the loot section is the only authored source of loot in a
shipped single-player map. The other things that make a sack are a death, a drop, a script command
and a random scatter, and the scatter is multiplayer-only at both of its call sites — so a campaign
map that does not read its own type-8 section has **no** loot at all.

**This contract draws nothing and picks nothing up.** A placed sack is invisible and unreachable;
what changes for a player is that a script which asks about one gets the true answer.

## Scope

**In scope.** The type-8 grammar in the format leaf, both head widths. The item code's class and
index. A dump subcommand for the corpus. Ground sacks as simulation state, the one-per-cell merge,
the byte form and the digest. Placement from the map. The sack query.

**Out of scope.** Picking a sack up, and everything it needs — an actor's container, a player's
purse, the order-to-state relay. Drawing a sack. Stocking an actor from a stock record. Resolving
an item code to a definition row. What the type-9 list is and what an element's link into it means.
The random scatter and the mission description file's own item list, neither of which reaches a
shipped campaign map.

## The contract

### The format leaf

**FR-1** `(*Map).Loot() (Loot, error)` decodes the type-8 payload. **The payload carries no count
word**: the number of records is the type-0 metadata word at `+0x2c`, which this reader already
exposes. The decoder reads that many records and reads nothing else.

**FR-2** A record is a head followed by `n` elements of 10 bytes, `n` being the head's own leading
word. The head is **20 bytes** when the file's format version is **989 or above** and **16 bytes**
below it; the fifth head word is the purse and is **0** on the short head.

| Head word | Meaning |
|---|---|
| `+0x00 u32` | `n`, the element count |
| `+0x04 u32` | the owner — **0 puts the record on the ground** |
| `+0x08 u32` | X, fixed point |
| `+0x0c u32` | Y, fixed point |
| `+0x10 u32` | gold — present only on the 20-byte head |

| Element word | Meaning |
|---|---|
| `+0x00 u32` | carried whole; the **item code** is its low 16 bits |
| `+0x04 u16` | carried whole; the original reads it only on the stock arm |
| `+0x06 u32` | a **1-based** index into the type-9 list; **0 names none** |

**FR-3** The walk **consumes the payload exactly**. A record that would run past the payload, and a
payload with bytes left over, are each an error with no partial result — never a truncated record
list and never a panic.

**FR-4** `Loot()` reads the raw type-8 body, the metadata count and the format version, and **stores
nothing on the map**. A map that is opened and written back is byte-identical whether or not
`Loot()` was called, exactly as the script decoder already leaves it.

**FR-5** An absent or empty type-8 record yields an empty result and **no error**.

**FR-6** The item code's fields are exposed as two functions over a `uint16`: **class** is bits
8..11; **index** is bits 0..4, except when the class is **14**, where it is bits 0..7. **A class of
0 or 15 names no item** and this build says so rather than inventing one. The code's other bits —
12..15 and 5..7 — are carried by the code itself and are **not** given a meaning here.

**FR-7** A record's cell is each axis word read as a signed 32-bit value and shifted right by 8.
This build **does not** truncate the result to a byte.

**FR-8** `almtool loot FILE` prints the section: per record its kind, cell, gold and elements with
each code's class and index, and a closing census — records by kind, elements, total gold, and
whether the payload closed exactly.

### The simulation

**FR-9** A world holds a **ground-sack list**. Each entry is a cell, a `uint32` purse and a list of
`uint16` item codes. The codes are **carried whole and interpreted nowhere** in this tier.

**FR-10** **One sack per cell.** The constructor folds entries naming the same cell into one: the
purses **add**, wrapping at the 32-bit width as the original's own addition does, and the item lists
**join** in argument order. This is the original's rule — its sack maker looks the cell up first and
pours into what is already there rather than building a second.

**FR-11** The list a world holds is **ascending by (Y, X)**, so a world's contents depend on the
sacks named and not on the order they were named in. An entry whose cell is outside the world's
bounds is **refused** — not clamped, not wrapped, and not dropped.

**FR-12** A sack list reaches a world through **one new constructor**, taking everything the widest
existing one takes plus the sacks. The four existing constructors keep their signatures and build
worlds with no sacks. Every constructor still funnels through the single body, so no rule about
what a world may hold can differ by the way it was built.

**FR-13** `(*World).Sacks()` hands back a **copy**, in the world's own order, with each item list
copied too.

**FR-14** The byte form becomes version **22** and carries the sacks as a **counted section between
the group section and the script section**: a `uint32` count, then per sack `int32` X, `int32` Y,
`uint32` gold, `uint32` item count, and that many `uint16` codes. The digest is taken over the byte
form, so the sacks enter it by construction.

**FR-15** The decoder **refuses** — rather than repairing — a truncated section, a sack out of
bounds, two sacks on one cell, and a list that is not ascending. The constructor produces none of
those, so nothing it builds fails to read back.

### The sack query

**FR-16** This build **evaluates** check opcode 14. Its register becomes **1** when a ground sack
occupies the named cell and **0** when none does. **Nothing of the sack reaches the register** — not
its gold, not its contents, not how many records were merged into it. The original discards the
pointer it found, so no script can read a sack.

**FR-17** The named cell is the **low byte of each of the check's first two plain parameters**, X
then Y. No shipped node exercises the truncation; it is the check's own arithmetic and is
implemented as such.

**FR-18** A cell outside the world's bounds answers **0**. It is not an error and it does not stop
the pass.

**FR-19** The compile-time report of checks this build cannot evaluate **no longer names 14**, and it
does so because the report and the runtime dispatch read one table.

### Placement

**FR-20** Building a world from a map places **one ground sack per ground record**, in file order, at
the record's cell, with the record's gold and its elements' item codes in element order.

*Folded from hotfix `eef792f` — see `docs/hotfix/ARCHIVE.md#eef792f`.* EVERY path that rebuilds a
mission's world carries the map's ground sacks, not the widest load path alone.

**FR-21** A **stock record places nothing**. Forty-three of them ship per root, twenty on one map; a
build that read them as sacks would put loot on the ground that the original does not.

**FR-22** A record whose cell is outside the map is **dropped, and the load succeeds**. The
simulation refuses such a sack; the loader does not fail a whole map over one record.

**FR-23** A map with no type-8 record, an empty one, or one this build cannot decode, loads with **no
sacks** — a map whose loot cannot be read still plays.

## Acceptance

**AC-1** Over both roots, `almtool loot` consumes the type-8 payload **exactly** on every shipped
map — no record running past the end, no bytes left over. **Reported, per root, in
`verification.md`**: maps carrying a section, ground records, stock records, elements, total gold,
and how many codes resolve to no class. These are measurements, not thresholds.

**AC-2** A synthetic map with a **short head** (format version below 989) decodes with the same
records and **zero gold**, and one byte more or fewer in the payload is an error.

**AC-3** A map opened, `Loot()`-decoded and written back is **byte-identical** to its input.

**AC-4** Two records naming one cell produce **one** sack whose gold is the sum and whose item list
is both lists in order.

**AC-5** A world built with sacks named in a scrambled order equals — and hashes equal to — the same
world built with them in ascending order. Two worlds differing only in one sack's gold hash
differently.

**AC-6** A world with sacks marshals and unmarshals to an equal world; the four refusals of FR-15
each have a case that returns an error and no world.

**AC-7** A script node of opcode 14 over a cell with a sack reads 1 and over a cell without one
reads 0, and a trigger conditioned on it fires exactly when the sack is there. The world's
unimplemented-check report does not name 14.

**AC-8** A map loaded from a shipped `.alm` with a loot section carries exactly the ground records'
cells as sacks and none of the stock records'. **Reported in `verification.md`**: for one map, the
placed set beside the section dump.

**SC-1** `go build ./... && go vet ./...` clean, `gofmt -l` silent, `go test -trimpath -count=1 ./...`
green, and `scripts/check-no-game-assets.sh`, `check-doc-budget.sh` and `check-sdd-audit.sh` clean.

**SC-2** No test reads a game install. Every fixture in this story is bytes built in test code.

## Decisions and divergences

**D-1 The contents are carried and nothing reads them.** No tier in this build interprets an item
code. They are carried because FR-10's merge is otherwise unobservable — without a purse and a list
there is nothing for a merge to do — and because discarding the map's authored payload at load time
would have to be undone by the story that adds a pick-up.

**D-2 The per-cell lookup is a search, not a bit plane.** The original keeps a per-cell flag beside a
hash bucket and rejects on the flag first. This build searches its ordered list. Same answer,
different mechanism, and the mechanism is not observable through the sack query.

**D-3 Stock records are decoded and not applied.** Owed by the story that gives an actor a
container. The grammar lands here so that story adds a consumer and not a decoder.

**D-4 The element's second word and its type-9 link are carried and unread.** What the type-9 list
is for is not established, so a link into it cannot be resolved without inventing a meaning.

**D-5 A sack is invisible.** Which sprite the original draws for one is not established anywhere this
build may read from. Owed by research before it can be owed by a story.

**D-6 Nothing is manufactured to satisfy an orphan check.** Five shipped check-14 nodes name a cell
no ground record occupies, four of them cited by triggers, two disagreeing with their own
author-written labels. Those nodes read 0 here, which is what the original gives them.

**D-7 The placement's coordinate truncation is not modelled.** The original stores each axis into a
byte, so a cell above 255 would wrap; this build drops such a record instead (FR-22). No shipped map
has one. The **check's** truncation is modelled (FR-17), because there it is the arithmetic that
selects the cell rather than a storage width.

**D-8 The purse wraps.** FR-10 adds at the 32-bit width because the original adds into a 32-bit
field. No shipped map comes near it — the largest single authored purse is half a million.
