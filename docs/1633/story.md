# Long names in a packed mod

## Intent

A mod that works as a folder must publish with its file and directory names
intact. This includes `data/weapon-bodies.toml`, long map stems and their
text folders. A component may hold at most 255 UTF-8 bytes.

## Authority

Owner direction accepts the fixed mod tree and decision O1(b): carry a name
table inside the existing RES container. This is Againrom's mod format;
the original game does not read it. No divergence row is required.

## As-built behaviour

`mod.Pack` writes short RES names only where a component exceeds 15 UTF-8
bytes or cannot be encoded in CP866. It writes one root `mod-names.json`
file with version 1 and entries pairing each raw RES path with the original
UTF-8 component. Short names avoid folded sibling collisions. The name table
is bounded at 16 MiB. `docs/MODS.md` specifies the format.

`newPackedTree` reads and validates the table once, restores names before
building its index and hides the table from every consumer. The existing
`checkKindNames` validation refuses the reserved root name in a folder mod.
Malformed tables, invalid or oversized components and restored folded twins
are refused. Existing packs without a table keep their tree and digest.
A pack needing no renaming keeps its previous bytes.

| Kind | Builder |
|---|---|
| Packed mod writer and name table producer | `mod.Pack` |
| Packed tree and name table reader | `newPackedTree` |
| Mod file name admission | `checkKindNames` |
| RES container | Existing `res.Write` and RES readers, unchanged |
| Mod digest and set | Existing `Tree.Digest`, `ContentDigest` and `Set`, unchanged |
| Save mark | Existing `pkg/game/modmark.go`, unchanged |
| Mod check report | Existing command report and mod runtime, unchanged; tests exercise restored script paths |

No game identity branch or second builder was added. The editor, version
files, knowledge pin and install RES handling are unchanged.

## Proof

The regression first fails on an ASCII component of 16 bytes through
`mod.Pack`. Synthetic tests then compare the folder and packed tree through
`WalkDir`, `FS` and `ReadInside`, and compare digests. They cover every byte
length from 16 through 64, ASCII and Cyrillic, at the root and under `maps/`
and `data/`, nested renamed directories, an alias collision, names outside
CP866, the 255-byte boundary and a 256-byte refusal. They also check packing
determinism, reserved-name collisions, folded twins and malformed tables.

A test constructs the old RES header, payloads and node records without a
name table. Its tree and independently calculated digest match; repacking
its short-name folder gives those exact bytes.

CLI tests publish `data/weapon-bodies.toml` and compare `-check` output for
a folder and pack whose script loads a long module path. Runtime tests
compare the weapon-body data and canonical mod set of both forms. The
recorded merged-data fixture now expects the packed long-name weapon data
and mod mark to equal the folder's, on both languages. The comment-byte
baseline falls from 10542981 to 10542722.

`TestReleasePackedModPlaysAsItsFolder` now loads a module with a long
Cyrillic name. It compares rendered results, World state, town and mission
SAV bytes and mod marks, and loads the folder's SAV under the packed mod.
The named final-candidate runs use the first game's EN and RU roots.

## Review correction

A folder now finds a requested path through its own listing under
`namefold.ModTree`, in `Tree.host`, instead of trusting the host's folding. A
path that Windows would open only by folding a letter outside A-Z and CP866
Cyrillic (`éclair.star` for `Éclair.star`) is "no such file" in the folder, as
in the pack. `TestFolderAndPackedMatchScriptNamesByOneRule` failed before the
change on the three outside-the-rule cases and passes now; no digest or pack
byte moved.

## Open debt

The seat runs the fast gate outside this sandbox, pushes the candidate and
obtains the sole review. No fast gate or network operation runs in this
lane. Final gate results and any sandbox-denied witnesses are recorded in
the lane return. The next release owes the same named packed-mod witness on
EN and RU; the lane's selected run does not establish a release pass.
