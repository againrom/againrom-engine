# Hurt voices follow k147

## Intent and authority

A person's hurt voice follows his sex and class, never his weapon, and the
events that produce a hurt voice, the device each plays on and the gate each
takes follow the original where the claims are High. Owner report: Danas with a
bow and the hired healer sounded like Naira. Authority: `ANIM-094`,
`ANIM-095`, `ANIM-096`, `ANIM-125` (High, one Medium), `ANIM-126` (High for
the death arm, the bleed and the entry projection; Medium for the sender
list), `ANIM-127`, `ANIM-128`, `ANIM-SND-022`, `HERO-APPEAR-055`,
`HERO-FIGURE-061`. Owner direction: change behaviour only where a claim is
High and the difference is audible; state the rest.

## As built

- The bank of a person is one selector for wounds, bleed, fall and replies
  (`voiceBank`): the mage pair for a mage bit, the hero pair for a hero-shaped
  type, the weapon pair or the peasant pair otherwise, each by sex. Equipment
  and the drawn class change none of it (landed earlier; `ANIM-096`, High). The
  bank sex of a person below type 0x1a is the drawn figure's, which already
  follows the setter's parameter bit 0x80 through the spawner bytes and the
  Humans row (`ANIM-127`, High). The one departure is `DIV-1574`.
- Event 0 (no-damage cue). The simulation now reports a unit strike that took
  no health, with the victim's level twice in `DamageEvent`, when the victim's
  health stays above -10. A structure strike reports nothing. The viewer plays
  index 1 of the drawn class on the effects device, with no gate and no
  timestamp, for a bank-voiced and a class-voiced entity alike. A miss and an
  absorbed blow are such strikes. High for the sites and gates; Medium that
  shipped blows reach zero damage in play (`ANIM-125`).
- Events 1 to 3 play on the speech device, not the effects device, for a
  bank-voiced entity (named leaf) and a class-voiced entity (`Sound[k+1]`).
  The speech device is the one the command replies use. Without it these
  voices are silent; the strike cue still plays (`ANIM-128`, High).
- Bleed (event 2). A body of the local player's whose health is -1 to -9, or reaches -10,
  falls by a point with no blow plays `hard` through the 1500 ms voice gate
  (`ANIM-126`, High for the bleed). The original sends it to the owner alone.
  Bodies of other players are silent.
- Unchanged: wounds band by the health held before the blow, silent at -10 or
  below and within 1500 ms of the one voice timestamp; the fall plays `die` or
  `Sound[4]` with no gate and no timestamp.

## Proof

- `TestAStrikeThatTookNoHealthPlaysSoundOneOnTheEffectsDevice`,
  `TestWoundsAndFallsPlayOnTheSpeechDeviceAlone`,
  `TestAFallenBodyBleedsHardToItsOwnerOnly`,
  `TestAMissingSpeechDeviceSilencesWoundsOnly` (pkg/ui).
- `TestAnAbsorbedStrikeIsReportedWithHealthUnchanged`,
  `TestAZeroStrikeOnAFallenBodyIsReportedAboveTheFloor`,
  `TestAStructureStrikeReportsNoDamageEvent`,
  `TestReportingAZeroStrikeChangesNoWorldState` (pkg/sim).
- `TestAHerosWoundsPlayFromHisBankOnTheSpeechDeviceWhateverHeWields` (the same
  fight as archer and as fighter class; fails if the bank follows the class),
  `TestAStrikeThatTakesNoHealthPlaysTheDrawnClassCue`, and the existing
  `TestAFightVoicesEveryBlowAndTheFallFromTheOwnBank` (pkg/game).
- Release, EN and RU: `TestReleaseHurtVoiceFollowsSexAndClass` and
  `TestReleaseHurtVoiceOfAHiredHealer`, with the speech device wired.
- No World hash, save byte or pin changes beyond the knowledge pin to k147.

## Open debt

`DIV-1489` is narrowed to the entry-projection die cue and the other state-sync
senders; the viewer holds no client corpse stage. `DIV-2153`: one cue per frame
and entity. `DIV-2154`: the original attenuation words are not carried; the
channel percentages stay. Unknown: whether the client's stored health can
differ from the server's at a zero-damage message (`ANIM-125`). `DIV-2155` and
`DIV-2156` are returned unused.
