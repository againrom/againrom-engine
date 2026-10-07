# Plan — the campaign mission start

## Decisions

**DD-1 — the band test keeps the signed key; the searches keep the byte truncation (FR-3).** The
engine compares the sign-extended word and truncates only inside the search, so the two are written
as two different reads of one record rather than as one shared conversion. A shared conversion would
be cheaper and would silently answer the Humans band for a negative key, which is the arm a wrong
answer would be least visible in — the shipped domain is `1..80`, so no map can tell them apart and
the code has to say which it means.

**DD-2 — `Resolve` keeps its signature, its arm names and its `Resolution`; only the ladder moves
(FR-3).** Two call sites already read it (`pkg/game/tiers.go`, `cmd/classdump`), and the story's
subject is which arm a record takes, not what an arm is called. `ArmHumansByType` stops being a
carve-out inside a wide band and becomes the ladder's last rung, which is the same name for the same
search.

**DD-3 — the npc table is a `pkg/data` type and rides on `mapload.Table` (FR-4).** `pkg/mapload` may
not import `pkg/formats/reg`; `pkg/data` may. So the registry is parsed one tier down into a lookup
of definition id by npc id, and the world builder receives it beside the two collections it already
receives. The sentinel is applied at the *table*, not at the arm: `26` is the registry's own value
meaning *composed*, and a consumer that got the raw number back would have to know that. A nil table
answers no entry for every id, so a caller with no campaign container needs no branch.

**DD-4 — the type-7 walk lives in `pkg/mapload`, not in `pkg/formats/alm` (FR-5).** The format tier
preserves that payload raw by contract, the node grammar is a corpus-only reading, and the trigger
runtime is another story's. Putting the walk at the consumer keeps the format tier's contract intact
and leaves exactly one interpreter of those bytes in this story's blast radius.

**DD-5 — the walk requires the framing to tile the payload exactly, and yields nothing when it does
not (FR-5, AC-7).** Three counted arrays of 796, 796 and 184 bytes must consume the payload with no
residue. That is the framing claim's own discriminator, and asking it per map at load turns a Medium
reading into a per-file test: a map it does not fit contributes no cell instead of contributing a
cell read out of the wrong offset. A truncated or absent trigger record is the same answer by the
same path.

**DD-6 — the drop opcode is matched, and no other opcode is interpreted (FR-5).** The walk reads two
words of a node — the opcode and, when it matches, the first two stored values — and skips every
other node by stride. It builds no vocabulary, no parameter table and no node type, which is what
keeps it off the script runtime's ground.

**DD-7 — the draws come from the simulation's own generator, seeded from a constant of this loader's
(FR-6, P-1).** `pkg/sim` owns the tree's only generator and keeps it private to a world; a start must
choose before a world exists, so that package gains one exported, standalone draw sequence over the
same generator, and the loader seeds it from a constant distinct from the world's. Two sequences from
one seed would agree value for value the moment anything consumes the world's own, which is a trap
worth one constant to avoid. The draw is *inclusive of its bound*, because the engine's own helper is
— that is the clause `MISSION-DROP-002` corrects, and it is spelled once, in the draw.

**DD-8 — the fallback is per axis, and the pick happens first (FR-6, P-1).** The order of draws is
part of the contract, not an artefact: pick, then — if needed — column, then row. Reversing them
would give a different world from the same seed.

**DD-9 — the party's outward walk is a Chebyshev ring scan, and it is ours (FR-7).** Radius 0 is the
drop cell, then each ring outward in a fixed order, first acceptable cell wins, bounded at the map's
own larger side so the walk terminates on any map. The engine's rule is a radius whose constant is
unread and whose consuming routine is undecoded, so this is a named divergence and not a
reconstruction; it agrees with the claim on the one thing the claim states, which is that everyone
starts from the hero's cell.

**DD-10 — party entities are appended after the map's (FR-8).** The original inserts the party first.
This tree's entity id *is* the placement's slice index, pinned since 0019 and relied on by the
render seam, so appending keeps every existing id and every existing digest for a map whose arms did
not change. The divergence is recorded rather than absorbed.

**DD-11 — the campaign entry is a `pkg/game` function returning the map, the world and the report
(FR-2).** That tier is the one that already opens archives, holds the container identity and loads
the definition table. It returns the decoded map as well as the world because the caller needs the
map for everything this story does not own — the script, the text, the tiles — and re-decoding to get
it would be a second chance to disagree about what the file says.

**DD-12 — `LoadTable` gains the npc registry and a missing one is fatal (FR-4).** That function
already treats every way it can fail as a startup failure, for the reason that what lies behind it is
the health and the domain of every placed unit. The npc registry decides *who* fifteen shipped
placements are; a front-end that ran without it would put a generic peasant where the mission's own
dialogue names a witch, silently, with every test green.

**DD-13 — the evidence tool is a new verb on `classdump`, over the two archives (AC-2, AC-4).** That
tool already resolves a loose map's placements against the table and already owns the arm vocabulary;
what it cannot do is reach the 28 maps inside the campaign container, which is exactly the corpus the
census must cover. The verb walks both sources, prints the arm census and the drop cells, and prints
no name, no string and no parameter value. **Named a map as well, it also prints one line per
placement of that map that takes the NPC arm**: the subscript, the definition id the record carries,
the server id the registry names, and the entry each of the two would reach. That is the only shape
in which "the NPC arm never reads its own definition id" is a measurement rather than a restatement
of the code that implements it — a count cannot show that two routes disagree.

## Traceability

| FR | Decisions | Criteria |
|---|---|---|
| FR-1 | DD-11 | SC-1 |
| FR-2 | DD-11, DD-12 | SC-1, SC-6 |
| FR-3 | DD-1, DD-2 | SC-2, SC-3 |
| FR-4 | DD-3, DD-12 | SC-3 |
| FR-5 | DD-4, DD-5, DD-6 | SC-4, SC-7 |
| FR-6 | DD-7, DD-8 | SC-4, SC-5 |
| FR-7 | DD-9 | SC-5 |
| FR-8 | DD-10 | SC-5, SC-6 |

## Success criteria

**SC-1** — a mission number resolves to its address, and a non-positive one is refused, under unit
test; a start over synthetic archives yields a world whose bounds are the map's.

**SC-2** — over a lawful install's 38 shipped maps the four arms count 6672 / 1405 / 15 / 2 of 8094,
and `10.alm` counts 19 / 14 / 2 / 0. The rival this story replaces is reported beside it, so the
figure that moves is visible rather than merely improved.

**SC-3** — `10.alm`'s two script-named placements resolve through the npc registry, and the entry
each reaches is not the entry its own definition id names.

**SC-4** — every one of the 38 shipped maps yields exactly one drop cell, and `10.alm`'s is (17, 66).

**SC-5** — under unit test: the fallback reaches both 30 and 100 on both axes; a party is placed with
the hero on the drop cell, no shared cells and no closed cells; and two starts from one input agree
on the drop cell, every party cell and the world digest.

**SC-6** — the full local gate is green, and a world built from a map with no party and no
band-crossing placement keeps the digest it had.

**SC-7** — a trigger payload the framing does not tile yields no cell, and the start reports the
fallback.
