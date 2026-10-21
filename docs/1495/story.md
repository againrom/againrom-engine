# A mod as one RES file

## Intent

A mod ships as its folder or as one file, `mods/mod-<id>.res`. The game reads the packed file as
the folder it was packed from: the same manifest, scripts, settings, data files, texts and
`files/`, under the same checks. A pack command writes the file from a mod folder. A save made
with one form loads with the other.

## Authority

Owner decision T5b of the mod track (`pipeline/BACKLOG.md`): keep the folder form and add the
packed form; RES only, no ZIP; the pack refuses a path component over the RES name bound.

## As built

### One mod-tree source

`pkg/mod/tree.go` holds `mod.Tree`, the files of one mod. `OpenTree(path)` opens a folder, or a
regular file as a packed RES; a path that is not a regular file stays a folder, so a missing
folder fails where it is read, as before. The folder backend is the code the readers ran before,
moved: `os.ReadFile`, `os.Stat`, `os.Lstat`, `os.ReadDir`, `os.DirFS`, `filepath.WalkDir`, and
`ReadInside`'s confinement through `resolvedPath`. The RES backend (`packedTree`) indexes the
archive's nodes by their stored names and serves the same calls; it implements `fs.FS`,
`fs.ReadDirFS`, `fs.ReadFileFS` and `fs.StatFS`. A name matches without case, as a folder's
name does on Windows: ASCII `A`-`Z` and the CP866 Cyrillic capitals (`А`-`Я`, `Ё`, `Є`, `Ї`, `Ў`) fold to
their small letters (`foldKey`), in the index and in the query, inside the mod tree only; the
game's own reading of an install's RES files keeps the original's ASCII comparator
(RES-LOOKUP-023). Every tree reader (`ReadInside`, `ReadFile`, `Stat`, `Lstat`, `ReadDir`) refuses a
path with a component starting with a dot, for a folder and a RES alike, with
`<path>: a name starting with a dot is not read from a mod`. Opening refuses a RES holding an empty, `.` or `..` name, a name with a
slash or backslash, or two names of one folder that are one name under `foldKey`.

A mod's location stays a string (`Entry.Dir`, and `Dir` of item rows, body sheets, town records
and town descriptions): the folder or the RES file. Every reader opens the tree at that string:

| reader | before | through the tree |
|---|---|---|
| `mod.Load` (manifest) | `os.ReadFile(dir/mod.toml)` | `Tree.ReadFile` |
| `mod.LoadSettings` | `os.ReadFile(dir/settings.toml)` | `Tree.ReadFile` |
| `mod.ContentDigest` | `filepath.WalkDir`, `os.ReadFile` | `Tree.Digest` (`WalkDir`, `ReadFile`) |
| `mod.ReadInside` (scripts, `load()`, data files, sprites, body sheets, town art, map texts) | `os` with `resolvedPath` | `Tree.ReadInside` |
| `mod.LoadStrings`, `mod.TextLookup` | `os.Lstat`, `ReadInside` | `Tree.Lstat`, `Tree.ReadInside` |
| `mod.ReadInstallFiles` | `os.Lstat`, `os.DirFS` | `Tree.Lstat`, `Tree.FS` |
| `mod.Mission.ReadTexts` | `os.ReadDir`, `ReadInside` | `Tree.ReadDir`, `Tree.ReadInside` |
| `modrt` entry default (`main.star`) | `os.Stat` | `Tree.Stat` |
| `pkg/game` deferred reads: `moditems.go` sprites, `modbodies.go` sheets, `townlayer.go` art, `modmapsave.go` missions and texts | `mod.ReadInside(dir, …)` | unchanged calls, now through the tree |
| `cmd/starter` settings | `mod.LoadSettings(e.Dir)` | unchanged call, through the tree |

`digest.go` changed in `ContentDigest`'s body only, which now opens the tree; `Differences` is
untouched. The map editor stays folder-only: `MapInspector.SetMod` refuses a packed mod, since
the editor saves into the folder, and the editor's page reader `readPageTree` also reads install
map folders, which are no mod.

### Mods directory

`mod.Scan` lists the subfolders holding `mod.toml` and the files whose name, case folded, is
`mod*.res`. A packed file must be named `mod-<id>.res` with a valid id, and its manifest id must
be that `<id>`; otherwise it is listed with an error naming the file. A folder and a packed file of
one id are both listed with the error `mod "<id>" is both <folder> and <file>; keep one of them`.
`mod.Resolve` (`-mods`) looks for `<id>/` and `mod-<id>.res`, refuses both with the same error, and
names both when neither exists. A folder sorts before the packed file of its id.

### RES writer

`pkg/formats/res/write.go` holds `res.Write`, the writer the format's write sequence describes
(`knowledge/formats/res/format.md`, "Write sequence"): the 24-byte header, payloads from `0x18`,
then the node array. The top-level nodes come first, so header `0x04` is 0 and `0x08` their
count (RES-HDR-017); header `0x0C` is 1, a directory with the sorted flag clear (RES-HDR-018).
Nodes are assigned breadth first, so each directory's children are one contiguous range
(RES-NODE-008). Children stand in the byte order of their ASCII-folded names, and no directory
carries the sorted flag: the lookup then scans and folds ASCII (RES-041), so a stored capital
letter does not hide a file. The reserved word is 0 and a name is followed by NUL bytes.

Name rule, `res.EncodeName`: a name is written in CP866, the code page the engine's reader
decodes names with (`decodeName`); an ASCII character is itself and a Cyrillic letter one byte.
A name is refused when it is empty, `.` or `..`, holds a slash, a backslash or a control
character, holds a character CP866 has no byte for, or is longer than 15 bytes once encoded: the
name field is 16 bytes with its NUL, and the original's writer and lookup bound a name at 15
characters (RES-NODE-007, RES-LOOKUP-023). Two names of one folder that differ only by ASCII case
(the writer), two paths equal under `foldKey` (`Pack`, naming both), and one path given as a file and a folder, are refused. An archive past 32-bit offsets is refused.

`res.Archive.Tree` and `res.FileArchive.Tree` return the nodes as stored, empty directories
included; `ReadNode` reads one file node. The packed tree uses them: the existing index
(`Entries`, `ReadFile`) folds names and drops directories, and a mod's digest and its install
files' paths keep the spelling the folder has.

### Pack command

`againrom -pack-mod <mod folder> -pack-out <existing folder> [-pack-replace]` runs before any
install is looked for and opens no window. `mod.Pack` loads the folder as a mod (manifest id equal
to the folder name), walks it, and refuses naming the file: a name the writer refuses
(`mod "<id>": <path>: cannot be packed: <reason>`), a link or other non-regular file, a file over
16 MiB (`MaxInstallFileBytes`, the largest any mod reader takes). Names starting with a dot are
left out, as the digest and the install-file reader leave them out; no reader takes one, so the
left-out file cannot change the game. The packed bytes are read
back through the packed tree and must give the folder's digest. `game.PackMod` writes
`<out>/mod-<id>.res`: `-pack-out` must be an existing folder below no folder holding an install's
archives (`requiredArchiveAncestor`); an existing file is refused unless `-pack-replace`, which
writes a temporary file beside it and renames it over. A refused run leaves no file.

### Mod mark

The mod mark records `ContentDigest`, which hashes the files by relative path and content. The
packed tree walks the stored spellings in the same order, so a folder and its RES give one digest,
one mod-set digest and one mark.

### Could the original game read the file

Inference, not tested against the original program. The file follows the format the knowledge
snapshot describes, and the original's reader validates only the magic (format.md, "Header").
The original opens archives only from its own fixed list (RES-SET-032) and never looks in
`mods/`, so it would not read a packed mod as a mod. A Cyrillic name is CP866 bytes; the
original converts no name bytes (RES-TEXT-021), and every shipped name is ASCII (RES-TEXT-022),
so how the original would match such a name is Unknown.

## Proof

| proof | test | install | loss control |
|---|---|---|---|
| the fixture mod (settings, a rule set from a setting, a picture, a sound, texts per language) from its folder and from its packed RES: menu picture, sound, briefing, town square and world map headless frames, home title, town and mission SAV bytes with the mod mark, mission 10's World bytes and hash equal; the folder's town SAV loads under the packed mod | `TestReleasePackedModPlaysAsItsFolder` | EN, RU | a session without the mod must differ in the menu picture and the home title; with the packed reader dropping `files/` the test fails at the mod-set comparison (digest, 0 install files) |
| every reader answers alike for folder and RES: manifest, settings, digest, install files on both languages, strings, mission pages, script reads and their refusals | `TestPackedModReadsAsItsFolder` (`pkg/mod`) | none | a second folder with one changed text must differ; with the packed reader dropping `files/` or `data/` the test fails |
| a mod run from folder and RES gives one result: rules from a setting, set digest, items, towns, install files, on `rom1-en` and `rom1-ru` | `TestPackedModRunsAsItsFolder` (`pkg/modrt`) | none | without the setting flag the rule and digest differ; dropping `files/` fails at the counts, dropping `data/` at `game.data.add` |
| a folder rule refuses the RES in the same words (an empty `files/de`) | `TestPackedModIsHeldToTheFolderRules` | none | refusal text compared |
| digest is the documented form for both forms | `TestDigestIsTheDocumentedFormForFolderAndPackedFile` | none | oracle built in the test |
| a script path the folder loads loads from the RES: a dotted script or folder is refused by both naming the path; `карта.star` loaded as `КАРТА.star`, `ёлка.star` as `ЁЛКА.star` load in both | `TestPackedModLoadsScriptPathsAsItsFolder` (`pkg/modrt`) | none | with the fold limited to ASCII the Cyrillic cases fail in the RES; with the dot refusal removed the dotted cases load from the folder and fail only in the RES |
| every reader refuses a dotted path in folder and RES; case twins under the full fold are refused at pack naming both and at open | `TestDottedPathsAreRefusedByEveryReader`, `TestPackedNamesFoldEveryLetterARESCanHold` (`pkg/mod`) | none | without the refusal or the fold the tests fail |
| pack refusal names the file: 18- and 20-byte names, a character without a CP866 byte; a Cyrillic name packs and reads back | `TestPackRefusesANameTheResCannotHoldNamingTheFile` | none | exact refusal prefixes |
| folder and RES of one id refused naming both; misnamed, renamed and unreadable RES files refused; other `.res` ignored | `TestFolderAndPackedFileOfOneIdAreRefusedNamingBoth` | none | the packed mod alone resolves |
| `-check` on the RES prints what it prints on the folder, but `dir=`; folder and RES side by side refused | `TestCheckReportsAPackedModAsItsFolder` (`cmd/againrom`) | synthetic | stdout and stderr compared whole |
| `-pack-mod` writes over a file only with `-pack-replace`, never into an install, names a refused file, leaves nothing on refusal | `TestPackModWritesOnlyWhereAskedAndNamesARefusedFile`, `TestPackModFlagsGoTogetherAndExcludeOtherModes` | synthetic | each refusal asserted |
| writer layout pinned byte for byte; read back with case, empty folders, contiguous children, no sorted flag, CP866 bytes; refusals | `pkg/formats/res/write_test.go` | none | literal bytes |

The fixture mods are generated in each test; no install byte is in the repository. Without a mod
no reader changes behaviour: the folder backend is the previous code, and the existing mod and
install-files release tests run unchanged in the release gate.

## Divergences

None. Packed mods are an Againrom mod facility; the original game has no mods. DIV-3901 to
DIV-3908 are unused.

## Open debt

- `data/weapon-bodies.toml` is 18 bytes, over the 15-byte name bound: a mod with weapon bodies
  cannot be packed and stays a folder. Renaming the data file, or an alias, is an owner decision.
- The map editor edits and test-plays mod folders only.
- Case folding in the mod tree covers ASCII and the CP866 Cyrillic letters; another Unicode case
  pair cannot occur in a RES name.
- `internal/synth.Archive` remains a separate test-input builder of RES bytes: it keeps the
  caller's node order and the shipped 0xCD padding so reader tests stay independent of
  `res.Write`.
- Whether the original program reads the written file is untested (see above).
