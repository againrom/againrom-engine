# Story 1145 — independent Building acceptance

An original mission load must retain the saved Building roster and values.
Existing focused health/ruin tests do not independently compare the whole
discovered corpus. This story adds that acceptance instrument, directly
reading source field bytes and comparing the restored simulation.

Scope: roster, current/maximum health, geometry, attachment mask, Token and
base fields, subclass tails, and saved cell Building identities. The archive
decoder supplies object start offsets, class/index metadata and cell-table
location only. SAV-TOKEN-034, SAV-BLDG-037, SAV-CLASSSER-173/176,
SHOP-SAVE-015 and SAV-CELLLOAD-109/110 define the independent layouts.
Later scalar LOAD writes normalize the overlapping raw base bytes. Two new
generic byte locators on `sav.WorldHalf` delimit the Building root list.
The witness reads its raw count and MFC tags (SAV-STREAM-010/013), including
aliases, so a dropped decoder row cannot shrink both sides silently. Typed
null Building roots remain refused by the existing reader; none is present
in this corpus. Explicit cell-null overwrites are admitted and compared.

The discovered corpus must name every refusal and count each subclass;
zero subjects is absence of an original witness, not a universal claim.
The registered release witness must cover both original LOAD doors,
ordinary native SAVE/fresh LOAD and subsequent native continuation.
No original game is launched and no world SAV writer is introduced.

EN and RU each resume 62 discovered mission files: 1396 Building records,
10657 final cell links (7781 nonzero), and 1396 raw/scalar normalizations.
There are zero mismatches and zero refusals. The corpus has no Outpost,
Tavern or Shop instance in this list; synthetic tails cover those readers.
The alias/last-write fixture and 15 deliberately lost-state controls test
the comparator itself. Extended Outpost counts and truncated tails are checked.
The seat follow-up to review D1 reads the cell-table short/extended count and
payload start directly from bytes; a deliberately wrong decoded count and
a stale payload locator no longer hide the final null overwrite.

The registered release witness uses `game0017.sav`, SHA-256
`eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0`:
11 Buildings and 117 cell links agree through both original LOAD doors,
ordinary menu SAVE and fresh FrontEnd LOAD. A separate memory-only variant
changes raw Token/base values and health to 7/31. Both variants agree through
64 subsequent native driver ticks on each root. Full Go tests pass.

No restored Building state defect was reproduced. The production change
only exposes byte locators; the remaining files add acceptance and controls.
DIV-1009..1014 remain unused. Original subclass services, destruction timing,
opaque reference consumers and original post-load continuation remain the
existing debt of story 1114. This proves source/live retention and native
continuation consistency, not original runtime equivalence. Final paired
gates and the sole adversarial pass belong to the seat landing record.
