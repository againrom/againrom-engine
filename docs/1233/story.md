# 1233 — current dead container in SAV

## Intent

A terminal dead actor whose current World has no container must not regain an
old container flag from the retained AGS document during SAV. Actors with a
current container keep its flag and bookkeeping.

## Authority

The owner's `mission111-portrait-input.ags` (SHA-256
`522d5cd01dea60a6d64637884f0427b58888c329be3458259c14d17bce5afe35`)
has 43 terminal dead actors. The current World has 42 absent containers and
one present container. Its retained SAV document still carries an older
`HasInventory=1` and tail `[10000,0]` for the 42 absent containers. The
pre-change first and second SAV both promoted those stale fields, and cold
LOAD adopted them. This is a current-state preservation rule. Original-game
behavior for these terminal objects remains untested.

## As built

The dead-actor producer writes `HasInventory` from the current
`OriginalDeadSource.ContainerPresent`. When absent, it removes the guarded
container count, references and tail fields from the retained document before
the item graph retires disconnected objects. The final dead-actor pass repeats
the current value after graph reindexing. When present, it writes the current
tail. The SAV grammar requires guarded fields to be absent when the flag is
zero.

## Proof

- `TestCurrentDeadContainerOverridesRetainedDocumentAcrossSaves` creates an
  independently written terminal Unit with a valid stale retained container,
  both empty and carrying one old Item. It checks the first SAV wire, cold
  World and second SAV against a genuinely present control. The one-Item
  control exposed a SAVE refusal on the first candidate; it now writes SAV
  without retaining the stale Item.
- `TestReleaseCurrentDeadContainerUsesCurrentWorld` checks all 43 dead actors
  in the exact owner AGS on the lawful EN and RU installs. It compares first
  SAV wire, cold World and second SAV wire, including the 42 stale records.

## Open debt

The original game's LOAD and gameplay use of these newly written SAVs remain
Unknown. Other raw `Extra` backing differences in the M7 audit have separate
owners and are not covered by this container result.
