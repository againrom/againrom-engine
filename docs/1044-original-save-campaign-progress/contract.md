# Contract draft — story 1044 original-save campaign progress

**Seat-finalized dispatch source.** Open `wt-story-1044` from exact implementation
`d5b76f4c6c519098f28c924b1291dce318c181a4`. Advance its research pin to exact
`23daf74f6ee83e5fee5474981faf7c276680cc9f` in the canonical contract commit before production
code. Copy this file byte for byte to the story folder. Story `1040` and `EXP-0232` have landed. Do
not fold this work into a world-half session overlay story. Do not re-run campaign-state research.
Use `DIV-417` through `DIV-424`; stop and ask if the range is spent. Do not rebase or use stash. If
implementation master advances, make a coherent commit and merge the exact new master normally.

## Outcome

A ROM1 save resumes the campaign record it actually came from. Main progress, selected and retained
child missions, announcement and town-offer state, mercenary state, documents and markers replace
fresh defaults before any campaign consumer runs. The world map and town expose the lawful next
choices, and lower main missions cannot reopen.

The owner discriminator is the save labelled `we have brian !!`: it restores Brian in mission 41,
while its campaign record is main 50, selected 41, retained children 41 and 51, and
`InnMission=[0,50,51]`. It contains no mission 30 or 40 state. After mission 41 completes, this build
currently offers mission 30, then mission 40, and can add a second Brian. The story passes only when
completion returns to main 50 without that replay. Brian remains unique because lower progress is
not reopened; ROM1 has no Brian-specific identity guard in the mission-40 handover.

## Dependencies and domains

- Story `1040` has landed so party and item identity have their final form-61 representation.
- `EXP-0232` publishes the exact serialized fields, load order and consumers in
  `SAV-CAMPAIGN-076` through `SAV-CAMPAIGN-088`.
- The independently active token-load, cell-rebind and dead-actor-projection SAV wave owns those
  three concerns. This story neither consumes nor cites that work. It uses only the pinned
  campaign-state publication and does not allocate a new research id or SAV claim id.
- Touched domains: **Assets & Formats**, **Campaign & Scripts**, **Persistence** and the party
  identity boundary. World-half script state, actors and cells are not touched.

## Behaviour

1. `pkg/formats/sav` exposes an immutable typed projection only for the decoded campaign fields:
   main and child records, each record's mission, MapObject, Payment, shop bounds, announcement
   latch, `AddHero[]`, `EnableMercenary[]` and child age; paired mercenary-count arrays; hire flags;
   the six candidate/unlock arrays; documents; selected mission; `AutoGetMission`; `LastMission`;
   the first-MapPoint flag; mission time; and marker cache. The unnamed scalars at `+0x114` and
   `+0x128` are rejected or ignored explicitly, never guessed into canonical state.
2. Original-save loading applies that projection after a fresh campaign object exists and before
   town, world-map or mission-list consumers run. Validation is complete and atomic; malformed sets
   do not partially mutate the campaign.
3. The projection replaces fresh-campaign defaults. It is not unioned with chapter 30 and is not
   inferred from the currently loaded mission alone.
4. Finishing the loaded mission advances from the restored state exactly once. Main progress stays
   monotone, a completed side mission is removed from the retained-child array, and acceptance
   consumes the matching persisted town candidate before setting the record's announcement latch.
5. No new companion-deduplication rule is introduced. The `game0020.sav` Brian remains one member
   with one stable identity, equipment set and carried state because mission 40 cannot reopen below
   restored main progress 50. A forced noncanonical replay remains disclosed rather than silently
   changing ROM1's handover semantics.
6. Againrom's own save/load and hash or envelope state preserve the restored campaign projection.
   No unnamed raw SAV bytes are copied into canonical state.
7. Simulation form 61 remains current unless the implementation must change its canonical binary
   representation. If that change is necessary, this story owns form 62 and must provide every
   upgrade, hash and malformed-input witness. A game-layer envelope change that does not alter the
   simulation form does not consume form 62.

## Witnesses

- A field-complete table test changes each decoded campaign axis independently and proves exact
  validation, atomic rejection and canonical persistence. It also proves that the two unnamed
  scalars cannot enter canonical state.
- The fixed owner EN `game0020.sav` witness confirms main 50, selected 41, children 41/51,
  `InnMission=[0,50,51]`, one Brian and no mission 30/40 state; completing 41 returns to main 50 and
  exposes neither mission 30 nor mission 40.
- Both lawful install roots drive an unrelated original-save control through the same importer and
  prove that each root's authored campaign fields, rather than EN constants, supply the projection.
- A forced attempt to reopen mission 40 is refused by restored monotone progress. It does not add a
  new roster-dedup mechanism or change Brian's identity and full story-1040 item instances.
- Fresh-campaign, between-mission againrom save and an unrelated original-save control retain their
  existing progression.
- Mutations that ignore the overlay, union defaults, reorder application, reoffer a consumed row or
  deduplicate only by display name make the production witness red.
- All witnesses are static or headless. Do not launch or control the game GUI. Any remaining
  player-only discriminator is returned to the owner as a question.

## Bounds

Do not absorb world-half registers/latches/diplomacy, live actors, sacks, structures, terrain,
area effects, orders or casts. Do not redesign the campaign graph. Any field not established by the
experiment remains out of canonical state and is disclosed.

## Review ceiling and stopping condition

At most three fresh adversarial passes. Only P returns the story. Stop at the first no-P pass with
an empty remaining-surface list; apply W and D without another pass. The first brief names the whole
campaign-field producer/consumer population, every fresh-default writer it must displace, and the
Brian discriminator.
