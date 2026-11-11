# Mod project writer and manifest fields

## Intent

Edit the manifest in W and save every editor change through one staged project
writer. Adding a speaker is one draft edit with Undo/Redo.

## Authority

Owner direction: mod workbench P5, with the mod structure specification taking
precedence. No original-game behaviour changes. No divergence is needed.

## As-built behaviour

`mod.WriteManifest` sets top-level keys while preserving every byte outside the
changed values. It preserves BOM, line endings, comments, unknown source keys,
tables, key order and an existing entry. `WriteManifestTitle` delegates to it.
The ordinary `ParseManifest` keeps the accepted manifest population. It reads
string metadata and ignores other documentary values, as before. Its authoring
option checks editable metadata and dependency syntax with the same parser and
the existing ID and constraint readers. Only editor fields and save admission
use that option; ordinary activation retains its own checks.

The existing fields page of W edits version, author, description, applies-to,
requires, conflicts and load-after. The base field has six choices. Dependency
fields take comma-separated typed IDs, with optional requires constraints.
Re-picking a displayed source value outside the six choices changes no bytes
and adds no history step. ID stays fixed after open. API is 1 and hidden. Manifest changes share the
project workspace and history, including a mod with no map open.

`writeEditorProject`, in `mapedittestplay_files.go`, calls the existing
`stageMissionPackage` and `installMissionPackage`. W save, SaveIntoMod and
speaker persistence and Save As into a mod use it. Existing mod folders receive
only changed files, each staged beside its target and replaced through one
`hostfile.Set`, with backups for rollback. Unwritten files keep their bytes,
attributes and modification times. Conflict checks compare only write targets
with their read stamps. Admission uses the current admitted tree with those
changes overlaid in a private validation stage; it never copies documentary
or hidden files for an in-place save. Handwritten mods gain no ownership record
and stay handwritten. W imports still require a new working copy. Ownership
records list editor-written paths, not a census of the author's folder.
Existing path fences and destination refusals remain. A confirmed save with report errors stages the map
while retaining other pending edits and the existing partial-save note.

Project staging uses the authoring option of the same `modrt.Load`. It reads
data and runs scripts in the editor's base context without activating the
author's target or companions. Normal open reports, F5, Publish and game launch
keep activation checks. No data decoder or applier is duplicated.

AddSpeaker uses `mod.AppendSpeaker`, retains the row and face in the project,
and previews them through the existing admitted fork and `SetModData`. Undo
and Redo rebuild that context. Neither AddSpeaker nor its preview writes a file.

An opened mod binds to an existing workspace through the same project builder,
keeping pending pages and their history. Save resolves the selected mission
from the draft, including an added map not yet on disk. Per-file SHA-256 read
stamps bound conflict-check memory independently of file sizes. F5 and Publish
keep their runtime content-digest check. Whole-folder publication remains only
for a new copy and for the disposable F5 folder, whose previous recorded files
must be replaced as a set. Both use the same stage/install writer.

New mission packages retain the old manifest defaults and key order. The only
default manifest-byte change is removal of `entry = "main.star"\n` (20 bytes).
Their 66-byte generated script is absent. Base dumps retain their three-line
comment and values, use unaligned `key = value` assignments, omit entry and
the generated script. The ROM2 documentary town note now says the file is for
reading only; it no longer refers to the removed script.
The new package's ownership list loses the 12 bytes naming that script. A
base-dump manifest loses 23 padding spaces and its 25-byte entry line (48
bytes total); the removed scripts are 99 bytes on ROM1 and 140 bytes on ROM2.
Existing author scripts are kept unchanged. No new script is generated.

Kinds and builders: manifest (`mod.ParseManifest`, `mod.WriteManifest`); tree
paths (`mod.dataKinds`); project writes (stage/install in
`mapedittestplay_files.go`); W fields (WorkspaceFields/modFields); draft/history
(`mapInspectionEdits`); speaker rows (`mod.AppendSpeaker`); mission rows
(`mod.WriteMissions`); context (`admittedFork`/`SetModData`); check report
(`ModCheckLines`). No game identity branch is added.

## Proof

Synthetic tests cover each manifest key's surrounding bytes, insertion before
tables, multiline lists, round trips, entry preservation and reader messages.
Game tests cover manifest draft edits and undo, one project save, speaker undo
and delayed persistence, and default package bytes. Seven modes rows exercise
the new fields through W's existing field painter. A static architecture guard
rejects direct filesystem writes in the editor's mod producers.
The speaker undo assertion counts the one added step above the fixture's
existing page edit and compares the whole draft before Undo and after Redo.
A pure workspace test covers late mod binding without losing pending pages
or changing the existing history count.
Additional synthetic tests cover unrelated external edits, held folders/files,
read-only attributes and timestamps, 1300 git objects and 4200 documentary files,
changed-target conflicts, sibling staging and rollback, Save As admission and
ordinary-load compatibility. Every choice of every manifest choice field is
walked from all six starting targets, a multi-target list and an unknown target;
changed output reaches ordinary load on the selected base. Speaker Undo/Save
also checks that its empty picture folder is removed. The architecture guard
covers map editor, inspection, page and speaker producers outside the writer.

The lane's notes contain command receipts. Folder-based focused tests reached
the sandbox's existing physical-path refusals (`Access is denied`); they were
not retried. The seat owes those tests and the fast gate outside this sandbox.

Release evidence owed on EN/RU: TestReleaseEditorPublishedModChainSAV,
TestReleaseMapEditorOwnerFrames, TestReleaseMapEditorSpeakerPickNamesWhyEveryRefusedUnit,
TestReleaseDialogueThreeSpeakersTestPlay and the three TestRelease*BaseDump*
tests. The map-editor modes walk remains a landing obligation. No release test
was run in the lane.

## Open debt

P6 owns the mission layout, extended maps, and reconciliation of the non-editor
AssembleModMission and writeNewFiles routes. Neither is a call-only migration:
the former preserves partial writes and the latter refuses every existing
destination and accepts documentary data the game does not apply.
T4 owns generic asset import and speaker-face migration into assets/speakers.
P15 still owes an installed witness for adding a speaker. Sandbox-blocked
folder and byte-comparison witnesses remain unproved here.
A changed multi-line manifest list is written on one line and loses comments
inside the old list. Bytes outside that changed value remain preserved.
