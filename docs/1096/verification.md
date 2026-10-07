# Saved spellbook verification

## Player result

The owner source `gameversions/saves/2026-08-24/game0021.sav` stores 28
learned spells for Fergard. Its SHA-256 is
`7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c`.
The story base imported the Humans row-28 template mask `0x00041042`:
four spells. The candidate imports the source mask `0x1ffffffe`: 28 spells.

The EN and RU install-gated production witness loads that source through
App LOAD, gets 28 spellbook rows, casts Protection from Air (ID 16, absent
from the template), saves through ordinary SAVE to AGS, and loads the same
canonical world hash `e89b608715627a3e`. This is a headless production-route
witness, not an observed GUI session or an original-game compatibility run.
Both installs consume the same owner recording.

The expectation is independent of the production party decoder. Pinned
research `savreplay -mode trace` identifies the first Human at body offset
249 and book Spell bodies at `1119 + 11*i`, for `i=0..27`. The test checks
those literal source bytes against IDs 1..28. `MAGIC-SPELL-001` identifies
the ID field; `MAGIC-BOOK-002` identifies the sparse ID-indexed membership.
The baseline runtime probe used the master checkout at `d7606aa1` with
unrelated hotfix edits present; its book code was unchanged and its player
mask was `0x00041042`. The fixture and literal source control do not depend
on that probe's world hash.

## Contract coverage

- The hand-built archive tests distinguish absent, present-empty and sparse
  books, including null slots and a shared archive Spell referenced by two
  characters. The character projections are detached; the record graph
  preserves alias identity and typed ID/range/defensive/mana-cost values.
- Invalid IDs, slot disagreement, wrong classes, unknown references,
  truncated framing and excessive counts return `sav.ErrSpellbook`.
  Original mission and city load refuse malformed books before changing
  the active party, town or snapshot. This is a bounded importer policy,
  not a claim about every malformed file ROM1 accepts.
- Two distinct characters retain their exact learned masks through
  RestoreOriginal, App mission entry, native Encode/Decode/Restore and
  deterministic continuation. A subsequent absent-book load does not keep
  the previous character's learned mask.
- Mission mint does not teach from equipment already present in an imported
  book. New book use, newly equipped town items, CarryParty and next-mission
  mint retain genuine subsequent learning. Old native gob descriptors lack
  the two new flags and keep their zero/default compatibility behaviour.
- Original city export compares current membership with the source graph
  before accepting an unchanged native baseline. A legacy template-derived
  baseline cannot silently export a different source book. Native progress
  is preserved; it is not repaired by overwriting the party.

## Corpus and city controls

Direct SAV files in the six owner directories ending `08-02`, `08-12`,
`08-14`, `08-15`, `08-24` and `08-27` supplied 28 successful Party parses.
Generated subdirectories were not traversed. This finite check concerns
party parsing, not complete world fidelity.

The existing city source `gameversions/saves/2026-08-15/game0010.sav`, SHA-256
`89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4`,
loads two characters: Danath with no book and Reniesta with four learned
spells. EN and RU `cmd/savecheck -resave load` retain chapter 30, gold 683
and both characters through ordinary SAVE. Both generated city files have
SHA-256 `bbee204a94f45e6463ee107f2eb23aaef7059b835c45d6e5338a8090330f781f`.
Outputs are under `review/story1096/city-{en,ru}`, outside every repository
and preserved install.

Pinned `savdoc -mode unit-audit` with the owner `08-24` and required `08-02`
control directories completed. Its record traversal consumed all 58,324
decoded bytes of the primary source with zero trailing bytes. The first
invocation without the required control directory exited 2; its partial
output is not counted as a passing gate. Neither invocation ran ROM1.

## Gates and boundaries

Focused SAV, mapload and game package tests pass, as does the gated-test
manifest check. The new EN/RU release witness passes on both lawful roots.
`gofmt` and `git diff --check` are clean. The preserved-install manifest
check passes for all 181 files. Final full-suite and paired release gates await
the reconciled candidate and its serialized landing.

Reconciliation includes master `2cdd37688081030a98be8fa48ed0fc2d80800bad`.
Source-backed city descriptors return spell validation errors through the
shared provenance constructor. Native baseline validation retains the exact
version1 template policy and uses version2 for new source-book imports; the
state-store DTO independently remains version2. Both source-derived policies
validate every baseline field before session publication. Current native
progress is never replaced from the source graph.

The root ran the actual pre-1096 city AGS files from both the first1095 writer
and landed2cdd3768 through current LOAD on EN/RU, unchanged and with added skill
XP, Body and learned-spell progress. All eight cases preserve the complete
native party and their version1 baseline. These two real city fixtures happen
to have matching source/template membership: unchanged SAV export remains
allowed, changed Human export refuses. The independent synthetic regression
uses saved ID26 versus template ID1 to prove genuine old-book mismatch refusal,
with native continuation allowed, and rejects a forged version1 Body baseline.
Receipts: `review/story1096/legacy-native-{en,ru}.log` and
`reconcile-legacy-focused-final.log`. The four changed package suites pass.
The encode-side empty-world envelope pin is
`58fd2d08f8980358bc53e862289e0132660549d8ed30831c76b595a3ad3bea7c`;
older decode fixtures and canonical World bytes are unchanged.

Reconciled code `cfd67a0c1d4531027f3294281ace3341b19a642c` passes the complete
`go test -trimpath -count=1 ./...` chain and gofmt. The following asset guard
initially hit Git ownership protection; its exact-path process-local override
then passes without rerunning Go. New spellbook and city-converter production
witnesses pass on EN and RU: 28 learned rows, cast16 and native hash
`e89b608715627a3e`; city AGS13397 bytes and unchanged SAVaf8f9a5e/3215 bytes.
Logs: `reconciled-final-go.log`, `reconciled-final-noassets.log` and
`reconciled-release-{en,ru}.log` under `review/story1096/`.
The sole fresh review returned one transactional defect: a wrong-class actor
reference before a malformed book masked `ErrSpellbook`, allowing a shortened
party to replace the active session. The single correction retains the first
diagnostic and joins the fatal book identity at most once. Permanent controls
cover city and mission sources, one and64 malformed Humans, both import doors,
and unchanged active-session identities and complete snapshots. The original
independent review overlay and the new regression pass together; receipt:
`review/story1096/correction-independent-overlay.log`. No second review is
required. The final merge's paired release remains outstanding.

The headless mission runner reports zero UNSUPPORTED nodes for mission 10
and zero for mission 20, unchanged from `pipeline/milestone-baseline.txt`.
The changed observable result is the restored 28-row usable spellbook,
not the mission-script census. No simulation byte-form or envelope version
changes; two additive native party metadata fields change the gob descriptor.

`DIV-650` keeps the unapplied saved instance parameters explicit.
`DIV-651` keeps sim container-presence and the no-book teaching gate open.
`DIV-652` keeps old native import loss open with fail-closed original export.
No original process, GUI, install mutation, new research claim, new command,
general mission SAV writer or full-original-roundtrip claim belongs to this
story. Source saves and generated evidence remain outside Git.
