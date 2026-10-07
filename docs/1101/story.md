# Source-backed mage spellbooks

Imported mages retain their book's presence and each saved Spell's range,
Defensive byte and signed mana-cost word. Book casts, admission, retry, AI and
the popup consume those instances. Weapon, scroll and script casts keep their
own sources. Absent books stay absent; teaching an already-known spell does
not replace its instance. Legacy native records keep their former table-backed
semantics.

Source-backed school training uses the complete Human derive and refreshes
the existing book. Ordinary SAV output updates those Spell bodies only when
roster, items and the independently reconstructed training history agree.
Shared Spell objects require identical requested values from every alias;
conflicts refuse transactionally. Native continuation retains the same state.

Authority: MAGIC-SPELL-001, MAGIC-BOOK-002, MAGIC-CAST-003,
MAGIC-POWER-004 (unretracted formulas only), MAGIC-AI-012,
MAGIC-ARM-014, HERO-SKILLBUY-076, HERO-SKILLUP-073 and HERO-ORDER-014. The promoted derive
excerpt reaches L13186 -> R0903 -> R0904 -> R0625.

Proof covers independent literal records, absent/empty books, distinct instances
of one ID, repeated teaching, zero/Teleport range refresh, signed cost and raw
Defensive consumers, queued casts and native continuation, aliases, and the
actual town Train/SAVE route using the Reniesta/Danath city save and actual
mission casting from Fergard's saved book, replayed with both asset roots.
Those are not independent original-runtime locale recordings.

Excluded: first-book allocation (reachability Unknown), generated Human/world
SAV writing, original-process automation and lawful-install writes. Form71
appends 113 bytes to each actual form70 entity; its 173-byte dead records stay
unchanged. DIV-650/651 close; DIV-652 preserves legacy native behaviour,
DIV-690 retains the first-book Unknown, and DIV-691 records shared-object
transaction refusal. Later native Human recomputation remains DIV-675.
