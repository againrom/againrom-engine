# Divergences -- Second-game spellbooks

Part of the ledger described in `docs/DIVERGENCES.md`.

## Authored where research is silent

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-3921 | ROM2 magic / known spells in the book and bar | Owner: a new second-game mage's known spells are listed and castable | `R2-ASSET-076`: both mage templates carry bits 24 and 27 (High). `R2-ENGINE-290`: the generator loads that template. `R2-ENGINE-444` and `R2-ASSET-103`: the 24 slot values are High; their spell-row join is Medium, and the writer of the book masks and hover arrays is Unknown | The shared book builder uses the second-game slot table, including Heal at slot 4 and Shield at slot 19. The bar reveals those cells of the installed atlas; individual icons use the same slot. Generated spell membership, rules and saved bytes are unchanged | UNKNOWN | The Medium slot-to-row join is adopted for availability, clicks and icons; no original runtime witness establishes the new mage's displayed cells | A promoted second-game claim establishing the book-mask producer and its slot-to-row and icon binding | OPEN |
