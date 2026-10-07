# 1051 — Read spell books

## Result

A living mage can use a carried one-spell book from the inventory. The book
teaches its kind-42 spell and consumes one unit. The learned mask survives a
save, a mission boundary, and a persistent joiner boundary.

Mission 80's shipped Astral book now teaches Teleport. Teleport can target an
in-bounds unseen cell, so the disconnected grave is reachable. Range, terrain,
mana, caster state, and the spell's other simulation refusals are unchanged.

## Authority

The owner requires the Astral book and Teleport route for mission 80. The EN
and RU maps each place unit 143 at `(36,129)` with one code `0x0e17`, kind 5
book whose sole kind-42 effect teaches spell 26. The compiled map trigger moves
that complete item to the hero. The grave victory trigger accepts the hero near
`(16,12)`.

ROM1's precise visible, explored, or unseen Teleport admission remains Unknown.
`DIV-103` records the owner-directed rule.

## Boundaries

`ItemInstance.BookSpell` accepts kind 5 with exactly one kind-42 operand in the
spell-mask range. `sim.KindReadBook` is the canonical writer. Inventory use only
queues that command. A fighter, a dead mage, an invalid element index, and a
malformed book change nothing.

Shop generation is not changed here. It is a separate producer story; map
books and reading no longer depend on that producer.

## Evidence

`TestReleaseMission80AstralBookTeachesTeleportAndReachesTheGrave` runs against
both lawful installs. It reads the shipped item, drives the compiled Give All
trigger, uses the production inventory command, teleports from the main
component frontier `(27,46)` to unseen passable grave cell `(14,14)`, and
requires the shipped mission outcome to become won.

Package tests cover exact consumption, duplicate learning, refusal population,
deterministic replay, byte-identical save round trip, ordinary party carry, and
persistent joiner carry.
