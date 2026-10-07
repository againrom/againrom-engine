# City SAV export stops refusing the owner's own saves

The owner's own 13 city saves refused `saveconvert -to sav`: 9 for an older-AGS
projection with no captured score history (`fameForOriginal`), 4 for an
imported campaign whose live provenance (`f.originalCity`) did not survive
into the AGS a session's SAVE last wrote, even though `Town.progress` still
proved the campaign was imported. Both refusals were correct about the gap.
Neither served the owner: a refusal also removes his ability to try the file
in the original game at all, and AGS already exists for an exact checkpoint.
His direction: write the best representation and disclose what it
approximates, rather than refuse.

`fameForOriginal` no longer refuses on unknown score history. It never needed
to: the imported-city writer already preserves a retained document's own
score-event scalar untouched when the count is unknown rather than zeroing
it, and a source-free document already defaults to zero for both fields. The
refusal stood in front of correct existing code and produced a strictly
worse outcome for exactly the state it could not recover (DIV-1319).

`ExportNativeCitySave` now also accepts an imported campaign whose provenance
is gone (`Town.progress != nil`, `f.originalCity == nil`) instead of refusing
outright, and reconstructs the city from current live state — the same
construction a genuinely native campaign already used (DIV-1320).
`nativeCityHumanState` gained an approximate mode, used only on that one
fallback path, that writes a member's current-stat basis instead of refusing
when a retained original Human record has no live source continuity; a
genuinely native campaign's strict checks (`OriginalHuman != nil`,
`LiveLoad` exact-reproduction) are unchanged (DIV-1321). `citySaveAdvisory`
now also accepts the one further shape actually observed on the owner's own
saves — a settled, open town's advisory naming the campaign's own next
`Campaign.Main` entry after the current chapter, right after finishing a
chapter's main mission — and silently resets it on export, exactly like an
ordinary Offered-0 LOAD already resets it (DIV-1322). It keeps refusing every
other Offered value on purpose: the EN+RU release gate's own
`TestReleaseNativeTownSaveShopBaredSessionRefusesAndAGSFallbackRoundTrips`
uses an unrecognized Offered as its backstop against a dismissed chapter
companion `nativeCityDismissedCompanion` cannot see on its own (that check is
scoped to the current chapter, not one already passed); an unconditional
accept was tried first, found this and three other release tests, and was
narrowed back down to the one disclosed shape before landing.

Unblocking the native-city fallback surfaced a real, previously-unreachable
bug: `filterOffered`/`filterOfferedPaired` excluded already-taken building
offers but not already-won missions, while the children-record construction
they feed does exclude both. Three of the four newly-reachable saves
failed their own cold load over it
("...has no live main or retained-child record") until both filters also
excluded `t.won`.

Disclosure is two places, per the owner's instruction. Each choice has a
`docs/DIVERGENCES.md` row (DIV-1319..1322, ledger IDs DIV-1319 through
DIV-1322). `citySaveNotice1194` (`pkg/game/citysavenotice1194.go`) composes
the same three conditions into a plain-language sentence appended to the save
dialog's own acknowledgement message through a new `ui.PreparedSave.Notice`
field, whenever an export actually used one of them; it is read-only and
never a reason to refuse. `TestCitySaveNoticeNamesEachApproximation` checks
each row's own trigger condition names itself and nothing else.

Authority: the owner's own direction in this story's brief overrides the
refuse-on-gap default the earlier fame/native-city/advisory code applied;
where a field has no defensible original value, the code says so in the
ledger and names the choice as this project's own, not a reconstruction.

Proof: `saveconvert.exe -to sav` (built from this branch, EN assets) over
every file in `engine/saves/*.ags` (105 files) now exports exactly the 13
owner saves that used to refuse for the two in-scope reasons — five
city-shaped saves and eight timestamped town saves, named nowhere in this
tree because they are the owner's own data — and every one of the 13
resulting `.sav` files
cold-loads cleanly through `savecheck.exe ... load`. The remaining 92
world-path refusals are unchanged in count (79+8+4+1) and in exact message,
confirming no incidental unblocking and no new refusal; they stay out of
scope (milestone M7). Before: 0/105 export (`review/sav-export-census/
RESULT.md`, engine main 86d55b0). After: 13/105 export, 92/105 still refuse
for their own stated, examined reasons.

story1195's own round-trip census (`pkg/game/savroundtrip1195_corpus_test.go`,
`sessioncorpusaudit` tag, landed on main) checks more
than export success: it reloads the exported SAV and compares roster and
mission state against the AGS original. Run once on this reconciled tree over
the same 105-file corpus (`go test -tags sessioncorpusaudit ./pkg/game/ -run
TestSAVRoundTrip1195AGSCorpus`), its baseline six refusal-reason counts
(79/9/8/4/4/1) move by exactly the two this story targets: the 9 "unknown
campaign score history" and 4 "native city save requires a native campaign"
reasons both drop to 0; the other four (79 never-imported worlds, 8 local UI
settings, 4 Group graph lifecycle, 1 camera) are unchanged, and refused falls
105→92 as this story's own proof above already showed by a different count.
Under this stricter check none of the 13 comes back identical on every
compared field, and each difference is now named rather than counted. The
census reads `SAV-ROUNDTRIP-AGS-CENSUS discovered=105 round-tripped=0
disclosed=13 refused=92 mismatched=0 comparator-exercised=13`: all 13 reach
the comparison, and every field any of them loses is one this project
discloses, names a row for, counts, and caps.

| field | files | disclosed by |
|---|---|---|
| completed missions | 13 | not modeled in `sav.CampaignProjection` |
| offered mission | 13 | DIV-1322 |
| fame | 9 | DIV-1319 |
| hero record | 2 | DIV-1321 |

The completed-mission set is lost on read and on write for one reason:
`sav.CampaignProjection` models no such set, so no writer has anywhere to put
one and no reader can recover it. That is an absence in this project's model
rather than a writer dropping a field, and it is not this story's doing — none
of the three files involved appears in this story's diff. Whether the
original's own SAV format carries such a field is a separate, open ROM1
question: no promoted claim settles it, this codebase is not evidence about
ROM1, and EXP-0375 asks it directly. No divergence row covers the loss because
a DEVIATION row has to state what ROM1 does, and nothing establishes that
yet. Under the owner's ruling
that campaign position and the Valuable Documents journal are what must
survive, it is disclosed rather than fixed. Campaign position itself is
identical across all 13 pairs, and the document collection survives every
round trip in the original corpus.

Two losses this story was first reported to cause turned out to be the
instrument measuring itself. The roster `Worn`/`Carried` flags on all 13 and
the carried-pack flags on 6 members were both artifacts: the comparator read
the lowest-precedence of four equipment representations, and then compared
three fields the shipped table answers for an item code rather than state the
save carries. Seat hotfixes `d5cf8f7` and `c93425c` corrected both on main,
measured against the original corpus, which stays at `mismatched=0`
throughout. What actually remains on the roster is the hero record on 2 files,
disclosed by DIV-1321.

The instrument's committed baseline now records the gain instead of the state
before it. Its round-tripped floor rises from 0 to 13, so all 13 files falling
back to refusing fails rather than passing silently, and the two reasons this
story removed are deleted from the refusal histogram rather than left at 9 and
4 -- `sav1195CheckBaseline` reads an absent reason as a ceiling of zero, so
either one returning fails on its first file. Without both edits the
instrument would still have passed with every file this story unblocks
refusing again.

Gates from the merged branch (origin/main `7bd6c6d` reconciled in): `gofmt`
clean except the pre-existing, untouched `pkg/game/installtext.go`;
`go test -trimpath -count=1 ./...` all packages ok; `check-no-game-assets.sh`
clean; one `check-release-tests.sh` invocation over EN and RU together —
`ok (312 of 312 ran, 0 lacked a subject)` on both roots; `check-preserved-
installs.sh` — `ok — 554 file(s), every root as recorded`. The first
release-tests run (before the DIV-1322 narrowing described above) failed 5
distinct tests on both roots; all 5 pass after the fix, with no other change.

Open debt: DIV-1319 and DIV-1320 stay OPEN — nothing recovers the discarded
original FAME dwords or retains `f.originalCity` losslessly across every
AGS re-save; DIV-1321 stays OPEN for the same reason, one level down (a lost
Human's modifier history). DIV-1322 is ACCEPTED: the advisory field is
write-only, confirmed by read-site audit, so losing it on SAV round-trip has
no consumer to matter to. The 92 world-path refusals (never-imported worlds,
UI-settings/camera/lifecycle gaps) are unchanged and explicitly out of scope
here; they are milestone M7. story1195's stricter round-trip check found
that all 13 exports this story unblocks lose their completed-missions list.
That set is absent from `sav.CampaignProjection` on both sides, so it is
missing from this project's model rather than dropped by a writer. It carries
no divergence row because a divergence row states what ROM1 does, and whether
the original's format holds such a field is exactly what EXP-0375 is open on.
Under the
owner's ruling it is disclosed rather than fixed, and the instrument now caps
it: `sav1195AGSDisclosed` names the field, counts the files it reaches, and
fails if it reaches one more.

The roster losses first reported alongside it were the instrument measuring
itself, and both are closed rather than deferred. `Worn`/`Carried` flagged on
every member of all 13 because the comparator read the lowest-precedence of
four equipment representations; the carried pack flagged on 6 members because
it compared three fields the shipped table answers for an item code. Seat
hotfixes corrected both on main against the original corpus, which stays at
`mismatched=0`. What genuinely remains is the hero record on 2 files, which
DIV-1321 discloses and whose resolution is a ROM1 question: whether the
original's own city Human record holds the effective or the base stat. No
promoted claim settles it, so no writer here may guess.
