# 1008 — save safety

## Result

An `.ags` save becomes visible under its final name only after its complete contents have been
written, synchronized and closed. Final-name publication is an exclusive operating-system
operation. Concurrent processes cannot overwrite each other after choosing the same timestamp.
A failure before final-name commit leaves no final save and attempts to remove the sibling
temporary file. A refused close or pre-commit removal is returned together with the primary
failure. On non-Windows systems, exclusive hard-link creation is the commit: refusal to remove the
private source name afterward does not turn the visible valid save into an API failure and never
rolls the final name back. Retirement runs one serialized recovery pass after removing a writer
from the active set. The last of several concurrent writers therefore sees and reaps every inactive
private alias the operating system permits while still skipping active and foreign namespaces.
The next save retries any continuously refused residue before creating another file. If the process
exits first, a hidden staging file can remain as a second hard-link name, but it never appears in
the load list or replaces an existing save. Existing saves are never overwritten.

Temporary-file activity is keyed by the physical directory and filename, not by the spelling of
the path used by a caller. Relative, absolute, case and junction aliases cannot make recovery
delete another active writer's staging file.

Loading an `.ags` file prepares the complete replacement game before the current game is released.
A stale simulation version, invalid world, unavailable mission map, asset decode failure or later
mission-construction failure leaves the current screen, town, party, offer state, live map and
shared presentation caches unchanged. A successful prepared load commits once and enters the
prepared town or map.

One prepared mission opener commits at most once even if two callers invoke it concurrently. An
`.ags` file is limited to 16 MiB for the complete envelope. Listing rejects a file whose physical
size and declared envelope size disagree, and loading uses a same-handle size check plus a bounded
stream read. Before the standard gob decoder can allocate a `Snapshot`, a wire preflight charges
every array, slice, map and type-field count against one aggregate 65,536-element ceiling. A 64 MiB
sparse file and a tiny checksummed payload claiming a million-entry map are both refused before
their hostile allocation. If a listed file grows before selection, the refusal leaves the live
game unchanged.

The repository carries a synthetic fixture for the released envelope version 1 and simulation
form 53. The fixture is a compatibility census, not a migration promise. This build still refuses
every other simulation form and states that limitation in the divergence ledger. The envelope
reader rejects bytes after the declared payload and bytes after the one gob snapshot inside that
payload, even when the declared length and checksum cover those bytes. The byte form remains
envelope 1 and existing envelope-1/form-53 fixtures decode unchanged; a customized old save whose
aggregate gob container population exceeds 65,536 is now refused by authored safety policy.

## Observable

The normal Save and Load controls keep their current labels, paths and result screens. An injected
write failure produces no `.ags` row. A pre-commit cleanup refusal reports both causes and a retry
either removes the hidden residue first or refuses before starting a new save. A non-Windows
private-name unlink refusal after hard-link commit reports the final save as successful; the next
retiring writer and then the next save recover any alias the operating system allows them to
remove. A continuously refused alias remains hidden and prevents a later writer from creating more
residue. Choosing an incompatible, over-populated or otherwise invalid `.ags` row shows the refusal
on the load window while the game behind that window remains usable and exact. Oversized,
future-version, truncated and trailing files do not appear as load rows; a file changed after
listing reports its bounded refusal on that same window.

## Authority and research

The `.ags` envelope, filename and directory are authored by againrom. No ROM1 claim governs their
byte form. The 16 MiB byte ceiling and 65,536 aggregate gob-container ceiling are authored safety
and customization limits. The owner directed the atomic-write, transactional-load and
compatibility-baseline rules on 2026-08-16.

The original `.sav` disclosure is reconciled against these active claims from research pin
`4aae01f87685dc840988ca5b8bebde3ad591149e`:

- `SAV-DOC-053` enumerates the original top-level document and distinguishes world and
  between-mission forms.
- `SAV-PLAYER-028`, `SAV-PTRMAP-035`, `SAV-HERO-059` and `SAV-HEROID-065` establish player and
  party identity in the original form.
- `SAV-CARRY-050`, `SAV-HEROXP-063` and `SAV-HEROSKILL-064` establish carried, equipped and skill
  progress fields.
- `SAV-FOG-061` and `TERR-FOG-145` establish the original world form's explored-terrain record.
- `SHOP-SAVE-015` establishes that original shop stock is regenerated rather than serialized.
- `MENU-ITEM-011` and `MENU-ITEM-012` establish the original mission and town menu rows. They do
  not establish againrom's omitted typed-name, delete, overwrite or autosave controls.

The original save's undecoded or unapplied world and campaign state remains `DIV-026`. This story
does not infer missing original fields from againrom code.

## Domains

- **Persistence** owns the atomic store, compatibility policy and prepare/commit boundary.
- **Campaign & Scripts** supplies the current-install campaign and mission constructor used to
  validate a candidate.
- **Client** keeps the load window and current map intact until preparation succeeds.
- **Sim Core** validates the saved world through its existing version and invariant checks. The
  canonical simulation form stays at version 53.

## Twelve aspects

| Aspect | Scope |
|---|---|
| Data | The committed synthetic `.ags` fixture fixes envelope 1 plus sim form 53; the complete envelope is capped at 16 MiB and its aggregate gob containers at 65,536 entries. |
| Runtime state | The old and candidate games, including mutable body caches, coexist until one concurrency-safe commit. |
| Simulation | Existing unmarshal validation is exercised; no canonical state changes. |
| Player input | Save and load controls use the existing UI path. |
| AI | N/A; restored AI state remains the sim serializer's existing responsibility. |
| UI/HUD | Load refusals stay on the load window without releasing its backing screen. |
| Triggers/scripts | A prepared mission validates the saved trigger state through world unmarshal. |
| Inventory/equipment | No new representation; failed load must not replace the current party. |
| Persistence/save-load | Physical active-temp identity, serialized retired-alias recovery, exclusive no-replace publication, bounded bytes and gob counts, one exact gob value per envelope, candidate construction and one commit. |
| Campaign/session | Town, campaign progress, party and offer state change only at commit. |
| Shipped content | Mission-load witnesses run through EN and RU installs. |
| Existing mechanics | Default directories, unique names, traversal refusal, menus and original-save import remain. |

## Scope limits

This story does not add migrations for simulation forms 50 through 52. It does not restore the
original `.sav` world's full state, materialize original difficulty or omitted spellbook/session
fields, enforce an asset digest, write original saves, add typed names, overwrite, delete or
autosave controls, or change simulation form 53. Corrupt-file visibility in the list is cut unless
it composes without changing list ownership. It does not stream saves larger than 16 MiB or accept
more than 65,536 aggregate gob container/type-field entries; raising either authored ceiling
requires a new memory and customization decision.

`DIV-095` through `DIV-102` are allocated. Only supported rows are spent. Unused ids are returned
permanently in `closure.md`.
