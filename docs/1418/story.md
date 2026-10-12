# Story 1418: hiring in the second game's inns

## Intent

In the second game's inns the player hires and dismisses the companions the campaign has made
available. The inn page seats them, Hire and Fire act on them, and the selection is campaign state
that SAVE and LOAD keep. The NPC 5 talk arm stores what the original stores. The mission entry of a
selected hire is not built here (DIV-3111).

## Authority

Second-game claims at pin k275:

- `R2-ENGINE-493` (High): TalkTo's two kind 0 arms. NPC 5 at any topic stores 536=1, 556=3, 774=2;
  NPC 675 topic 78 stores 770=2 and adds the class-4 mask. Kind 1 stores `bank[511+ID]==0` into
  511+ID. Kinds 2 and 4..7 store nothing.
- `R2-ENGINE-495` (High/Medium): Hire on a kind 1 word shows `npc<ID>accept<stage>`, sets bit 31 and
  calls TalkTo; on any other word only `npc<ID>reject<stage>`. Fire clears bit 31 and calls TalkTo.
- `R2-ENGINE-496` (High/Medium/Unknown): on the generic inn page the center click and button 0 run
  Fire when the copied word has bit 31, else Hire, on a hire seat; the center click runs TALK on a
  talk seat; button 1 formats `npc<key>about` on a hire seat.
- `R2-ENGINE-497` (High/Unknown): hire seats before talk seats; none while the selection is -1; key
  257 passed with the seat rectangle on a hired seat.
- `R2-ENGINE-498` (High): the first copied option with the seat's low ID is the word acted on.
- `R2-SESSION-191` (High/Medium): Hire and Fire toggle bank 511+ID; kind 2 is availability 2.
- `R2-ENGINE-221` (High): the dynamic tail. `R2-ENGINE-283` (High): the generator commit writes
  slots 776 and 781 from the class and the sex.
- `R2-ENGINE-380` and `R2-ENGINE-381` (Medium): the hire bank and the mission-entry loop, cited by
  DIV-3111 only.

## As built

- `pkg/game/secondinn.go`: `talkTo` adds the NPC 5 arm and the kind 1 toggle of 511+ID for IDs 1..20.
  `hires` lists the kind 1 and 2 options; `innSeats` is hires then speakers. `innWord` takes the
  first option with the low ID and the page's per-visit copy of bit 31; `hireOrFire` runs Fire on a
  set bit and otherwise Hire, returning the accept or reject section key from bank 768. `innRows`
  reads HIRE n, FIRE n or TALK n.
- `pkg/game/secondtown.go` and `secondcampaign.go`: every seat click selects the seat and runs its
  action. The hire button acts on a selected hire seat only, the talk button on a selected talk seat
  only. Hire seats draw `hire-back` (ManBack) and the actor; a hired seat draws main-text 257
  centred on its rectangle; while no seat is selected no hire seat is drawn. The hire button's
  caption is 259 or 258 on a selected hire seat and none otherwise. The page's bit copy resets when
  the page opens.
- `pkg/town/seats.go` and the three `rom2-town*.json` descriptions: the `hire-back` seat art.
- State: the selection is bank slots 512..531, already written and read with the campaign bank; no
  save field was added.
- Census: `secondEngineReach` unions one walk per character choice (slots 776 and 781 from the
  generator description). The inn offers 75, 76, 77, 81, 85, 86, 87, 94, 95 and 96 needed a class or
  sex choice and are now reached; 83 is reached through the NPC 5 arm. Mission 52 still has no
  producer.

Rows: DIV-3109 (hire seat look), DIV-3110 (routing), DIV-3111 (mission entry, `rom2.md`), DIV-3112
(talk button on a hire seat). DIV-2636 and DIV-2639 closed. DIV-2964 updated.

## Proof

- Focused: `TestSecondInnHire*` in `pkg/game/secondinnhire_test.go` (toggle, Hire and Fire with kind
  2, the first-word reject at stage 50 with its loss control, routing and captions);
  `TestSecondInn*` talk effects; `TestSecondEngineReachTakesEachCharacterChoice` with the fighter male
  walk as loss control.
- Release on `rom2-en` and `rom2-ru`: `TestReleaseSecondTownInnHireSaveAndFire` (Kaarg inn after
  mission 20's departure stores: HIRE 1 first, Hire shows npc1accept30 and sets slot 512, the hired
  mark changes the seat, SAVE and cold LOAD keep slot 512 and FIRE 1, Fire clears it with no
  dialogue, the fired seat paints as the unhired one) and `TestReleaseSecondTownInnHireRejectsKindTwo`
  (npc1reject30, bank unchanged). The campaign position is constructed as `secondTownTwo` builds it.
- Ordinary route: `TestReleaseSecondStageFortyInnOffersItsMissions` plays missions 10 and 20 and
  finds the stage 40 inn rows HIRE 1, HIRE 2, then the five speakers, before and after SAVE and cold
  LOAD.
- Census on both ROM2 roots (`TestReleaseSecondGameSupportCensus`): entry maps 34 to 45, ready maps
  9 to 14; only mission 52 is blocked on entry.

## Open debt

- DIV-3111: a selected hire does not join the party in a mission.
- DIV-3112: the talk button on a hire seat does not show `npc<ID>about`.
- DIV-3109 and DIV-3110: the hired mark's look and the Kaarg and druid pages' own routines are
  Unknown; the generic page's rule serves all three inns.
