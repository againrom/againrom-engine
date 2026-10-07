# Analysis — the table every placed unit's numbers come from

## Intensity & terrain

**spec-anchored / static**, per the profile's `pkg/formats/*` default: `Data.bin` is a game file
whose grammar is the durable contract, and a second consumer of the same table is already
foreseeable. No watcher tool exists here, so the sync is discipline.

Terrain, declared per changed area: **greenfield** for the new format package and for the typed
definitions beside it; **brownfield** for `pkg/mapload`, whose shipped world-building behaviour is
what acquires a second input.

## What was not known

Nothing in this tree reads `Data.bin`. Three landed stories record the same hole from three
directions, and each left a marker rather than a workaround: the health every placed unit is born
with is a constant declared provisional at its own spawn site; the passability derivation refuses to
bake a structure because the footprint masks live in a table nothing here reads; and the kill arm
folds a per-class countdown into a second blow because that countdown's length is a column nothing
here reads. All three markers name the same file.

So the question this story opens is not "does the baseline still hold" — **there is no baseline**.
The old clean-room material stops one story short of this one, so every clause here is either
traceable to a claim at the submodule pin or is ours by choice and says so.

## What we looked at

The claim ledgers for the file, for the non-hero actor, and the two registry rows that say which
file carries which number. Four things were read closely enough to change the shape of the contract.

**The file cannot be read in part.** The Units collection is the fifth of eight groups, and every
group's length is a function of what precedes it — counted string arrays, counted parameter arrays,
per-class payloads that differ in shape. Reaching the table this story is named after means walking
the seven other groups, so "parse the whole file" is not scope creep; it is the only way in.

**Two of the eight groups count differently from the other six.** The collections of the first two
groups serialize every entry; the rest allocate an entry 0 and never write it, which is what makes
every consumer index 1-based and a count word one larger than the entries behind it. A reader that
treats all eight alike is wrong at the third group and then wrong about everything after it.

**The streamer is a sequence, not a field map.** Thirty-eight parameter slots are consumed in order
by four helpers, each of which compares the value with −1, skips its store on a match and advances
the cursor either way. Two slots go to locals and become a spread; one routes that pair through a
switch whose selector is a local pre-set to zero; one is read and dropped. A loader written as
"field *i* takes column *i*" mis-assigns everything from the damage pair onward — and would look
right, because the eleven fields before it would still agree.

**The information display and the map spawner run the same arithmetic.** A three-valued setting
weakens or strengthens a placed non-hero at the moment it is built — health, and at the top setting
to-hit and defence as well — and both adjusting arms then bring current health up to the new
maximum. It is consumed once, at spawn, so nothing downstream can recover it: a spawner that copies
the template's numbers straight through disagrees with the original on two of its three settings.
The owner has since named the setting: it is the **difficulty**, chosen on the character pre-create
screen. The arithmetic and the control are decoded; the name is the owner's.

## Assumptions checked rather than carried

- *Is a unit's health derived, as a hero's is?* No — there is no derive on the non-hero arm at all:
  the one virtual that could hold it writes a single mana floor and reads one field. The template
  numbers **are** the stats. That is what makes this story small enough to be worth doing.
- *Does a hero come out of the same table?* No. A placement whose key selects the humans arm reaches
  a different collection with a different and much shorter slot list, and a hero's health maximum is
  computed by a derive from body and experience rather than read from a column. The complete humans
  slot list is not published in a claim, only inside an experiment's evidence file — so it is a
  request to research, not a gap to fill by inference.
- *Is the `(typeID, face)` key unique?* On the shipped population, yes — a second, independent
  consumer of the same two columns builds a table on that key with no collisions. But the engine's
  own search is a first-match ascending walk, so uniqueness is a corpus property and not a licence
  to refuse a file that breaks it.
- *Does the low byte matter?* The engine truncates both class keys to a byte before the search. A
  key of `0x140` therefore selects the same entry as `0x40` — which no shipped map exercises, and
  which a reader passing the full word would get wrong.
- *Does a class's own body reach its carrying capacity?* No: the constructor computes capacity
  from the **default** body before any column is read, and nothing recomputes it on this arm. A
  decoded quantity that no class can change is not worth carrying.
- *Are the two dead arms of the damage switch reachable?* Not on either shipped root: the selector
  is absent on 48 of 56 rows and takes one other value on the remaining 8. The two unexercised arms
  write a different field pair whose consumer is an open question in the hero ledger.

## What we did not look at

The equipment column and its name grammar; the spell columns and the spellbook they build; the
treasure columns; the seven collections this tree only frames; the buildings footprint masks; and
the display's layout, which research states is unreachable without the interface layer. Each is
named in the contract's own out-of-scope list with the reason it stays there.
