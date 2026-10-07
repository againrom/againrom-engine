# First ROM2 campaign route

## Contract

Ordinary New Game opens type-2 location ID 1. The first inn TALK reads the installed `main/text/town.txt` section `npc517talk10` and unlocks type-1 location ID 10 immediately. Dialogue acknowledgement pages text without changing availability. Leaving the initial location exposes only the available mission record. Selection, cancellation and ENTER are distinct actions.

Mission 10 retains its installed dialogue and victory trigger. Victory acknowledgement runs the bounded first Leave transition and exposes type-1 ID 20. It does not enter a numerically inferred successor. The player selects the actual available record and enters its map through the ordinary mission opener. Failed entry retains the campaign and party. Failure exits to menu/load without Leave, completion or a successor grant. A later New Game drops the prior campaign.

## Authority

Pinned public knowledge: `32416bb8a6cde07eef03fd51296fe4ee8c56b5db` (k180), read individually with `knowledge/tools/claim`.

- `R2-ENGINE-073`: separate initial type/ID, inn TALK unlock, record-selected mission address (Medium; complete native schedule Unknown).
- `R2-ENGINE-074`: first victory acknowledgement versus failure choices (Medium).
- `R2-SESSION-023`: cleared 1024-DWORD NewGame bank with768=10, first Leave bank/list algebra (High; arbitrary bank writers and party carryover Unknown).
- `R2-ENGINE-048`: mission entry clears752..767; selected ID10 Leave adds ID20.
- `R2-ENGINE-049`, `R2-ENGINE-050`, `R2-ENGINE-052`, `R2-ENGINE-060`: installed text sections, portable text conversion boundary and actual result producers. Prior dialogue/collection debt remains.
- `R2-ENGINE-075`, `R2-ENGINE-077`: native conditional movie selection and companion-key consumers. No media decoding or input rule is inferred.
- `R2-ENGINE-076`: additional-unit filter/template paths do not establish a complete party.

## State and boundaries

`Town.second` owns the ROM2 campaign location list/current record and the detached bank. It introduces no FrontEnd field and is replaced by the existing session clear. Historical sessions have nil ROM2 campaign state. ROM1 campaign records and native/binary save defaults retain their prior route.

A successful mission entry copies the campaign bank, clears752..767 and commits the current type-1 record only when the complete mission driver activates. A failed decode or entry commits none of those writes. Existing ROM2 binary script state stores the bank deterministically; ROM2 SAVE remains unavailable, and no new save format or restore claim is added.

First Leave takes the live bank. It sets773=0, clears512..531, normalizes retained532..551 to1 or2 according to the incoming working flag, sets906=1, advances the measured incoming stage and installs ID20 availability. Nonzero775 refuses the unclaimed restoration branch. Later completions are refused before closing their victory panel or consuming delayed Victory. Existing Continue/Escape remains usable without completing the campaign. No universal succession table is added.

Current hero identity, equipment, inventory and current character state use the existing mission-return/entry path. That portable party is not the native ROM2 main-hero/additional-unit constructor. The existing mercenary/effect normalization remains an explicit ROM2 boundary. Initial/default hero construction and later template variants need their own promoted producer evidence.

## Presentation and remaining work

The campaign controller uses a portable row screen and existing dialogue compositor. Native town, inn/world-map geometry, portraits, all inn topics and exact report/movie presentation are outside this slice. The selected teleport output is not decoded or played. The bounded slice discloses continuation after mission20 as unavailable, while leaving mission20 playable and its victory panel/menu usable.

Debt: DIV-2390 (town/destination presentation), DIV-2391 (party constructor/return), DIV-2392 (later/restored availability), DIV-2393 (report/movie presentation). DIV-2394 and DIV-2395 remain unused. Earlier ROM2 script/text/collection debts remain open. A private research candidate supplies no authority.

## Evidence

External evidence index: `review/story1354-rom2-first-campaign/PROOF.md`.

`TestSecondCampaignInputUnlockCancelAndFailedEntry` drives App menu input, TALK/page acknowledgement, cancellation, failed ENTER/retry and unavailable destination refusal over independently built archives. `TestSecondCampaignFailedNewGameRetainsSession` refuses a missing initial source without replacing a prior session. The bank algebra test covers zero/nonzero signed cells and unclaimed branches.

`TestReleaseSecondCampaignFirstRoute` drives the ordinary App route on each ROM2 root. It uses a bounded setup placement near the installed final NPC, then a real App selection/move and normal ticks execute the installed final dialogue/win trigger. The witness is not a full native or unassisted mission playthrough. Immediate Victory and Continue followed by End Quest Victory expose20; duplicate acknowledgement, cancelled selection, completed10 refusal, party identity/equipment/inventory, bank carry/reset and post-entry20 ticks are checked. `TestReleaseSecondCampaignFailureDoesNotLeave` uses ordinary controlled-hero death and the visible failure menu choice.

Focused installed ROM2 EN/RU, ROM1 shared-route controls, touched ordinary packages, full ordinary Go, guards and no-assets validation are candidate checks. Sole adversarial review and promotion belong to the seat.

The initial-route RED fails at the initial location on the base. With only the transition source restored to the base, the installed mission10 victory RED reaches the picker without a successor. Candidate EN/RU runs pass both boundaries.

Auxiliary documentation limitation: the global claim-citation scan fails on a pre-existing unresolved world-map reference. PROOF.md records its exact diagnostic and base comparison. All new campaign authority citations resolve at k180. Historical prose is preserved; this slice does not claim the global scan passed.
