# Saved Player formation

Original LOAD now applies each saved Player's formation byte to Groups that
explicitly name that Player as owner. Commands and script instants update the
correct receiver. Ordinary native SAVE and source-free LOAD preserve the current
byte and exact owner, including the next real movement.

The candidate contains main `6cbe75c177b904bcf3facd3a0c91c28cacf2d070` and pins
knowledge k6, `84f328362e6c946303da3d15dddfcefbcfec4946`. It is ready for the
seat's final gates after its one bounded correction. The sole fresh review's P2
owner-conflict finding is corrected; no second fresh review is required. No
landing, stamped-binary witness or install result is claimed here.

## Authority and behavior

The pin's claim reader supplied SAV-PLAYERIDENT-830, SAV-PLAYERPOP-831,
AI-FORMOWNER-314, AI-FORMCMD-315, AI-FORMTRIGGER-316, AI-FORMACTIVE-317,
AI-FORM-037 and its amendment, MOVE-GATE-035, MOVE-FORM-036, SAV-662,
SAV-663, SAV-PLAYER-028, SAV-GRPOWNER-561, SAV-GRPCMD-578,
SAV-GRPSAVENEXT-572 and TRIG-PARAM-030. No private research evidence supplied
implementation behavior.

- Each distinct serializer receiver has an opaque native PlayerID, signed
  command identifier +04, trigger identifier +08 and raw formation byte. Source
  aliases do not duplicate Players; equal-valued distinct Players stay distinct.
- Commands update the first signed-word +04 match in retained Player order.
  Script instant7 writes its raw parameter through a unique +08 match. The client
  cycle reads its command target and retains the existing SelfSlot1 sender.
- Saved Move and Swarm2 read the explicit Group owner. The owner can differ from
  the containing Player and the actor owner. A missing owner distributes no new
  actor destinations. Missing or colliding trigger identifiers suppress the
  formation write; both unsupported boundaries report an issue.
- Native command-created Groups retain the existing container/owner policy.
  Formation identity survives Group creation, GiveUnit and document reindexing.
- Snapshot projects only PRaw32 byte31. The other 31 bytes stay opaque. Native
  LOAD verifies the current byte and independent object bindings; it never
  reconstructs a missing exact carrier from a retained Document.

Original SAV admission still requires +04==+08 and identifiers 1..16. Native
simulation controls distinguish unequal identifiers without asserting original
acceptance. Mode0 disables formation, mode2 uses the spread gate, and every
other nonzero mode forces formation. Offset arithmetic is unchanged.

## Native format and compatibility

Form90 appends a bounded footer after complete form89: ordered 11-byte Player
records and sorted 8-byte Group/owner records, each preceded by a count. A trailing
u32 gives the payload span. Zero span means legacy absence; a present empty
registry has an 8-byte payload. The footer participates in the World hash.

The decoder validates counts, exact Player coverage, command identity, unique
ordered Group IDs and owner membership before replacing the World. Envelope
EncodeSave, DecodeSave and Restore also require the native carrier, current
Document and exact bindings to agree. An unrelated Group export coverage gap
does not excuse an existing Group's changed formation owner: its exact ID,
native owner reference, bound Player and retained G44 must agree. Native-authored
owners retain their typed, zero-source-key representation. An unresolved owner
can retain its original unresolved address or its projected null spelling.

Readers accept form89 and append absent state for every supported predecessor.
Old native saves retain their legacy roster-slot modes even when their retained
Document contains PRaw32. Frozen released predecessor blobs and hashes stay
unchanged. Three current gob-descriptor fixture hashes change with the new
presence field. Form90 length/peel assertions cover sim, map import, save
migration and save repair. The destination architecture guard follows the moved
writer body and retains its victim-clearing checks.

## Proof

`TestSavedFormation1159AppIdentityAndContinuation` enters both title and map-menu
LOAD, compares native state with an independent raw-source reader, executes a
literal scheduled ALM instant and the real client cycle, then uses menu SAVE.
It removes only the private original, loads the native save into a fresh App,
advances both sessions 20 times with equality checked after EACH advancing tick,
issues a new ordinary Move and performs a second menu SAVE. Both imported actors
physically move; the fixture's native placeholder is not counted as a source actor.

The synthetic roots are first/null/first/second/second. First Player has
source key0x1111, nativeID1, command/trigger2, original mode0 and current mode255.
Second has key0x2222, nativeID2, command/trigger1, original mode1 and current mode0.
Group77 belongs to container2 but owner1; its actors retain actor-owner slot1
and start at (12,12)/(14,12). The scripted Move(30,30) targets (29,30)/(31,30).
A later native Group uses the second Player's current off mode and targets
both actors at (20,20). Nonzero residue detects writes beyond byte31.

`TestSavedFormation1159NativePairRefusalsAreAtomic` rejects 19 independent losses
or conflicts at all three envelope/restore boundaries: carrier, byte, Player ID,
command, trigger, owner, Document, bindings and presence. A failed Restore keeps
the running session, World, town and shop unchanged. Sim controls separately
cover signed and duplicate command identifiers, reversed trigger targets,
equal-valued distinct Players, absent/present-empty state and corrupt footers.
Existing Group creation/GiveUnit/reindex witnesses now assert exact formations
and owners, including duplicate semantic slots.

The correction adds coupled OwnerID/binding conflicts, including a matching
native reference with a still-conflicting retained G44. All 57 seam refusals
preserve the running session. `TestSavedFormation1159ValidOwnerCoverageGaps`
passes four complete Encode/Decode/Restore controls: resolved owner, null owner,
unresolved source address and projected null. Its real scheduled next Move keeps
the resolved owner's two formation targets and gives no destination to unresolved
owners.

`TestReleaseSavedFormation1159AppContinuation` is registered in the release
population and passed focused EN and RU runs on both preserved subjects:

| Source relative to the read-only save corpus | SHA256 |
|---|---|
| `2026-08-24/game0021.sav` | `7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c` |
| `2027-09-07/game0036.sav` | `dd45e761d806ffe5d0ad2104d3dd21cd30f5985dcd5695ac47bdf2ce5348b530` |

The 0036 Player+04=1 starts at formation0, distinguishing the previous default2
import. The test compares current exact formations/owners across sessions and
current byte31 plus opaque residue against the original. The second native file
must equal its own writer's complete Snapshot, including the retained Document.

Pre-correction candidate `8cb7e103fca953212d02ce2e8df5e5df4e96d9d5` passed the
complete Go suite and the seat's release/acceptance chain. Those results do not
replace the required final chain on this correction. Correction-focused tests
pass 30 top-level Group/formation tests and 145 subtests without skips;
gofmt and `git diff --check` are clean.

The unchanged private reviewer probe reproduced the two changed destinations
before the correction and now refuses both gap/no-gap inputs at all three seams.
The seat's private `review/story1159/` contains `correction-review-red.log`,
`correction-review-green.log`, `correction-controls-red.log`,
`correction-formation-green.log` and `correction-group-green.log`. The sole report
is `pipeline/reviews/story1159-pass1.md`; its private probes are unchanged.
The deterministic `formation-source.sav` remains 1962 bytes, SHA256
`d997957cbae757ad4b3c46340298a7aed55851dc168bda67280075d0ac5e88fd`.
Allocation sweeps before/after the ledger edit report zero missing answers;
`check-div-claims.sh` passes against k6.

## Open debt and remaining gates

DIV-957 now records only the unresolved complete active-client LOAD lifecycle.
DIV-1090 records native format/presence policy. DIV-1091 names trigger collisions,
null-owner reachability and discordant original admission. None is guessed closed.
DIV-958 still owns the 31 opaque Player residue bytes.

DIV-1092 records a separate current motion producer gap. When imported motion
becomes Current=false, both its projector and validator skip Block12/U154.
Original and fresh native sessions then retain different historical bytes despite
equal current World state. A private overlay reproduces this on pre1159 main
`6cbe75c177b904bcf3facd3a0c91c28cacf2d070`: 0021 has four raw differences on
objects86/87 at tick394; 0036 has four on objects29/91 at tick43. For 0036, first
SAVE is tick22, both sessions advance 20 ticks, then actors39/40 Move(20,20).
The baseline World hashes are respectively 83a446e43bb386f4 and 9dfb42b9f2707b63.
Exact before/after bytes and reproduction metadata remain in
`review/story1159/preexisting-motion/{baseline.log,result.json}` and
`probe-preexisting-motion.py`. Complete cross-session Document equality is not
claimed. A bounded current motion producer remains required; stale blocks are
not forced back into current motion state to conceal this gap.

The seat owns the final full Go, paired release, original-save milestone2,
assets, claims and preserved-install chain on the committed correction.
The fresh-ALM census population uses absent exact formation state, so its legacy
dispatch is unchanged; original-save continuation is the changed population and
requires its acceptance gate. No broad input-routing change is made. After
landing, the seat must run the stamped-binary source-free witness and build from
exact main. Lawful source saves and installs remain read-only.
