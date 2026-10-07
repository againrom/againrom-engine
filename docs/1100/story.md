# Original late-dead WORLD LOAD

Original WORLD LOAD must not recreate an authored actor named by the exact
dead-manager list. Stages 3/4 become empty late corpses at saved positions;
stage 5 remains absent from play. Import must not replay death, XP or loot.
Both load doors publish only a complete candidate. Ordinary AGS SAVE/LOAD
retains source provenance and coherent current dead state through later ticks.

Authority: SAV-DEADLOAD-124..130, SAV-CELLLOAD-113, amended HERO-DEATH-026.
The exact archive list, independent signed HP/timer and stage, and empty
post-teardown containers are established. ROM1 uses identity rebinding, not
an ALM join. This bounded importer joins unique authored MapUnitID records.
SAV-TOKEN-034, SAV-TOKENPOS-074 and the unaffected Item/Weapon clauses of
SAV-MEMBER-036 establish the terminal Unit held-Weapon members. That sole
nonempty worn arm retains a fixed typed, nonplayable Weapon DTO: its source
archive index and all 96 Token/Item/Weapon bytes, including named W52/W6A
members of unknown meaning. Effects and outgoing object references must be
absent. Nothing equips, exposes, drops or fires this retained Weapon.

Touched surfaces: SAV projection, original mission import, canonical dead
records, native persistence and terminal ID reservation. No original writer,
research change or install write is included. Early stages, nonzero timer,
effects, carried contents, other worn arms and dynamic/unmatched dead actors
are refused explicitly. The slice does not close full SAV restoration.
Unpublished form 70 retains its 74-byte dead prefix, adds one presence byte
and a 98-byte Weapon payload (173-byte stride), then a four-byte section span.
Form 69 and all older published forms remain compatible.

Proof: independent shared-reference synthetic streams, signed scalar and
atomic rejection tests, a literal native record transcription, corpse-cell
walk-through, target/credit cleanup, terminal ID reservation, and both original
load doors followed by ordinary AGS SAVE/LOAD and 64 equal production ticks.
The exact five-actor owner save and all 22 dead actors of game0009 are checked
against independent literal oracles on EN and RU. Source Weapon offsets and
synthetic member sentinels independently prove all 98 native bytes. Existing
1097 Victory/defeat witnesses remain unchanged. See `verification.md` for
commands and final gate state.
