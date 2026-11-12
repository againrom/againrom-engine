# Mission flow and the party of a mission

## Intent

A mod sets campaign links and the party of each mission. The main party can
play an added mission between installed missions. A guest can play an
installed or replaced mission while the main party waits.

## Authority

Owner direction in the brief authorizes automatic links through added side
missions and guest parties in campaign missions. REG-SCN-063 and SAV-1329
establish the routing key and the requirement that a request exceed the
current main mission. REG-SCN-059 describes the scenario sections;
REG-SCN-062 describes retained side records. Knowledge pin: k456.
SAV-1355 and SAV-1356 establish the carried regeneration stamp and Mover.

## As-built behaviour

`kind = "campaign"` in the existing missions file selects an installed
mission, replaces its map, or adds a side-numbered mission with a map.
`AutoGetMission` overrides the campaign value before a session starts.
`Payment` is an optional row of the existing completion. The default party
is the main party; `party = "main"`, `guest` with `chapter`, and a party
array state the alternatives. MODS.md owns the format and worked example.

Campaign is still the one decoded scenario value. Mission declarations use
ParseMissions, MergeMissions and WriteMissions; dataKinds still has one
missions row. New Game, numbered entry and LOAD share missionOpenerMode,
enterMission, startMission and mapload's party construction. Guest selection
extends startGuestMission and guestParty. The winRowSets guest row restores
the waiting SAV before chapterPlace applies the campaign completion. The
main party in an added mission uses the ordinary carry. Auto successors
select child records without advancing the main record until a main target
is reached. SAV uses ExportCurrentSave and campaignRecord; its routing word
comes from autoGetMissionValue. The check uses MissionData.Replacements.

A removed automatic link, or the end of an added mission with no link,
returns to the town with the next main mission announced. Campaign carries
that completion data and its town boundary. No reader of AutoAdvance checks
a mod or game identity.

Only a joining guest's state returns. Other guest members, purse, loot,
experience policy and knowledge remain with the guest session. The campaign
payment, documents, mercenary unlocks and town are completed for the waiting
party. Main-party loot, experience and inventory persist across added
missions. Waiting joiners remain available for their town grants.

The existing guest record gains an omitted-when-empty Previous field for
pending joiners. The session gains omitted-when-false GuestWaiting for the
internal SAV before a town exists. These fit SAV's existing continuation
record; no SAVX field or second producer is introduced. Legacy guest-chain
bytes are held by the base win-route witness.
The restored waiting party bypasses a second mission carry. Its kept actor
state survives unchanged, and completion records the guest-played map.
The campaign builder derives the town reached by every automatic path.
Guest and campaign rows share the check of that town and its companion grant.

Differences: DIV-4854 records side selection and completion in an automatic
chain; DIV-4855 records guest play of campaign missions; DIV-4856 records the
selected side successor and the waiting SAV continuation; DIV-4857 records
the early town and its announced next main mission.

## Proof

Synthetic parser tests cover parsing, writing, check output, precedence,
unknown missions, cycles and invalid targets, with file and line refusals.
Synthetic front ends exercise 10 -> 11 -> 20, the campaign record in mission
SAVs, LOAD inside 11, main-party carry, guest play and LOAD in mission 20,
and guest play before the first town. TestWinRouteRecordIsTheBaseRecord
compares legacy routes and SAV digests to main's committed witness.
An overlay of the base production files and the same raw-SAV instrument
also compares six full SHA-256 digests with the candidate, with no label
normalization. All six match, including the chain after wins 20, 25 and 27.
`TestFixtureModsGiveTheRecordedMergedData` keeps the recorded mission-table
hashes unchanged for both fixture languages. Empty new campaign fields are
omitted from JSON; all ten canonical mission forms equal the base bytes.

Synthetic correction tests refuse installed side targets and past joining
towns with file and line. Later main targets, added missions, added chains
and -1 preserve the live town's offer, hand-outs, party digest and SAV after
LOAD. The waiting party keeps every DTO field, including a nonzero kept
regeneration stamp; the next mission reads that stamp. Whole SAV controls
compare guest and main-party wins before and after a town, also with a LOAD
inside the guest mission. The before-town control starts from the same
waiting SAV; only the joining guest's member is aligned for comparison.
The documentation example loads through modrt.Load from a synthetic mod
folder with both languages' texts. Added mission 11 has loss, abandon and
restart controls. The check reports payment-only and party-only rows.

The notes RESULT.md holds the final gate receipts. The lane runs no fast
gate, installed witness, release test or push, as the brief requires.
The comment-byte baseline rises by 102 bytes for the new campaign-field and
campaign-validation documentation. No other storyguard count rises.

## Open debt

Automatic targets that are installed side missions are refused with their
source file and line, even when a mod supplies a row for the target.
Supporting those targets remains open debt. Allowed targets are later main
missions and missions the mod adds; -1 ends the automatic chain.

The editor's mission form needs the new campaign kind, links, payment and
main/guest party choices, with preservation on editing and test play.
The current guest decoder and entry bound pending joiners at 64. A larger
waiting-party representation remains outside this slice.
The shared scenario reader carries the overrides for either game, but the
second game's scenario bank owns its routing; ROM2 application is debt.

Original behaviour for an added side mission inside the automatic chain,
its completion, and the original's next action after loading its SAV are
Unknown. No original runtime was driven and no research question was sent.
EN and RU release witnesses remain owed: the campaign walk, guest mission
and chain, campaign replacement, and automatic-route SAVE/LOAD tests named
in RESULT.md, plus an installed 10 -> 11 -> 20 and guest campaign witness.
