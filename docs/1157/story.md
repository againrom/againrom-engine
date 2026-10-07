# Projectile source acceptance

Original mission LOAD is compared with independently read Projectile state.
`TestMilestone2Projectiles` reads the YA1 record table and pool from `File.Store`;
the production Projectile decoder supplies no expected values or population.
It compares both the imported World carrier and the complete retained Document
before Snapshot can project current values over the source.

## Authority and scope

`REG-FMT-031`, `REG-REC-032`, `REG-KIND-033`, `REG-KIND-034` and amended
`REG-VAL-028` define framing, child ranges, type flags and value storage.
`SAV-PROJSTORE-428` defines the allocator, ordered IDs and sixteen integer
leaves per Projectile. `SAV-PROJLOAD-429` supplies optional allocator/IDs and
the integer singleton compatibility arm. `SAV-PROJCORP-430` identifies the
one distinct nonempty authentic source; it does not establish an allocator
rule from that single example.

The raw reader bounds table/pool extents, depth and copied array bytes. Every
record must be reachable exactly once. Selected directory kinds, leaf kinds,
presence, full integer values and array bytes are compared with Document.
World comparisons use the loader's low sixteen bits for allocator and IDs,
retain ID order and multiplicity, and compare all sixteen opaque fields in
one item per distinct ID. The two representations have separate comparisons:
losing source ID high bits can leave World state unchanged.

The instrument covers Projectiles and decimal Prj sections. Other application
values, unused framing words and byte-for-byte pool layout are outside it.
City files are counted separately and have no live mission comparison here.
The reader's expected zero for an absent Prj leaf is this engine's declared
policy, not a claim about original constructor defaults.

## Proof

Focused EN/RU corpus runs discover 102 paths: 62 world, 39 city and one named
invalid-header file. All 62 worlds resume; 60 have no Projectile items. The
two nonempty paths contain the same authentic shape. In total the comparison
checks 2 ordered IDs, 2 items, 156 raw leaves and 384 value bytes, with no raw,
World or retained-Document differences and no resume refusal.

The registered `TestReleaseMilestone2Projectiles1157` uses preserved
`2026-08-15/game0018.sav`, SHA256
`1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b`.
It verifies literal allocator 267, ID 266 and 18 leaves, then drives both App
original LOAD doors, ordinary menu SAVE, removal of the private source copy,
fresh FrontEnd native LOAD and twenty individually advancing equal World
hashes. Explicit state checks accompany the hashes on both roots.

Synthetic App witnesses cover two distinct IDs with a repeated first ID,
full-width source values, singleton IDs, empty IDs and independently absent
allocator/IDs/section. Controls reject every lost World field, altered ID
order or multiplicity, item identity collapse, each changed/deleted Document
leaf, missing selected directories and unavailable Documents. Every truncated
raw prefix and seven malformed framing/type cases are rejected.

Two private compiling mutations discriminate the new acceptance. Replacing
the shared production decoder's X with zero leaves the old 1133 corpus audit
green but fails the independent raw-to-World comparison. Flipping only a
retained allocator high bit during native SAVE preserves the World hash but
fails the fresh Document comparison. The unmodified baseline passes again.
Reproduction scripts and focused logs are in private `review/story1157/`.
Final gates and review follow the earlier 1154..1156 landings; the candidate
incorporates their reconciled acceptance instruments and Group correction.

## Open boundaries

DIV-944 remains open: the restored carrier has no persistent live Projectile
driver, so this story establishes retention and native continuation only.
DIV-945 retains the single authentic nonempty shape and missing constructor
evidence. A synthetic partial Prj leaf set resumes with zero defaults but
makes the complete Document unavailable; a separate boundary test requires
the full-retention comparator to reject it. That shape is not counted as a
passing full-retention witness and was not found in the natural corpus.
No original executable was run. This is no new ROM1 round-trip witness.
