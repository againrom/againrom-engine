# Trailer state survives mission SAVE

The acceptance instrument compares all 100 raw trailer dwords with the
complete retained Document after original LOAD. An ordinary App witness
checks title LOAD, mission-menu LOAD, menu SAVE, a fresh FrontEnd LOAD and
20 advancing continuation hashes. The private original copy is removed
before fresh LOAD. It compares retained trailer state again
after continuation; the World hash alone cannot prove its survival.

## Authority and scope

SAV-790 transfers all 400 bytes after either marker branch. SAV-DECPAD-238
permits one arbitrary alignment byte after an odd logical end. TrailerOff
exposes only the structure start. Expected values are independent little
endian reads from Body, not File.Trailer or DecodeDocumentData.

SAV-647, amended by SAV-694, names two trace controls. SAV-791 leaves other
consumers unresolved. This story retains bytes without inventing a live
consumer. DIV-968 remains open for trace activation. City documents and an
original-format mission writer remain outside this mission milestone.

## Proof

Synthetic App inputs carry 100 distinct nonzero dwords through both marker
branches and both alignment parities. Controls reject each missing dword,
swapped leading dwords, missing Document state, wrong offsets and extents.
The discovered corpus instrument is TestMilestone2Trailer. The registered
EN/RU witness TestReleaseMilestone2Trailer uses an unchanged owner input
and a private derived nonzero variant. Neither executes the original game.

Focused EN/RU discovery compares 6200 raw dwords across 62 resumed mission
files with zero differences. It separately names 39 city exclusions and
the one unreadable game9000 input. All 62 mission trailers are zero, so
nonzero App fixtures are required. Compiling overlays that drop dword 99,
swap dwords 0/1, or locate the trailer at len(Body)-400 all fail the new
App witness. Evidence is private under review/story1153 at the seat.
An additional compiling native SAVE mutation drops only dword 99: fresh
LOAD still has the same World hash, but the retained trailer check fails.
