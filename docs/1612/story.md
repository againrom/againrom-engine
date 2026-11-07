# Party membership, identity words and joins

## Intent

Use one membership model and identity codec, and one layered join catalog for
both games. Preserve the people who join, their join moments, failure reasons,
mission-entry order and SAV bytes. This is architecture work with no player
change.

## Authority

The owner requires one builder per kind, in the second game's shape with the
first game expressed as profile data. Audit ranks R6 and R7 name the membership
homes, identity word and chapter, talk and departure join paths. The behaviour
record was committed before production changed, against `963d9ea207ec395c600de7393b708e876c364b6e`.
The existing talk-end behaviour and EXP-2080 remain. This change makes no new
claim about either original runtime.

## As-built behaviour

`mapload.Membership` carries kind, identity, Humans or Units arm and temporary
continuity. `PartyMember.Membership` reads the existing wire fields. `Is`,
`Matches`, `Hired`, `UnitMember`, `CampaignHireID`, `TavernType`, `CompanionID`
and `ValidMembership` are the sole predicate and validation set. `Membership.ID`
builds companion, campaign hire and tavern squad identities. Wire field names,
JSON layout and the SAV producer remain unchanged.

`mapload.PartyIdentity` and `DecodePartyIdentity` encode the identity word and
admit a saved companion through the same `CompanionIdentity` catalog lookup.
The decoder preserves the actor-arm, hero-flag and matching-Humans-row checks. The
hero-band binder keeps its role and identity sources; it calls this codec.

`base.PartyProfile` supplies the profile rows, tavern types, Units types and
composed companion name identities. The first game's tavern types are 1..15,
Units types 1 and 2, and composed name identities 21..24. The second game's rows
are hero 21 with failure reason 2; companion 22, Humans key 10220, at the end of
NPC 22 topic 30, with reason 3; companion 23, Humans key 10230, through departure
bank slot 773, with reason 4. The slot's current value still selects the member.

`townInstall.joins` layers campaign AddHero rows, in sorted chapter order and
original grant order, over those profile rows. Mod rows replace their chapter
grant and preserve mod order. `joinMember` is the one append path for chapter,
mod talk-open, profile talk-end and leave triggers. Pending grants and departure
bank values remain current-state inputs. Guest reuse, worn-spell application,
city group settlement and duplicate suppression retain their existing order.
The `endInnTalk` call site is unchanged.

The hire producer, transaction and offer list remain `hireMembers`,
`hireDeal.run` and `innOffers`. They read the membership predicates and profile
types. Character construction still uses `CampaignActor` and `CharacterMember`.
Speaker rows, the census party and failure reasons project the same catalog.

## Proof

`TestPartyJoinRecord` compares the base record after chapter, talk, mod join and
leave triggers, including repeated and unrelated triggers. It records mission
entry members and order, identity words, failure reasons and SAV lengths and
SHA-256 digests. The golden is unchanged. A planted first-game grant 22 to 23
fails the SAV digest comparison.

Focused mapload and game membership, companion, identity and hire tests pass.
`TestCheckMembershipNamesEachShape` plants field comparisons, aliases, switch
and loop conditions, codec copies and word masks, join reader copies, raw
catalog reads and a direct leave-slot read. `TestPartyMembershipHasOneBuilder`
checks production sources. The import DAG and existing hero-band, hire and loss
reason guards pass. No new package dependency requires a DAG allowance.

The hero guard recognizes the membership form and names planted copies in
both production trees. The membership guard rejects direct tavern type bounds
in hire, tavern and mercenary contexts. `Edition.TavernType` and `TavernUnit`
read the sole profile data for hire arms, talk resolution and saved-type reads.
The `siegeHireName` guard entry retains type 2 as the Ballista name selector;
it does not decide membership. Fixed SAV array widths remain format limits.
The town catalog's `mercenaries.stock-types` entry remains the stock actor
count: replacing it with the membership list would discard authored catalog
counts and change runtime ID allocation for those catalogs.

The correction runs the focused mapload and game selections, the whole
`internal/archtest` package and the fast gate. No release test runs in the
correction. Receipts retain commands, commit, outcomes and literal searches.

## Open debt

The seat rerun passed the four editor tests previously denied by the sandbox;
the correction's sandbox run denies the same four writes:
`TestEditorPartyRowsPreserveTemplateAndEmptyOnSaveAndReopen`,
`TestEditorPackageAdmissionRefusesAnUnresolvedPartyBeforeWritingOrStarting`,
`TestEditorImportedPartySurvivesWorkspaceProjection` and
`TestTestPlayOfAPartyPackageKeepsTheDeclaredParty`. The broader correction
selection also fails `TestPersonWindowNPCListCarriesTheSpeakerListsFaces`
on NPC 8. An overlay of all pre-correction production files reproduces that
failure. These failures are outside the two returning findings. Two optional
tavern release witnesses require output outside the lane's write boundary and
remain skipped. Full release acceptance belongs to the seat.

No behaviour divergence was added. DIV-4625 and DIV-4626 are unused.
