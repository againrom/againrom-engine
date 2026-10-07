# Named saves and a detached return to town

## Result

SAVE and F2 open a dialog with a directory browser, save list and user name.
Town offers AGS, SAV or BOTH; a mission offers AGS or SAV. Each occupied target
needs confirmation of its exact path, including files that are not valid saves.
Cancel writes nothing. LOAD follows the directory of the last successful save.
Automatic tool writers keep their existing collision protection.
AGS retains Unicode names and labels. LOAD draws through the installed font;
RU renders Cyrillic, while EN shows an ASCII fallback for unsupported letters.
SAV labels explicitly require printable ASCII because the original label's
encoding remains unpromoted; a Unicode directory is still allowed.

Mission AGS is the exact current map checkpoint. Mission SAV is a detached
ordinary return to town: survivor cleanup restores HP and mana, while current
XP, possessions and purse survive. The mission remains unfinished and available
to restart. A pending town-arrival companion for the current chapter is created
on the copy with its ordinary grant latch; no victory, objective reward,
payment, next-chapter unlock or live state changes occur (DIV-1230).
City export requires the current campaign to have reached its town boundary
(main chapter 30 in the shipped EN/RU campaign). Earlier chapters refuse SAV
before creating a destination; AGS remains an explicit separate choice.

## Implementation

The UI receives scalar request/list values and a prepared commit callback.
One validated basename removes the optional format suffix exactly once, so
selecting `Chapter.ags.ags` confirms and replaces that same file.
Both town payloads are encoded from one captured state before any publication.
Named replacement uses staged, synced files and backups; a publication failure
rolls back earlier paths. Target content/identity and the resolved directory
are rechecked after confirmation. Original/install write fences apply to both
formats. A crash or a failed rollback I/O operation cannot provide a filesystem
transaction across two names; recovery backups are retained and named if
rollback fails. Automatic no-replace writers remain separate.

The mission copy uses actual party/entity bindings before CarryRoster clears
item handles. Imported-town lineage keeps its immutable city source and current
return graph. Direct original missions project current fields from their own
complete World document into a city; they never become native constructors.
Native Humans keep genuine future derive inputs alongside independently
maintained current pools and exact XP. Current typed load, count, ordered pack,
book fields and owned item children are preserved (DIV-1231). Current AGS gob
DTOs and historical envelopes are unchanged.

## Proof and limits

Focused storage tests inject a failed second publication and cover overwrite,
corrupt targets, cancellation, names and confirmation races. UI tests and EN/RU
installed-font captures cover list/edit/format/confirmation routing and layout.
The version-9 scenario drives GUI NEW GAME through mission AGS, mission city
SAV, cold town LOAD, BOTH, overwrite and cancel. The one-hero M30 start creates
its ordinary pending companion once and writes a stable second city save.

Registered release witnesses cover native missions, imported-town M30 with
actual armor pickup, direct original mission31 with actual Sack acquisition,
and town BOTH. They compare the live snapshot, current pools, XP, ordered item
owners/counts and purse, then cold-load and restart the unfinished mission.
Applied damage distinguishes restored city pools from the live injured actor.
Three following driver ticks under the same move order match complete World
bytes from an independent AGS restore. Only the unordered Commanded map-set
enumeration is sorted when comparing driver DTOs; World bytes stay unchanged.
Read-only played M100 additionally proves four current members and maintained
pool values that differ from their next XP-based derive result. Native field
controls cover raw BookPresent slots, signed load/index/accumulator, current
secondary damage/timing, input immutability and unsupported source rejection.
The real pre-town M20 active-effect fixture refuses CITY before publication,
keeps the live state and cold-loads through an explicitly selected AGS instead.
Repeated suffixes cover row selection, exact-path confirmation, cancel,
replacement and cold LOAD for AGS, SAV and BOTH.

Unsupported source graphs still refuse the requested SAV before writing; they
never silently switch format. Original processes and install writes are outside
this result. The seat owns the sole review, final combined gates and exact-binary
post-review receipts.
