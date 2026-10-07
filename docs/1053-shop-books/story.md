# 1053 — Shop spell books

## Result

The fourth shop selector opens a shelf of one through eight readable spell
books. Each eligible installed Spells row has a positive slot-21 cost within
the shop ceiling. Its school selects the shipped generic book code. The book
stores kind 5, one mode-0 kind-42 effect naming the spell, and exactly the
slot-21 value.

A purchase keeps that complete identity. Inventory use reaches story 1051's
read command, consumes the book, teaches the spell, and carries the learned
mask into the next mission. Selling an unread book returns it to the fourth
shelf. Equal books may stack; different spells sharing one school code do not.

## Authority

The owner requires generated readable books. `SHOP-GEN-005` gives the fourth
shelf a count from 1 through 8. `SHOP-POOL-021` identifies Spells slot 21 as one
of its two priced spell-item pools but leaves the book-versus-scroll subtype
mapping at Medium. The owner-directed kind-5 book producer uses the same
kind-42 instance that both lawful mission-80 maps ship.

The five school codes are Fire `0x0e15`, Water `0x0e14`, Air `0x0e13`, Earth
`0x0e16`, and Astral `0x0e17`. The shipped item-name table supplies the generic
book label. The selected Spells row supplies the taught spell name.

## Surfaces and proof

`pkg/mapload` reprices an exact one-effect book from Spells slot 21 and applies
that rule to ALM-loaded items. `pkg/game` owns generation, trade identity, the
fourth selector, and the installed label. `pkg/ui` already owns all four
selector rectangles and now has an explicit fourth-selection paint witness.

Focused mapload, game, and UI tests cover price, code mapping, eligibility,
count, stacking, sale routing, label source, and the selector. The gated
`TestReleaseShopBookPurchaseReadAndMissionCarry` runs on each lawful root. It
generates, buys, reads, carries, and starts the next shipped mission through
the production boundaries.

## Open debt

Scrolls, Spells slot 20, and the six literal potions remain absent. `DIV-465`
records that bounded fourth-shelf fidelity debt. This story is stacked on 1051
until its book-reading dependency lands.
