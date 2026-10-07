# analysis — 0133

## The three-phase read of the changed areas

**Architecture.** Three tiers carry a person's health. `pkg/data` holds the definition streamed off a
`Humans` row and the derived-stat graph a character goes through; `pkg/mapload` resolves a placement
to a definition and mints an entity from it; `pkg/sim` holds the entity and hashes it. The graph is
outside the determinism wall and the number it produces crosses it.

**Module.** The graph has one entry point and every accessor beside it is a one-line call into it.
Its health arm and its mana arm are written as neighbours in one function, and the two look alike:
each has a column flag, a class multiplier, an experience term and a growth term. The placement arm
does not call the graph for health at all — it takes the definition's streamed maximum.

**Detail.** Three questions had to be answered against the source rather than assumed:

* *Does anything downstream overwrite a person's health after the placement arm sets it?* The entity
  literal writes the pair once from the resolved block. The difficulty adjustment is a function of a
  **creature** definition and the person arm never reaches it. The party mint is a separate loop over
  members the map does not place.
* *Is the graph's mana arm column-gated in the same way as its health arm?* It is not, and the
  difference is the whole reason the health arm's gate is wrong. The two arms were written from one
  reading of one routine, and only one of them matches it.
* *What does the column flag still do once the health arm stops reading it?* It is read in one more
  place — the party mint, which uses it to tell a member built off a shipped row from one built off
  nothing. That reader is unaffected by the correction and its own justification is not.

## What we did not know at the start, and how it was settled

* **Whether player-owned placements already wear their row's equipment.** Measured before the
  contract was written, by building mission 20's world from both lawful roots and reading the worn
  set off it. They do: the placements in question carry six worn items each. The equipment half of
  the owner's report needed no work, and the story is the health half alone.
* **Whether a tool already prints a placed person's health.** One does — the definition-table verb of
  the class-dump tool prints a per-placement health maximum read off the built world. So the story
  needed no new instrument, only two fields the existing one does not carry.
* **How far the correction reaches.** Applied as a spike and reverted, it moved exactly six landed
  tests across three packages, every one of them asserting that a placed person's health is his row's
  column or that a character with no row derives a maximum of 2. No simulation test moved, which is
  what says the byte form is untouched.

## Looked at and not used

* The placement record's per-actor statistic block. It is decoded and published, and this tree's map
  reader does not carry the bytes it lives in — so applying it would have been a format change ahead
  of a behaviour change. One person-arm record in either root's whole corpus uses it, on a map
  outside the campaign.
* The health addend a worn item folds in. Established as zero on this path rather than ignored: the
  block it lives in is cleared at construction and no instruction on the placement path writes it.
