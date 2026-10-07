# 0147 — provenance

Every fact this story reads from the original game, with the claim id it comes from and the
confidence that claim carries. The `research/` submodule is pinned at `d1e38ad` for the whole
story; nothing here was read from an experiment folder.

## What the object walk is built on

| Claim | Confidence | What is used |
|---|---|---|
| `SAV-OWNER-048` | High | The human participant's own objects are the subtree of the stream's FIRST top-level record. The walk starts at record one; it applies none of the three tests, because the claim measured them. |
| `SAV-TOPLVL-052` | High | The stream does not chain above the first record: one `Player`, one back-reference tag, then the `0000` null-objref word. This is the walk's own extent test. |
| `SAV-STREAM-013` | High | The `CArchive` framing: one shared index counter starting at 1, a class record taking the next index, `0x8000 \| classIndex` introducing a later instance. |
| `SAV-MEMBER-036` | High, **NARROWED** | The eleven `Serialize` bodies field by field, the counted lists, the `CString`, the two presence flags. Its `Human` clause is retracted and is not used; see below. |
| `SAV-HUMAN-043` | High | `Humanoid::Serialize` writes `Unit`, 24 raw bytes and **thirteen** object references. This is the clause that replaces the narrowed one. |
| `SAV-EMBED-039` | High | The class at each of the eight embedded sites, and that `Group+0x20` is first in a group record's file order. |
| `SAV-WLIST-040` | High | The unnamed `u16` list class: a `CArchive` count then two bytes per element. |
| `SAV-SPELLBK-041` | High | `Spellbook::Serialize` writes a count and one fewer reference than the count. |
| `SAV-DIARY-042` | High | A `Diary` record is `CDWordArray`, `CWordArray`, `u32`. |
| `SAV-SPELL-044` | High for the shape | A `Spell` record is nine bytes. |
| `SAV-UNITLEN-045` | High for the programme and the 603 | A `Unit` record's fixed part, term by term. Used as a CHECK on the programme rather than as an input: see verification.md. |
| `SAV-TOKEN-034`, `SAV-PTRMAP-035` | High | The 37-byte head, its field positions, and the identity/reference pair. |
| `SAV-OBJ-016` | — | Group records are a direct inline call with no class record, so a group takes no index and is read into the enclosing `Player`'s record. |

`SAV-MEMBER-036`'s retraction row was read before the walk was written. A programme built on its
`Human` clause desynchronises eight bytes into the first `Human` of every save and never recovers.

## What is applied to a restored character

| Claim | Confidence | What is applied |
|---|---|---|
| `SAV-UNITFLD-049` | High for the nine named words and the alignment anchor | `+0x84 +0x86 +0x88 +0x8a` as body, reaction, mind, spirit; `+0x94/+0x96` as the health pair; `+0x9a/+0x9c` as the mana pair; `+0x98/+0x9e` as the two regeneration periods. |
| `UNIT-STREAM-001` | High | The nine words are that claim's own slots 0 to 8, and the two copy instructions `+0x94 ← +0x96` and `+0x9a ← +0x9c` are why the pair reading is a pair. |
| `SAV-CARRY-050` | High for the twelve worn slots and for the four constructs being distinct | The twelve armour references at `+0x198 + 4i`, the presence-flagged container at `+0x7c`. |
| `ITEM-HUMEQ-030` | High | `Weapon::Equip` stores into `actor+0x74` and `Shield::Equip` into `actor+0x78`, so those two references are equipment as well. |
| `ITEM-SAVE-014` | High | `u16 +0x40` is the first field `Item::Serialize` writes past the effect list, and the container's serializer stores its insert index and load rather than recomputing them. |
| `ITEM-APPEAR-023` | High | `item+0x40` is `(material << 12) \| (slot << 8) \| (shape << 5) \| row`, assembled by one builder five routines call with the same four arguments. This is field for field the sixteen-bit word `data.ItemCode` composes, so a saved item's identity needs no mapping table. |
| `ITEM-DEF-002` | High | The head's `+0x0c` is a row index into the collection the C++ class picks, and `Human::Serialize` resolves against `L02111`. |
| `DAT-OBJ-002` | High | `L02110 + 0xa0` is Humans, which is what makes `+0x0c` a Humans row for a `Human` record. |
| `SAV-DEATH-051` | High | Every actor reachable from a `Player` is alive, and the stage byte at `+0x13c` is what says so. The walk reads it and reports it rather than assuming it. |

## What is read and deliberately not applied

Each row is an axis this story reaches and does not write into the simulation. The threshold is
High because a value applied to a restored character changes the world hash, so an axis whose
meaning is Unknown is not applied at all.

| Axis | Why not | Claim |
|---|---|---|
| `+0x8e` and `+0x90` | Unknown. | `SAV-UNITFLD-049`'s own Unknown clause |
| Carry capacity `+0x92` | Named, and this tree has no capacity field to put it in. The value is a fossil of the constructor default for a monster and live for a person, which is a fact a later story needs and this one cannot use. | `SAV-UNITFLD-049` |
| Speed `+0x8c` | Named, and this tree derives a party member's step rate from his Reaction by the original's own branch. The restored Reaction is what reaches the entity; the file's own word is read and not written, so the two cannot disagree in the world. | `SAV-UNITFLD-049`, and 0082 FR-10 for the derivation |
| The spellbook's spells | The record's shape is High and what its three bytes and its word MEAN is Unknown. `+0x08` is looked up in the collection at `L05046`, which `DAT-OBJ-002` places as Spells, but the claim carries the meaning as Unknown and this story does not read it as a spell id. | `SAV-SPELL-044` |
| The journal's two arrays | Unknown what they hold, and this tree has no journal. | `SAV-DIARY-042` |
| The six skill levels and their experience | No published claim locates either in a `Unit` record. The fourteen `u16` words are the whole of what `SAV-UNITFLD-049` names and none of them is a skill. | — |
| Everything past the terminator | The walk reaches one `Player` subtree. The map's own units, ground items, sacks, corpses, cell records and session block are past the `0000` word and nothing published says where the next top-level record starts. | `SAV-TOPLVL-052` |

## Ours by choice

- **The equipment routing.** A restored piece goes into the slot its own item code names
  (`data.EquipSlotFor`, field B), not into a slot chosen from the site the file read it from. The
  file writes fourteen equipment references and the code already states the slot, so there is no
  site-to-slot table to get wrong. Measured: 200 worn pieces over the owner's fourteen saves, every
  one with field B in 1..12.
- **Withdrawing a claimed placement.** A character carrying a map unit id the map still holds is
  one person the file names twice. This story withdraws the map's record. The divergence it accepts
  is stated in spec.md L-2.
- **The fallback.** A save whose walk reaches no character opens with the party a fresh start
  builds, and the report says so. Nothing published says a save must carry a character.
- **The mage flag and the figure directory** stay at their fresh-start values. Which archetype the
  class flag names is this project's own contested reading (`pkg/mapload.PartyMember.Mage`), and a
  saved character states no archetype this tree can read.

## Open, and a question for research

**Does `Spell+0x08` name a Spells-collection row?** `SAV-SPELL-044` reads the load arm looking that
byte up in the registry at `L05046` and grades the byte's meaning Unknown. If the lookup is the
row index, then a saved spellbook maps directly onto `sim.Entity.KnownSpells`, whose bit `i` is
already the Spells-collection index this tree resolves a spell name to
(`mapload.SpellIDByToken`). Two of the owner's forty-two characters carry a spellbook. Until the
byte's meaning is published, a restored character knows exactly the spells a fresh one knows.
