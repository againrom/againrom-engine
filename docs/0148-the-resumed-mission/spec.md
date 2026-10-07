# 0148 — the resumed mission

## Contract

A mission resumed from an original save runs the mission it was taken in, and lets go of the save
at the mission boundary.

This is one contract with three behaviours under it. Each is a way the resumed mission stopped
being the mission it was taken in: its script fired arms a fresh start does not fire, its party was
led by the wrong character, and its saved cell followed the party into the next mission.

## Background, self-contained

`pkg/game.RestoreParty` turns an original save's characters into this tree's party. A character the
map placed and the player then took into his own group — a hired mercenary, or the witch mission 10
hands over — keeps the map unit id of the record he was placed from. `pkg/game.WithdrawRestored`
removes that record from the map, so the same person does not become two entities.

A map's type-7 script names units by a `Target_Unit` parameter. Below 10001 it names a placed unit
record's own identifier word. `pkg/mapload.ScriptUnits` builds the id-to-entity table from the
records the map holds, so a withdrawn record's id is in no table and resolves to nothing.

An unresolved reference does not disarm a check. The check is still built, still takes its
register, and returns before writing anything. The register keeps its initial zero, and a
comparison against zero can hold. That is stated in `pkg/mapload/script.go`'s own `ScriptReport`
doc; 0147 did not apply it to the case it created.

`pkg/mapload.Saved` is what a save recorded for one member: his cell and his two pool pairs. It is
a pointer field on `PartyMember`, and `pkg/mapload.CarryParty` copies the member struct when a
mission is won, so the pointer travels into the next mission's party.

## Functional requirements

**FR-1 — The restored party leads with the human participant's own character.** The character whose
runtime creation-order id is 1 becomes party member 0. Every other character keeps the order the
file's actor list holds. Where the file carries no character with that id, or more than one, the
file's order is kept unchanged and the report says the rule did not apply.

**FR-2 — A withdrawn map record's unit id resolves, in the mission's compiled script, to the entity
the restored character became.** A restored character carries the map unit id his record claimed.
The id-to-entity table the script compile builds resolves that id to the party member's own entity,
so a check naming the withdrawn unit measures the person where he actually stands rather than
writing nothing.

**FR-3 — A `Saved` does not cross a mission boundary.** The party a won mission hands to the next
one carries no `Saved`. The next mission places its members by its own map's drop cell and mints
their pools from the fold, exactly as it does for a party that came from no save.

**FR-4 — The resume report states both in the reader's units.** The report says which file position
the leader came from, or that the rule did not apply, and how many withdrawn placements the script
rebound.

## Acceptance criteria

Every criterion below is checked headlessly. AC-1 and AC-2 are also checked against a real install
in `verification.md`; the AC itself is the synthetic test.

**AC-1 — A resumed mission fires no script arm a fresh start does not fire at the same tick, for
the reason FR-2 names.** Concretely, over a map whose script holds a proximity check on a unit id a
restored character claims: the check writes its register, and the trigger reading that register
against a small constant does not fire.

**AC-2 — No resumed mission is decided by an unresolved unit reference.** Measured on the install in
`verification.md`, over all six corpus saves that resume into mission 20 and withdraw a placement:
every unit reference resolves, and the outcome each save reaches is the mission's own. Three reach
tick 400 undecided, which is what a fresh mission-20 start reaches; two lose when the escorted
character is found dead; one wins on a measured distance of 3.

This criterion first read "reaches tick 400 undecided ... over all six". The corpus refutes that
universal: three of the six do. The other three are decided by checks that resolve and that could
not fire at all before this story.

**AC-3 — Party member 0 is the character whose runtime id is 1**, over every corpus save that holds
one. Where none or several do, the party is the file's order unchanged.

**AC-4 — A party carried out of a mission carries no `Saved`**, and its members are placed by the
next map's own drop cell.

**AC-5 — A fresh start is unchanged.** A party that came from no save has no `Saved`, so FR-2 adds
no entry to the id-to-entity table and the compiled script is the one this build already produced.
The script-gap census over both installed roots is unchanged.

## Design decisions

**DD-1 — The discriminator is the runtime creation-order id, and the rule is total.** `SAV-ID-015`
is High that the hero's id is 1, and reads the allocator: the id is the lowest free bit of a
bitmap, so the hero holds 1 while nothing has been freed. The claim's own Medium clause is that a
corpse decay frees a bit and the next spawn reuses it, which is why FR-1 states what happens when
the id is absent or duplicated instead of assuming it cannot be.

**DD-2 — The rebinding travels on the party member, not on a side channel.** `mapload.Saved` gains
the map unit id the character claimed. `ScriptUnits` takes the party and adds one entry per member
whose `Saved` names a unit. The alternative — passing a separate table from `WithdrawRestored`
through `StartMissionFrom` — puts a second, parallel description of the same fact on a path that
already carries the member.

**DD-3 — A party member's binding wins over a map record holding the same id.** The withdrawal
removes the record, so the two do not normally collide. Where they do, the entry the party writes
is the one that survives: the commandable figure is the one a script arm should measure.

**DD-4 — The `Saved` is cleared in `CarryParty` and not in `NextParty`.** `CarryParty` is the
mission boundary; `NextParty` is read more than once per mission and reads a field it does not own.
Clearing at the boundary makes the party in `FrontEnd.Carried` correct for every reader, including
the snapshot encoder.

**DD-5 — Reopening the same mission is not a boundary.** This tree's own snapshot of a live mission
carries the party it was started with, `Saved` included, and restores into the same mission number.
That is not a crossing and is left alone.

## Limits

**L-1 — The ordering rule is AUTHORED.** `SAV-ID-015` supplies the discriminator; nothing published
states what order a consumer should place the characters in, and the file's own actor-list order is
not hero-first. Population: the 23 save files of the preserved corpus, 18 distinct by content, which
is `SAV-OWNER-048`'s own population. Exactly one character carries runtime id 1 in all 23, and that
character is at file position 0 in 15 of them and elsewhere in 8. What the file's order means is an
open research question and is not answered here.

**L-2 — A restored character's own map record is still withdrawn**, so a script arm that acts on
that record as a map unit — an order to a group it belongs to, for instance — reaches a unit that is
now a party member. FR-2 fixes what a check MEASURES; it does not make a party member obey a group
command. Nothing in the corpus was observed to need that.

**L-3 — The explored plane is still not restored.** A resumed mission shows only what the party
reveals after the load. The plane is not decoded in `pkg/formats/sav` and this story does not decode
it.

**L-4 — A resumed mission may still win at once, correctly.** A save taken after its mission was
won records the party standing at the objective, and the objective's proximity trigger then fires
because it is satisfied. That is the mission being re-won, not the defect FR-2 removes: every unit
reference in that case resolves. The report is what tells the two apart.
