# Settled return campaign state in city SAV

An ordinary mission20 return in a fresh campaign can save SAV before accepting
the next town mission. A fresh SAV LOAD keeps that mission unannounced. Accepting
the NPC conversation consumes its candidate and announcement survives a second
SAVE through the source-bound writer. Full imported mission return retains its
separate current-actor refusal; it is not claimed complete here.

## Authority and behavior

The public knowledge pin supplies active SAV-CAMPAIGN-076/077/079/080/081/085.
Selection, announced records, remaining building candidates and documents have
different meanings. Both writers already have the typed campaign setter.

`FrontEnd.Offered` is this engine's old map-list advisory. Its non-test consumers
are immediate completion messages, Snapshot/Restore, city guards and savecheck
diagnostics. Town availability, building acceptance and world-map action do not
read it. Shared `citySaveAdvisory` permits omission only for zero or the current
open-town chapter. SAV LOAD continues to default it to zero. This policy is
DIV-1161, not an invented original field. AGS retains the advisory unchanged;
other values and pending travel still refuse SAV.

The generated main record now writes `Town.available[chapter]`, as child records
already do. It no longer publishes a premature main announcement. Both writers
keep existing roster, Human, spellbook, marker and provenance checks. No World,
simulation version, renderer or original binary changes are made.

## Proof

`TestReleaseTownReturnCurrentCampaign1168` runs two bounded paths on each install:

| Path | Before accepting an NPC | After accepting and cold second SAVE/LOAD |
|---|---|---|
| Actual App mission20 opener, then controlled `LiveCompleteCampaign` -> `FinishMissionWithRoster` with its World/party/IDs/roster | Main/selected30; child31; announced empty; inn `[30]`/NPC `[22]`; shop `[31]`; documents empty; money600 = opener purse100 + transition500 | Announced `[30]`; inn/NPC empty; child31, shop31, documents and money600 unchanged |
| Imported game0010 **campaign-only endpoint**, `Town.Won(30)` plus advisory40; source actors unchanged | Main/selected40; child41; announced empty; inn `[40,41]`/NPC `[22,90]`; school/shop empty; documents `(1,1),(2,1),(3,1)`; money1683 =683+1000 | Announced `[40]`; inn `[41]`/NPC `[90]`; child41, documents and money1683 unchanged |

Completion is controlled at the production boundary; neither path claims a
mission played to its victory trigger. The fixed imported city is the preserved
`game0010.sav` with SHA256
`89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4`.

The oracle reads raw campaign framing and literal arrays independently of both
production projectors and `File.Campaign`. Each path keeps a serialized AGS
advisory baseline, compares zero/current-advisory SAV bytes, checks no source/live
mutation, performs ordinary SAVE, deletes the private imported source, and uses
fresh FrontEnds. An unchanged SAVE and a fully read but declined NPC conversation
leave money and offers alone. Actual town row dispatch accepts the NPC; the
second writer preserves the actor body and state store. Reload cannot pay the
previous main again. Negative, side/future and custom advisory values keep AGS;
pending-trip export still refuses.

`TestReleaseTownReturnImportedActorDebt1168` opens the actual imported party in
mission30 and calls the same completion boundary. It names two members losing
`OriginalHuman` and `Saved` in `CarryRoster`. The unchanged Human guard refuses
SAV. Source-free AGS LOAD keeps campaign40, money1683 and actor values; the legacy
upgrade adds explicitly retired, inactive Human-basis markers. This is the
DIV-1162 follow-up, not a successful imported-return SAV witness.

Focused gates, standalone converter evidence and loss-control results are
recorded in the seat's ignored `review/story1168/` receipts. The seat owns the
single review, final merge gates and `builds/current` promotion.

## Remaining scope

Full imported actor return, generated actor defaults, consumed companion grants,
terminal campaign150, travel progress, generic roster/item mutations, marker
selection history and world SAV authoring remain separate obligations. This
story does not upgrade their authority or claim original-runtime acceptance.
