# Fixed mod tree

## Intent

A mod's held files define its data without a script. One description defines the
tree for loading, checking, packing and hashing, with old row paths kept readable.

## Authority

Owner direction adopts the fixed tree and D2: read each data kind when its file is
held. Existing manifest entries and old paths remain supported. The mod format is
Againrom's own; no original-game behaviour or research pin changes. DIV-4846 through
DIV-4849 are unused.

## As-built behaviour

`pkg/mod/kinds.go` extends `dataKinds` to 24 rows: the twelve existing data kinds
and twelve non-data kinds. It holds paths, formats, language and discovery rules,
state policy and row-reference metadata. `DataKinds` is the data view; `TreeKinds`
is the complete view. Fixed reader names and install-file extension classification
come from the same file and table.

Language admission uses `locale.Known`; language enumeration and the table's
language text use `locale.All`. The normative document test compares that text
with the table. The tree owns `MapLabelFile` because both admission and the editor
need the name and `pkg/mod` cannot depend on `pkg/game`. The editor keeps the JSON
reader and writer and reads the shared name; the map-label guard names both homes.

The screen-art guard admits only picture admission globs in `pkg/mod/kinds.go`.
Its allowlist row names that reason. Concrete picture names and decoder calls in
the table remain findings. The table opens no picture. Mission-page warnings use
`Mission.ReadPageLayer` and `ResolvePages`, the runtime's existing page resolver.

`Tree.Describe` joins the table with held files and paths named by rows, including
legacy mission, page and asset paths and an existing manifest entry. Town JSON
structure uses `town.Parse`, also used by `town.Decode`. `TownArtFiles` in
`pkg/mod/towns.go` resolves town art for discovery and the runtime reader through
`Description.ArtEntries` and `screenart.Entry.Keys`. It keeps literal names and
exact series/index members. `modFilePath` in `pkg/mod/tree.go` resolves literal
references for admission and `ReadInside`. Only mission references use
`missionPath` in `pkg/mod/missions.go`, shared with the mission reader.
Packer admission and digest admission
use this description. Dotted names are ignored; the read-me packs without entering
the digest. Pack validates install files through the existing `ReadInstallFiles`
before filtering, so a malformed install-file tree refuses as its folder does.
Migration warnings compare cleaned map/text paths and retain the language folder
in a legacy map's suggested path. The packed logical-name codec is unchanged.

`readBesideManifest` loads all twelve held data kinds before optional script init.
Its file-read step is also the explicit-add route. A first explicit add confirms
the held file; a second add, including another case spelling, refuses. Resolved
settings are available to data readers before scripts run. Named mission discovery
still uses `readMissionMaps` after mission rows; parsing and admission are unchanged.

Holding a data file except settings sets the state mark, even for empty data or
speakers. The save-mark producer is unchanged. An existing speaker-only mod that
previously added no file now enters the save mark. A save under its old state-free
set can therefore report an added mod. Excluding previously hashed unread files
also changes those mods' digests once; old saves then report different files.

The shared `ModCheckLines` report appends tree warnings. The command prints those
lines on standard error; the editor sees the same report. Warnings cover unread
files, the second weapon-bodies name, legacy mission map/text paths and title or
briefing keys overriding pages. Asset-reference and unused/missing-map warnings
remain with T4 and T3. The normative MODS table has an exact table-comparison test.

## Proof

Synthetic tests reproduce missing held-data loading, then cover script-free state,
conditional scripts, malformed held data, explicit-add idempotence and duplicates,
each new warning, legacy references and stray-file exclusion. A tree covering all
24 rows preserves the previous pack bytes and independently reconstructed digest;
folder and packed reads agree. The existing logical-name tests now use admitted
tree paths, preserving their codec coverage.

All 18 fixture mods, completed with existing synthetic fixture completions, have
equal `Data` with and without scripts on ROM1 EN and RU. Removing a script also
removes its explicit manifest entry; a retained explicit entry remains required.
All 18 fixture content digests equal the previous unfiltered digest. The independent
merged-data record changes only six mark lines: speakers alone, folder/packed, and
the full fixture set, on each language. Existing data and refusal lines stay equal.

Fixture check lines remain equal to the frozen report after removing new warnings:
four for guest-chain, two for reniesta, one for war-hammer-body on each language.
The incomplete east-town fixture keeps its prior refusal. The static architecture
guard forbids fixed tree path constants outside the table, allowing the separately
owned editor profile folder. Focused and final gate receipts are in the lane return.

Correction tests cover every locale row's fixed and legacy referenced paths,
unknown-language exclusions, map-label stem preservation and the screen-art
allowlist's refusal of concrete names and decoder calls. Existing warning tests
cover shared pages and other-language fallback through the page resolver.

Six reference regressions reproduce on the reviewed head: town art with a dot
segment or brackets, and a sprite, either body sheet or portrait with a literal
language token. Each compares folder and pack descriptions, file lists, digests
and read bytes, requires the file in the pack and no check warning on either
language, and changes the file to prove it enters the digest. Series/index tests
require exact members and exclude unreferenced siblings. The source guard pins
the shared resolvers. Install-file refusals and migration warning tests cover the
three local optional corrections. The accepted state, held-data and unread-file
policies remain unchanged.

## Open debt

The sandbox lane runs no fast gate or installed release witness. The seat owes the
fast gate, publication and landing, and the named mod release tests on EN and RU.
Installed behaviour and SAV reload are not established by synthetic equality.

R3 remains open for T3: join held maps with row and numbered-file declarations in
one map list, and make the editor inventory, mission/kind lists and applied report
views of that description. Editor appliers, W and mission writers were outside this
lane. Their legacy script and layout output remains for P5/P6; existing output loads.

The spec's conditional 15-byte warning no longer applies: the existing logical-name
table already supports 255-byte components. The spec's 16 count is the redundant
scripts; the inspected population is 18 fixtures, and all eighteen received the
script-free Data comparison. No owner mod material or preserved install was read.
