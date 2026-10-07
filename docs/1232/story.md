# 1232 — current spell duration in SAV

## Intent

An original-loaded mission can retain a Poison rule with effect duration 8
while the installed rule has duration 128. The first current SAV must keep the
live rule, its derived cloud payload, and the rule used by a later new cast.

## Authority

The owner's `save-20260908-223315.ags` (SHA-256
`F9FC8630477875590E6A2E8AF26A94E8E662CE9D085FE705174DAB46D42618EF`)
is a local read-only source. Its imported Poison rule has duration 8 and its
live cloud projects `E40=0x0008fffc`. The lawful EN and RU installed rules have
duration 128. This is a current-state preservation result; the original rule
for constructing a later Poison cast remains a separate research question.

## As built

- `CurrentSpellPolicy.EffectDuration` is an optional 16-bit value. The producer
  writes it only when the live rule differs from the installed rule. An absent
  field in an older SAV uses the installed duration; a present zero remains
  zero.
- LOAD applies only that duration deviation before comparing native area
  payloads with their rule. The later continuation still validates and applies
  delivery, speed and duration. It does not move unrelated restoration or
  force a payload to be derived. An edited ordinary `AE44` stays explicit.

## Proof

- `TestReleaseCurrentPoisonDurationSAV` reads the exact owner AGS through the
  production mission LOAD, writes two ordinary SAVs, cold loads each, checks
  duration 8, derived Poison `E40=0x0008fffc`, and compares 18 continuing
  pulse ticks. The first and second normalized SAV World hashes match. The
  test separately checks 36 nonhuman capacities normalized from 0 to 300;
  source actors have no other field differences in this witness. It does not
  assert source-to-first-SAV World hash equality. Both lawful installs pass.
- The controlled `pre92-poison8-next-cast` EN/RU witness now saves and cold
  loads SAV before a new MapAttack Poison cast. Its attachment starts at
  counter 7 from duration 8, rather than the installed duration 128.
- Simulation tests check two native-area cycles, exact derived payload,
  pulses, an edited ordinary payload, absent old-field default, explicit
  zero, repeat application, and atomic refusal of duplicate policy rows.

## Open debt

- The new build reads older SAVs through the absent-field default. An older
  build reading a new SAV is not promised compatibility and may reject the
  unrecognized field.
- The original game's LOAD and future cast of the newly produced SAV have not
  been exercised here. Original-runtime acceptance remains Unknown.
