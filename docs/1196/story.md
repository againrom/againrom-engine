# story1196 — the roster survives a SAV round trip

## Player result

A character's stats, worn set and carried pack survive a city SAV round trip
unchanged, at the same fidelity story1194/1195 already measured and
disclosed. This story was held open for `SAV-1031`; that research answer is
now read against the exporter and importer, and it changes no code. The
measurement below is this story's own result.

## What this story checked

`SAV-1031` (knowledge k54, `cd knowledge && go run ./tools/claim SAV-1031`)
answers two questions this story's brief posed: does our exporter write
post-fold values where the original does, and does our importer re-apply a
fold the file already contains.

**Exporter.** `nativeCityHumanFromDerived` (`pkg/game/nativecityhuman.go`)
writes `h.Attack`/`h.Defence` (the file's `+0xa6`/`+0xbe` spans) from `next`,
the result of `h.Derive()` — already-folded, equipment-included values — not
from a raw base the reader would have to refold. `HumanState.foldModifier`
(`pkg/data/humanstate.go:261`) folds exactly five scalars: Speed, Capacity,
HealthMax, ManaMax, Sight. `SAV-1031` independently names the same five file
offsets (`+0x8c`, `+0x92`, `+0x96`, `+0x9c`, `+0xa4`) as `HERO-MOD-016`'s fold
destinations inside the file's fourteen-word stat run. The match is by stat
name, taken from `HERO-MOD-016` itself, which names each destination beside its
offset — speed, capacity, healthMax, manaMax, sight. CORRECTION (F3,
pipeline/reviews/story1196-review.md): this story first derived the match by
comparing the claim's offsets against this code's own field order, which is
againrom code interpreting ROM1 offsets and is what B1 forbids; it also could
not place the fifth, because Sight is not in that array. The conclusion is
unchanged and the derivation is now the claim's own naming. `foldModifier` also
assigns rather than adds `Attack.ElementalKind`
(`h.Attack.ElementalKind = m.Attack.ElementalKind`), matching `SAV-1031`'s
"one field is assigned rather than added" for the 24-byte layout. `h.Base`
(the file's `+0x114` span) is written directly from `member.Hero.Skill`, not
reconstructed by subtracting `Modifier` from `Attack` — consistent with
`SAV-1031`'s finding that "live minus modifier is not in general the base."

**Importer.** `HumanState.Derive` carries its own doc comment, unchanged by
this story and already correct: "LOAD does not call it." `Hero()`
(`pkg/data/humanstate.go:287`) and `Derived()` both read `h.Attack`/`h.Body`
etc. straight back with no addition of `Modifier`. `cityHumanState`
(`pkg/game/originalhuman.go:51`) reads the raw `Attack`/`Base`/`Defence`/
`Modifier` spans into `data.HumanState` with no arithmetic at all. Neither
side refolds.

**What `SAV-1031` does not settle.** It names the fold destinations inside
the fourteen-word stat run but not which of the other nine words in that run
— including Body, Reaction, Mind and Spirit, the four stats DIV-1321 is open
on — carry an original base or an original live value; it corroborates the
mechanism `HERO-MOD-016`/`UNIT-CTOR-004` already named, at Medium confidence
on that naming itself, and does not extend it to those four. DIV-1321's own
open question is unchanged; `docs/DIVERGENCES.md`'s DIV-1321 row now cites
this check so a later story does not redo it.

One further Unknown this story leaves open, named here because `SAV-1031` is
the first claim to give the field structural weight (F4,
pipeline/reviews/story1196-review.md). On the approximate native writer only,
`nativeCityHumanFromDerived` fills the base copy at the actor's `+0x114` from
`member.Hero.Skill` alone: the six skill words are written and ToHit, the
damage bytes, the active byte and the elemental triple go out as zero. The
export guard never compares that span, so nothing in this project detects it.
No promoted claim names a reader of those fields, so nothing is demonstrated to
be lost — but nothing establishes that they are unread either, and `SAV-1031`
grades the base naming itself Medium and does not exclude a reading in which
the actor's `+0x114` is a second live copy. The retained-document path is
unaffected: it preserves the real base copy from its own source.

## Measurement

Instrument: `pkg/game/savroundtrip1195_corpus_test.go`
(`sessioncorpusaudit` tag), run unmodified on this branch, no code changed.

AGS corpus (the owner's `engine/saves`, 105 files, EN assets):

    SAV-ROUNDTRIP-AGS-CENSUS discovered=105 round-tripped=0 disclosed=13 refused=92 mismatched=0 comparator-exercised=13
    SAV-ROUNDTRIP-DISCLOSED AGS-corpus completed missions=13 (ceiling 13, not modeled in sav.CampaignProjection; EXP-0375 open)
    SAV-ROUNDTRIP-DISCLOSED AGS-corpus fame=9 (ceiling 9, DIV-1319)
    SAV-ROUNDTRIP-DISCLOSED AGS-corpus hero record=2 (ceiling 2, DIV-1321)
    SAV-ROUNDTRIP-DISCLOSED AGS-corpus offered mission=13 (ceiling 13, DIV-1322)

Original `.sav` corpus (`gameversions/saves`, 102 of 103 readable, EN
assets), the direction that actually exercises `SAV-1031`'s subject —
genuine ROM1-authored roster bytes read by our importer:

    SAV-ROUNDTRIP-ORIGINAL-CENSUS discovered=102 round-tripped=94 disclosed=0 refused=8 mismatched=0 comparator-exercised=94

Both are unchanged from the baselines `sav1195AGSBaseline`/
`sav1195OriginalBaseline` already commit: same round-tripped counts, same
refusal reasons at the same counts, zero mismatches on either corpus. No new
field is lost on either corpus; the four
disclosed families (completed missions, fame, hero record, offered mission)
are exactly story1194/1195's own set, at the same ceilings.

What this census does **not** do is corroborate "the importer does not
refold". CORRECTION (F1, pipeline/reviews/story1196-review.md): an earlier
draft of this document said it did. `TestSAVRoundTrip1195OriginalCorpus`
exports to AGS, not SAV (`savroundtrip1195_corpus_test.go`, `refuse("EXPORT-AGS",
...)`; every refusal in the run above carries that tag), and both sides of its
comparison descend from one `RestoreOriginal`. A refold would therefore appear
identically on both sides and the census would still print `mismatched=0`. The
instrument is structurally blind to the hypothesis. The finding that the
importer does not refold rests on the code reading above and on nothing else
here; the census establishes that no new field is lost, which is a different
statement.

The census's "hero record" comparison (`sav1195MemberDiff`,
`a.Hero != b.Hero`) reaches `data.HumanState.Hero()` — Body, Reaction, Mind,
Spirit and Skill — the exact four stats DIV-1321 is open on and `SAV-1031`
does not name. The two files it still flags, both on member 0, are the same
one-point Body drift DIV-1321 already discloses; nothing in this story's
reading changes that.

Unrunnable-script-node census (`cmd/missionrun -mission <N> -trace -ticks 1`,
EN assets, `grep -c UNSUPPORTED`), the milestone measure this branch's own
diff cannot move: mission 10 = 0, mission 20 = 0. This story's diff is two
Markdown files, no `.go` file, confirmed by `git diff --stat` against the
merge base (`9abb733`); the count is unchanged from master by construction,
not merely by remeasurement.

## Result

Nothing is lost beyond DIV-1319/DIV-1321/DIV-1322's existing disclosure. No
new divergence row is needed; DIV-1321's `Revisit condition` cell is updated
in place to record that `SAV-1031` was checked against it and does not
settle it. No code changes. Closing on this measurement rather than
inventing work, per this story's own brief.

## Touched surfaces

`docs/DIVERGENCES.md` (DIV-1321 row, one cell), `docs/1196/story.md` (this
file). No production code, no test, no ledger entry beyond the one cell
above.

## Reserved numbers returned

DIV-1326, DIV-1327 and DIV-1328 were reserved for this story. All three are
returned unused: the measurement found nothing new to disclose.

## Open debt

DIV-1321 stays OPEN. Its four stats (Body, Reaction, Mind, Spirit) still have
no promoted claim naming whether the original's own city Human record holds
the effective or the base value for them; `SAV-1031` names the fold
mechanism for the other stat words and the 24/22-byte layouts but not these
four. A future research question, not an implementation gap this story can
close.
