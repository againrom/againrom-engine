# Current city quick-spell bindings

Town SAVE writes current F5-F8 bindings to SAV for imported and generated
cities. Changing a binding after SAV LOAD stays on the SAV path. Explicit
city conversion uses the same current-state update.

## Authority and implementation

- `AI-QUICKSAVE-281`: kind6, 16 payload bytes, four signed controller indices
  in F5-F8 order; `-1` means unbound. Pressed and IsOpen are separate records.
- `AI-SPELLIDENT-286`: the 24-cell map includes real ID23 at index5 and ID18
  at index23. Both exporters call one inverse mapping.
- `sav.CityUpdate.Shortcuts` is an optional `*CityShortcuts`. Nil preserves
  the source record. A present array replaces all four words. Out-of-range
  or duplicate nonnegative indices refuse before any document mutation.
- The imported writer retains its source graph, party bindings and unrelated
  baseline guards. This adds one mapped field; it does not relax the generic
  party or campaign comparisons. Original provenance remains independently
  validated when an AGS is loaded.
- No AGS schema or simulation form changes. Existing and legacy AGS bindings
  keep their real-ID representation. IDs outside the 24-entry map still use
  ordinary AGS fallback; explicit SAV conversion refuses them.

## Proof

`TestReleaseCityQuickSpellsCurrentWriters1167` runs over both installed locales,
an imported city and a generated city. The imported input is the read-only
`2026-08-15/game0010.sav`, SHA256
`89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4`.

The same FrontEnd's production mission opener attaches its live session slots
to an installed App/Viewer. A controlled mage/book/mana drives real hover and
Ctrl-F5/F6/F8 keys. The probe does not claim a played campaign or town arrival.
Current chapter documents are collected before measurement; a whole town
snapshot comparison excludes every key-induced change except QuickSpells.
Ordinary town SAVE uses its real callback with `onMap=false`.

The independent physical YA1 table/pool oracle requires kind6, 16 bytes and
literal `[5,23,-1,7]` for current real IDs `[23,18,0,16]`. Body, campaign and
every other state-store byte match the unchanged writer control. The private
source copy is removed. A fresh FrontEnd loads the authored SAV without an
OriginalStore. F5/Ctrl-F7 moves spell23 and clears F5; the second ordinary SAV
contains `[-1,23,5,7]`. A second fresh LOAD retains that change. F8 targets
spell16 and spends mana after both LOAD boundaries. A separate current-state
control clears all four bindings and verifies four `-1` words through conversion.

The shared-writer test covers nil preservation, positive updates, complete
clearing, unrelated state, deterministic bytes and five atomic malformed
updates. The inverse-map test checks all 24 literal pairs and seven refusal
cases. Existing installed AGS fallback controls now use custom ID65535.
The registered three-process city conversion witness also carries mapped
bindings, changes the current AGS bindings after SAV LOAD and checks the
second source-backed SAV's literal indices.

## Remaining scope

Populated-slot original-runtime LOAD and invalid-value handling remain
Unknown. Only engine EN/RU continuation is claimed. Current selected spell,
Cast mode and book visibility remain transient. Moving a duplicate clears its
old key under the existing UI rule; no new clear gesture is introduced.
Offer state, campaign completion, roster/item graph edits and world SAV writing
are outside this story. DIV-827 and DIV-878 retain the remaining scope;
DIV-1153 through DIV-1160 are unused.

See [verification](verification.md) for the branch evidence and seat gates.
