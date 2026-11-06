# One hire producer, one hire transaction and one offer list

## Intent

Make the hire of both games one builder per kind: one hire producer, one
hire and dismissal transaction and one offer list, in the second game's
shape, with the first game as profile data. No player change: the same
persons are offered, a hire costs the same gold or bank words, the party
enters the mission with the same members in the same order, a dismissal
returns the same members, and every SAV keeps its bytes. Architecture work
counts zero toward a release.

## Authority

Owner rulings: one builder per kind; the first game is a reduced second game,
one builder in the second game's shape with the first game as profile data;
architecture work counts zero toward the release. The audit plan's section
"R5 Inn roster, hire and talk offers and R41 Companion hire/dismissal" names
the instances and the census. The behaviour kept is the base's behaviour at
`e8dccbd68` (game 0.148.0), which the hire record pins; the second game's
inn hire follows R2-ENGINE-493..498 and R2-SESSION-191 as the base cites
them. No claim, DIV row or behaviour changes.

## As-built behaviour

`pkg/game/hire.go` holds the hire rows of both games.

- A hire row is `{identity, tavern, arm, selection, availability, town}`:
  the identity, the actor arm `mapload.CampaignActorProfile` (row name or
  keyed row, stage key), and the bank slots the second game reads.
- A hire profile is the rows plus the moment (at the offer or at mission
  entry), the cost source (gold or the bank bit), whether worn spells are
  joined, the offer walk, the stage, the offer kind, the selection, the
  marching test, the squad size and the price.
- `tavernHires` is the first game's profile: fifteen tavern types, types 1
  and 2 the Catapult and Ballista Units rows, 3..15 the Humans rows; hired
  at the offer for gold; walk `tavernStockWalk`; stage
  `mercenaryLevel(chapter)`; offered when the chapter lists the type, it is
  enabled, not hired in another town and the pool is not empty; squad size
  the pool; price `mercenarySquadPrice`.
- `bankHires` is the second game's profile: twenty hires, the keyed arm
  (server base 10000, stride 10, stage divisor 10, bias -2), selection
  `511+id`, availability `531+id`, town `551+id`, stage `bank[768]`; read at
  mission entry; cost the bank bit; offered when the town slot is the
  current town and the availability is 1 or 2.
- `gameFiles.hire` names each game's profile.

`hireMembers` is the one producer. Its callers are the tavern
(`townScreen.tavernSquad`), the native city restore, the inn subject's
second route in `campaignActor`, the siege member of an original save
(`siegeHireMember`) and mission entry (`enterHires`). `enterHires` replaces
both games' `missionParty`. It takes the campaign service, whose `files()`
names the profile and whose `hires` reads the town's hire state: the
second game's service picks the campaign's bank, the first game's has
none. With the first game's profile it returns the
party unchanged; with the second game's it keeps the party's members,
drops a hire that no longer marches and appends each marching hire not
present, in row order. `departHires` is the completed-bank loop.

`hireDeal.run` is the one hire and dismissal transaction. A deal states the
cost source (a gold price from a purse, or the word's TalkTo), the pool's
hired flag, the party list and the members, the seat copy of the word's
high bit, and the caption or refusal. The tavern builds its deal in
`Town.squadDeal`; the inn seat in `secondCampaign.seatHire`.

`innOffers` is the one offer list. With a profile in `innState.hire` it
appends `hireOffers`: the inn's hire options for the second game and the
tavern's squads for the first, which `tavernMercenaries` projects to
`TavernOffer`. The first game's talk offers come from the catalog's rows
through `innOffers` (`Campaign.chapter`), as at the base.

### Kinds and builders

| kind | builder |
| --- | --- |
| a hired party member, both games | `hireMembers` over the hire rows |
| a hire or dismissal, both games | `hireDeal.run` |
| an offer list (tavern, inn, talk) | `innOffers`, with `hireOffers` for the hires |
| mission entry with hires | `enterHires` through the campaign service (`files().hire`, `hires`) |

## Proof

Fixture-only tests read no install; the release runs read the EN and RU
roots of both games.

- `pkg/game/hirerecord_test.go`, `TestHireRecord`, committed alone before
  the change, writes `pkg/game/testdata/hire_record.txt` (290 lines): for
  both games the offer list, the party after hire, dismissal and mission
  entry (members, order, types, levels), the purse or bank words, and the
  SAV bytes' SHA-256. The file is byte-equal after the change. A first-game
  squad price moved by one fails it on line 1.
- `internal/archtest/hire.go`, `CheckHire`, with
  `TestCheckHireNamesEveryShape` (one planted source per shape and its
  admitted twin) and `TestHireHasOneProducerAndOneTransaction` on the live
  tree. Outside `pkg/game/hire.go` no `pkg/game` production source indexes
  or sets a bank slot 511..571, names a hire slot constant, sets `Hire` or
  `MercenaryType` of a campaign actor spec, writes or takes the address of
  a first-game hire flag, or declares a `hireDeal` method; no source
  declares a retired producer. Named entries: `towntravel.go:restoreTowns`
  (copies a validated travel record) and `tavern.go:squadDeal` (hands the
  flag to the transaction).
- The inn offers scan names `hire.go:hireOffers` and
  `tavern.go:tavernMercenaries` as builders and `TavernOffer` as an offer
  type. The campaign actor scan names `hireMembers`.

## Open debt

- `Town.chapterOffers` turns the chapter's live talk arrays into
  `TownOffer` values. Those arrays are SAV state (SAV-CAMPAIGN-081), so the
  projection stays; it builds no list from rows.
- `townScreen.mercenaryPartyCount` has no production caller left; tests
  still read it.
