# Spec — a character the player generates, and a hero whose statistics reach his health

## Terms

**Generation** — the choice of a character before a mission begins: a sex, a class, one trained
skill and four statistics.

**The spread** — the four statistics Body, Reaction, Mind, Spirit, in that display order.

**The budget** — 140. A statistic standing at *n* has a **cumulative cost** T(*n*); a spread is
**legal** when every statistic lies in [15, 45] and the four costs sum to at most the budget.
**Remaining** is the budget less that sum and may go negative.

**A base row** — one of the four shipped character rows of the definition table's `Humans`
collection, named `PC_Danath`, `PC_Naira`, `PC_Fergard`, `PC_Reniesta`. A generated character starts
from one; it is where his health column, his mana column, his class flag and his face come from.

**The archetype pair** — a row's class flag and its gender, both carried in its own **type id**:
`typeID = 0x21 + 2*class + gender`.

**The profile** — the three things the derived-stat graph reads that are not a statistic, a skill or
an item: the class flag, whether the health column is present, whether the mana column is present.

**The pools** — the health maximum and the mana maximum the graph produces.

**The recompute** — `pkg/data`'s one implementation of the derived-stat graph. It exists, it is
complete, and this story neither changes nor re-derives it.

## Problem

Two halves of one thing.

The player has no character. Every mission opens with one hardcoded hero: 43/26/15/15, trained in
Blade, drawn as a male fighter, chosen by this project and written into three unrelated constants.
The owner asked for a generation window behind a flag, whose output is the character the mission is
played with and drawn as.

And the hero's statistics reach his health nowhere. The recompute produces both pools and nothing
consumes either, so a party member is minted at the spawn constant 100 whatever his Body is — the
defect the owner reported. He is also minted with **no mana pool at all**, which is a hard blocker
on every future spell. The reason the pools were left unwired is exact: the graph reads a profile,
and nothing in this tree could state one. A generated character states one, off the shipped row he
starts from. So the screen is the fix for the defect, not a decoration over it.

## Scope

A generation screen behind a command-line flag; the character it produces reaching the mission, the
map and the inventory figure; a profile on a party member; and the two pools reaching the entity on
both paths. Not here: the original's layout, art or portrait carousel; any change to the recompute
itself; saving a character; a second party member; casting.

## The contract

### The screen

**FR-1** The application gains a fourth screen. It shows only when the command line asks for it, and
then it is the first screen shown, before any window frame is drawn for the mission. Escape leaves
it for the main menu, which is intact behind it.

**FR-2** It offers, in this order: a **sex** choice, a **class** choice, a **skill** choice, then the
four statistics of the spread. One row has the focus; the focus moves up and down and wraps. On the
focused row, one input decreases and another increases — a choice cycles its options, a statistic
steps by one. Confirming begins the mission.

**FR-3** The drawing tier holds **no statistic name, no skill name, no option label, no budget, no
bound and no cost**. Every one of them arrives as a setup value built by the wiring tier from the
definition table and from the decoded arithmetic. A statistic's cost arrives as a table indexed by
the statistic's value.

**FR-4** A step is refused, silently and without changing anything, when it would take a statistic
below its floor or above its ceiling, or when it would take Remaining negative. Remaining is
displayed, is the budget less the four cumulative costs, and is never clamped.

**FR-5** The skill choice's option set is a function of the class choice's value. Changing the class
re-labels the skill row and keeps its index, so a class change never silently re-trains a character.

**FR-6** Confirming is refused while the spread is illegal, and the screen says so rather than
appearing not to respond.

**FR-7** The screen holds no archive, no definition table and no simulation type — it is given
strings and integers and hands back integers.

### What the choice becomes

**FR-8** A generated character starts from a base row, searched **by name** in the `Humans`
collection. The four names are in archetype order — fighter male, fighter female, mage male, mage
female — so the chosen class and sex index the list directly. A name that does not resolve falls
back to the first of the four that does; if none resolves, the character has **no base row**.

**FR-9** The row's **type id column is not the archetype**: it carries the drawn class id, and the
runtime type id is computed rather than stored. Nothing decomposes a column here — the archetype is
the slot the player chose.

**FR-10** The profile comes off the base row and nowhere else: the health column is present when the
row's health maximum column is nonzero, the mana column when its mana maximum column is nonzero, and
the flag the pool graph multiplies by is set for a row carrying **no** mana column. That last is
AUTHORED and its alternative is live: the flag's runtime writer sets it for an actor holding a
spellbook and a pool, which would hand the caster the doubled health instead. Research resolves
neither reading; this tree takes the one under which a fighter is the tougher of the two, and it is
one function to invert. With no base row the profile is the zero profile — no flag, neither column — which is exactly what every party member in
this tree has carried until now.

**FR-11** The character is trained in exactly one skill slot, at the ordinary starting level, with
the other five at zero — the same construction the tree already uses for its authored hero.

**FR-12** The inventory figure's directory is chosen from the class and the sex; its face is the
base row's own face column, or 1 when there is no base row. Neither is a constant in the wiring
tier any longer.

**FR-13** The starting weapon is the one the chosen skill slot names, through the table already in
the tree. The body name the character is drawn as, and the class key he carries, derive from that
weapon exactly as they do today — so **the skill choice changes the map sprite and the sex choice
does not**. That asymmetry is the game's and is disclosed, not smoothed over.

### The pools reach the entity

**FR-14** A party member carries a profile alongside his statistics, his body and his weapon.

**FR-15** A mission start performs **one** recompute per member, over that member's profile and his
weapon, and every derived number it writes onto the entity comes from that one call — the combat
block, the step rate, the sight radius and now the pools. No member is recomputed twice with
different arguments.

*Folded from hotfix `dbd5c89` — see `docs/hotfix/ARCHIVE.md#dbd5c89`.* A mission start may be
handed a party member carried from a finished mission, whose per-slot experience, the skill
levels those values imply, container and equipment slots are taken as he ended that mission
rather than minted. His two pools are still the recompute's own maxima (FR-16): a wound does not
cross a mission boundary.

**FR-16** The health pair is the recompute's health maximum on both fields when the member's profile
**states a health column** and that maximum is positive, and the spawn constant on both fields
otherwise. The mana pair is the mana maximum on both fields under the same two conditions on the
mana column, and zero otherwise. **The column is the gate, not the positive maximum.** The health
arm skips only its FIRST term when the column is absent and still adds the experience term, so a
trained character carrying no column derives a small positive maximum that is not a health at all —
two, for the character this tree ships today. Gating on the column is what keeps a character with no
base row at exactly today's behaviour instead of at two health.

**FR-17** No field is added to a simulation entity and the serialized byte form does not change, so
the form's version constant is not spent.

**FR-18** The margin of the pool graph's logarithm against an integer truncation boundary is
**measured** over the reachable space and recorded, as the recompute's own doc block demanded before
this seam could be wired. The measurement is a test, and the doc block is corrected to state the
figure instead of the debt.

### The door

**FR-19** One new flag opens the screen. It requires the mission flag: given without one it is a
command-line error naming both, before any window is opened, because a generated character would
otherwise reach nothing.

**FR-20** Absent the flag, **not one statement of the screen is reached** and the character the
mission starts with — his spread, his trained slot, his weapon, his body name, his class key and
his figure directory — is exactly the character it starts with today. Two things are not: his health
pair, which is the defect being fixed on both paths, and his figure **face**, which now comes off
the base row (FR-12) instead of the constant 1.

**FR-21** The headless check mode accepts the flag and reports what the screen would offer, opening
no window.

**FR-22** *Folded from hotfix `7e3176c` — see `docs/hotfix/ARCHIVE.md#7e3176c`.* The generation
screen states what a point buys: six derived lines under the footer, computed by the same
expression the party mint uses, absent entirely while the spread is illegal. The screen takes them
as a field on its setup, resolving nothing itself.

## Acceptance

**AC-1** With no flag, the party a mission starts with is identical to today's on every field but
the health and mana pairs. Witnessed against the values spelled in the test, not against a rerun.

**AC-2** A statistic raised from the start value by one costs exactly the difference of the two
cumulative costs, and lowering it back refunds exactly that much: Remaining returns to its previous
value.

**AC-3** No sequence of steps can leave the screen with an illegal spread: every reachable state is
legal, and the refusals are what makes it so.

**AC-4** A step that would break a bound changes nothing at all — not the statistic, not Remaining,
not the focus.

**AC-5** Changing the class re-labels the skill row and leaves its index where it was.

**AC-6** Confirming an illegal spread produces no result and leaves the screen showing.

**AC-7** A base row is selected by matching archetype pair; when two rows carry the same pair the
first in the searched order wins; when none matches the first that resolved wins; when none resolves
the profile is the zero profile.

**AC-8** A member carrying a profile with a health column has an entity health maximum equal to the
recompute's, and it **moves with Body**: two members differing only in Body have different maxima.

**AC-9** A member carrying the zero profile has the spawn constant on both health fields and zero on
both mana fields — today's behaviour, unchanged.

**AC-10** A member carrying a profile with a mana column has a positive mana pair.

**AC-11** The screen holds no archive and no definition table: its package's import list is unchanged
by this story.

**AC-12** The flag without a mission number is a command-line error that names both flags and opens
no window.

**AC-13** The logarithm margin is measured over two domains and both figures are recorded: the
whole integer experience range, and the range a character this tree can construct. The second must
stand clear of a logarithm's last-place error by at least ten orders of magnitude. The first does
not, and the test names the exact, finite set of experience values that carry the risk rather than
reporting a single worst case.

## Derived properties

**P-1** There is **one** expression in the tree that turns a generation result into a party, and one
that turns a base row into a profile. Nothing composes either by hand beside them.

**P-2** The screen's model is pure — no engine, no clock, no archive, no randomness — so every one of
its states is decidable in a test.

**P-3** Every label, bound, cost and option the screen shows is an argument to it. Pointing this
story's screen at a different table is a change to the setup builder alone.

**P-4** Only the wiring tier may open the archive or read the definition table. The drawing tier
receives finished values.

**P-5** Nothing authored here is wider than one edit: which gender addend is female, the default axes
when the flag is absent, and the screen's own labels are each one named place.

**P-6** Tests are synthetic. The base row search, the archetype decomposition and the profile are
exercised against a collection built in test code, never against an install.

## Out of scope

The original's chargen layout, its portrait carousel and its art. A second party member. Saving or
loading a generated character. Casting, spellbooks and the mage weapon arm. The experience-to-level
inverse. The Russian-language chrome question — the engine's debug font renders no Cyrillic byte and
that is a separate story. Deciding which archetype the pool graph's class flag names.

## Verification mapping

AC-1, AC-9 by a party-identity test in the wiring tier. AC-2 to AC-6 by the screen model's own tests.
AC-7 by a synthetic `Humans` collection. AC-8, AC-10 by a mission-start test. AC-11 by the existing
import-graph test. AC-12 by the command's flag test. AC-13 by the margin measurement. The 100-health
defect is witnessed by reverting the wiring and showing the test go red.

## Gate check

The project gate, on the clean committed tree: build, vet, gofmt over our own files, the full test
run with trimpath, and the three repository scripts for game assets, doc budget and SDD audit.
