package storyguard

// Committed is the one committed record every ratcheted count in this
// package is checked against. Regenerate it with:
//
//	go run ./internal/storyguard/cmd/measure
//
// then read every number before pasting it: measure regenerates all eight at
// once, so a routine paste can carry a rise nobody looked at. A count may only
// fall, with one stated exception: CommentBytes may rise for a doc comment on
// genuinely new code, and only by saying which file added it in the same
// commit. Every other rise is a regression Check reports. Any fall requires
// this file to be updated in the commit that caused it.
//
// The six comment forms stand at ZERO. There is nothing left to ratchet down:
// every one of them is now an absolute, and the only way the count can move is
// a regression. A comment-only change proves it moved no code with
// scripts/check-comment-only-change.sh.
//
// CommentBytes rose a further 5512 bytes over three files, all doc comments
// on genuinely new code witnessing the two hints the merged tree still left
// unscreenshotted: pkg/ui/tooltip.go's ComposeTooltipHint (the shared paint
// seam) and pkg/ui/tooltip_targets.go's MonsterSpellHint and MapListHint (the
// two hint-line wrappers), each exported only for the witness below under
// the same thin-wrapper idiom pkg/game/panelchars.go already established;
// and cmd/tooltipshot/main.go, a new command whose package doc states why
// neither hint's own screen (ScreenPicker, the mission screen) has any CPU
// composite path in this build to screenshot instead, and whose function
// docs state that every input it feeds those wrappers is real install or
// mission data, not a fixture.
//
// CommentBytes rose a further 549 bytes over one file: internal/archtest's
// own dag.go, registering the new cmd/tooltipshot in the architecture
// allow-map with the same reasoning style every other cmd/* row already
// carries, plus this paragraph's own bytes.
//
// Merging origin/main (the town-SAVE fix above, +6018 bytes over the shared
// 7687012 ancestor) into wt-story-1209-special-tooltips (the tooltipshot
// witness and its archtest registration above, +6061 bytes over the same
// ancestor) is two independent deltas over one ancestor, not additive with
// each other; a fresh measure on the merged tree, including this
// reconciliation paragraph's own bytes, gives 7700697.
//
// CommentBytes rose a further 3822 bytes, all genuinely new code from this
// correction pass's own spellbook-popup getter (composing the mission book
// and the shop's Book toggle from spells.txt names and installed labels,
// gated on availability): pkg/game/installtext.go (417), pkg/game/spell.go
// (686) and its own spell_test.go (466), pkg/ui/tooltip_runtime.go (321),
// pkg/ui/shopscreen.go (214) and pkg/ui/words.go (412), plus two new
// regression witnesses, pkg/game/spellpopup_release_test.go (834) and
// pkg/ui/spellpopupgate_test.go (472). This rise landed without a baseline
// update at the time; it is recorded here, in the same commit as the merge
// below, rather than left silently unratcheted.
//
// Merging origin/main (the attack-cycle latch and dying-pursuer's frozen
// order above, ending 7698265 over the common 7687974 ancestor) into
// wt-story-1209-special-tooltips (this correction pass's own rise above,
// ending 7704519 over the same ancestor) is two independent deltas over one
// ancestor, not additive with each other; a fresh measure on the merged
// tree, including this reconciliation paragraph's own bytes, gives the
// number below.
//
// Merging origin/main (the attack-cycle latch and dying-pursuer branch,
// ending 7698265) into this branch (the mission-worn-equipment branch,
// ending 7695411) sums the two deltas over the common 7687974 ancestor (this
// branch's own equipment-graph paragraph above and the other branch's three
// paragraphs above that), plus this note's own bytes. Re-measured on the
// merged tree rather than computed, per this file's own doc comment.
//
// Merging origin/main (the special-tooltip work above, ending 7715897 over
// the common 7698265 ancestor) into this branch (the mission-worn-equipment
// branch, ending 7706142 over the same ancestor) is two independent deltas
// over one ancestor; a fresh measure on the merged tree, including this
// note's own bytes, gives the number below.
//
// CommentBytes then rose a further 2235 bytes, this note included. Six files
// carry doc comments on genuinely new code: pkg/formats/sav/city_holdings.go
// (+523, applyCityCharacterGraphs and clearUnresolvedCityOwners),
// pkg/game/originalcity_return.go (+527, returnDerived and the worn-set
// comment that replaces the modifier comparison),
// pkg/formats/sav/city_holdings_test.go (+346, the owner Reference test),
// pkg/game/importedloot1172_controls_release_test.go (+278, the unclosed
// owner Reference test), pkg/game/cityequipmentreturn_test.go (+156, the
// rewritten equipment return witnesses) and
// pkg/game/importedreturn1169_release_test.go (+139, the current modifier
// subtest). pkg/formats/sav/city_semantic.go (+30) names the worn armour
// slots in containerOf's existing comment. pkg/game/cityholdings.go fell 767
// with savedCurrentGraphOwner removed. TestIdentCount fell 3: the sav test
// fixture cityHoldingsFixture and the unclosed owner Reference test no longer
// carry a number.
//
// CommentBytes then rose again, this note included, for the loaded town's
// current-state member writer. New code: pkg/game/originalcity_current.go
// (+2992, the writer's types and functions) and the new witnesses'
// pkg/game/citycurrentstate_release_test.go (+1673). Rewritten tests:
// pkg/game/originalhuman_native_test.go (+401),
// pkg/game/originalcity_sales_test.go (+284),
// pkg/game/originalwriter_route_test.go (+230),
// pkg/game/originalcity_persistence_test.go (+220),
// pkg/game/importedreturn1169_release_test.go (+201),
// pkg/game/originalhuman_test.go (+164),
// pkg/game/nativecity_release_test.go (+139),
// pkg/game/townreturn1168_release_test.go (+120),
// pkg/game/savedialog1173_release_test.go (+68) and
// cmd/saveconvert/main_test.go (+21) name the trigger that still refuses or
// the write that replaced a refusal; pkg/formats/sav/city.go (+137) documents
// the Difficulty field, and pkg/game/savroundtrip1195_corpus_test.go (+161)
// names why its AGS floor and two ceilings rose. pkg/game/originalcity_return.go
// fell 1265 and pkg/game/originalsave.go fell 1311 with the return shape
// comparison and the refusals removed.
// Current SAV comments describe new ordinary-reader population, capture and
// binding logic in game/currentactoradmission.go, savdocument.go and resume.go;
// native effect ownership transfer in game/savarea_current.go, sim/currentareas.go
// and currentdeliveries.go; sparse policy in sim/currentworldpolicy.go; and their
// new current actor, item, campaign, area, delivery and policy controls. Their
// measured increase is new-code documentation, including this declaration.
// Three shared fixture helpers now have semantic names in all their callers.
// The clock restoration API in sim/currentworldpolicy.go adds new-code docs.
// Actor identity restoration adds API docs in sim/currentidentity.go and
// game/savactoridentity.go. The source-copy fixture now has a semantic name.
// Ordinary character reading, current party reconstruction and coherent carry
// views add new-code documentation in sav/document_characters.go,
// game/currentpartyread.go, currentpartybase.go, mapload/currentweapon.go and
// mapload/carry.go, including this declaration.
// Current actor presence, terminal motion, exact item topology and area payload
// continuation add documentation in game/current*.go, sim/current*.go,
// sim/sourceequipment*.go and their new controls. The common snapshot fixture
// now has a semantic name; coordinator and command-literal counts also fell.
// New-code comments in game/savcurrentitems.go and currentscriptbindings.go
// describe current Sack units, retired-owner edges and late party-role binding.
// save_test.go names the additive Player bit in its current descriptor proof;
// this declaration records their measured documentation increase.
// TestIdentCount corrects the stale 5832 literal introduced at 1dae626: that
// commit measured 5848 identifiers. Three stale timer-helper references were
// removed after the live World stopped materializing those bytes; the current
// tree measures 5849. The correction records the existing population rather
// than admitting new debt.
// CommentBytes rose with the documented current-SAV producer, motion and
// city-Human ownership code, plus the generated city Spell supplement and
// explicit empty kind-41 attachment branches, including this note.
//
// CommentBytes rose with cityobjecttopology.go's HumanTails field note,
// explaining that ordinary SAV residue is rebuilt and is not a second actor
// value, plus this baseline note.
// Current StructureUse continuation and fresh-structure bindings add documented
// producer state. TestIdentCount also records the already-landed six-test rise.
// Current SAV city and item continuation corrections add documented new code.
// CommentBytes rose a further 2935 bytes fixing the generated mission SAV's
// mover RotationSpeed, publication mask, block-plane occupancy bits and
// AutoGetMission fields: new doc comments in pkg/sim/currentmotion.go,
// pkg/game/nativecity.go, pkg/game/currentworldspatial.go,
// pkg/game/generatedworldspatial.go and pkg/game/currentsave.go; the new
// regression witness pkg/game/generatedmissionsavfields_test.go; and a
// matching mirror-fix comment in the pre-existing
// pkg/game/generatedmissions_release_test.go, whose own pre-save campaign
// projection needed the same AutoGetMission derivation to keep predicting
// what SAVE now writes.
//
// CommentBytes rose a further 720 bytes correcting that publication-mask fix's
// own scope. nativeCityToken is shared by the mission-SAV and native-city
// document builders; writing mask 2 inside it unconditionally broke
// TestReleaseNativeCityPlayerConstruction, whose fresh native-city Human
// token must legitimately stay unpublished. pkg/game/nativecity.go's doc
// comment is removed with the reverted write; pkg/game/currentworldbuild.go,
// pkg/game/currentworldspatial.go (two call sites), pkg/game/savcurrentitems.go
// and pkg/game/generatedworldspatial.go each gain a scoped SAV-1093 comment on
// an explicit per-call-site mask write instead. The pre-existing
// pkg/game/nativecityplayer_test.go gains a comment explaining why its shared
// checkPlayerWireN4 helper now expects mask 2 only from the current-mission
// producer (ExportCurrentWorldSave), not the native-city one
// (ExportNativeCitySave), plus this paragraph's own bytes.
//
// CommentBytes rose a further 9830 bytes for a generated mission SAV review
// pass's five findings: the mover passability mask derived from domain, the
// Unit Capacity and own-weight fallbacks, the building-type word and
// block-plane sweep scoping, the AutoGetMission wiring, and the AI-start
// post/mover-constant repair. Each site's fresh bool parameter
// (pkg/sim/currentmotion.go, pkg/game/savactorproject.go) needed a doc
// comment distinguishing the one-time fresh-generation construction it gates
// from the ordinary SAVE and live-sync paths it must leave untouched, since
// getting that scope wrong silently changes hashed state for every actor
// already on disk. TestIdentCount stays at the prior value: the one new
// story-numbered test name this pass introduced was renamed instead of
// counted as debt.
//
// CommentBytes rose a further 2326 bytes for that same review pass's own
// correction: the block-plane sweep scoping above was still overwritten by
// the shared per-cell terrain builder every export runs after it, so an
// explicit, caller-supplied Snapshot field replaced the implicit "no prior
// document" signal that scoping had shared with unrelated ordinary saves.
// currentsave.go, worldsave.go and currentworldspatial.go each gained a
// paragraph on why the new field is an explicit opt-in rather than an
// inference, since the inference this replaces was silently changing hashed
// terrain state for a save this pass's own review never touched. The three
// pinned envelope-descriptor hashes in save_test.go each gained one sentence
// noting which additive field moved them, matching that file's own
// established convention.
//
// CommentBytes rose a further 1649 bytes finishing that same correction: the
// prior paragraph's count was taken before worldsave.go's currentWorldDocument
// gained its own one-line doc on the fresh parameter, and before
// currentworldspatial.go's projectCurrentTerrain doc comment was rewritten in
// full to state the new NativeMissionTerrain semantics rather than the stale
// implicit-fresh wording it replaced, plus this paragraph's own bytes.
//
// CommentBytes rose a further 1314 bytes closing the review's own remaining
// byte-value gap in that same block-plane delta: currentSpatialPlanes' own
// domain-grid read never folded domain 2's scenery/border bit the way
// originalstructures.go's loader does, so a genuinely fresh world's delta
// rows carried the right cells and shape but the wrong Static/Dyn bit2 within
// them. The new fold in currentworldspatial.go's currentSpatialPlanes gained
// its own doc comment explaining why it reads the raw grid value instead of
// relying on grid bit3, plus this paragraph's own bytes.
//
// CommentBytes rose 7660 bytes for the sharp-bilinear hotfix: pkg/ui/sharpblit.go
// (owner decision method B) and pkg/ui/sharpblit_test.go are genuinely new
// files, plus one doc paragraph at each final-blit call site it touches.
//
// CommentBytes rose a further 3444 bytes closing the review's own remaining
// footprint-cell gap and reconciling this branch with the concurrent
// sharp-bilinear hotfix merged into it: an intermediate attempt folded bit 2
// unconditionally on every attachment cell, which over-set it on cells the
// reference encoding leaves as plain record cells; currentSpatial's own
// attachment loop now guards the fold on the terrain's own ground-block bit,
// with a doc comment citing the reference file and its own exact 112/35-cell
// split. The two branches' own prior totals do not sum directly, since each
// was measured against its own point in history; this value is the merged
// tree's own direct re-measurement.
//
// CommentBytes then fell by 728 bytes before this paragraph's own text,
// correcting the Capacity/own-weight repair's own scope: the prior
// paragraphs' "freshMission" signal, true for any export whose caller
// supplied no prior document, broke two dozen pre-existing fixture tests
// that build a synthetic World with a deliberately zero live Capacity and
// assert an exact SAVE/LOAD round trip, because a session's first SAVE of
// such a world also has no prior document. worldsave.go's
// currentWorldDocument now reuses the same NativeMissionTerrain signal F3's
// own block-plane scoping already uses, removing the separate freshMission
// parameter and its call-site plumbing entirely; the doc comments
// explaining why are shorter than the ones they replace, net of this
// paragraph's own bytes.
//
// CommentBytes rose a further 4875 bytes fixing two owner-witnessed combat
// bugs in the generated-mission SAV kit's own hero record: it was caught in
// the AI-start mover/order/state repair (excluded by an owner gate in both
// currentmotion.go and savactorproject.go, with the DIV-1388 evidence trail
// recorded at each site), and chargenProfile's unresolvable-base-row
// fallback silently named every request a mage regardless of the archetype
// asked for (DIV-1389). New test file generatedmissionherofighter_test.go
// and the removal of a throwaway probe test file leave TestFileCount and
// TestIdentCount unchanged.
//
// CommentBytes rose a further 908 bytes updating two pre-existing
// chargen_test.go assertions (one renamed) that pinned the DIV-1389 fallback
// bug's own zero-Profile output as correct; they now assert the requested
// archetype's own Fighter bit instead, with a doc comment explaining why.
//
// CommentBytes rose a further 7198 bytes for the documents-text hotfix:
// pkg/ui/documents.go's docTextColor and docParagraphIndent each gained a
// doc comment naming the reference screenshots and the measurement each
// value was bounded by, pkg/ui/documentwrap_test.go is a genuinely new file,
// and pkg/ui/documents_test.go's own fixture comment grew explaining why
// every line in it is indented but none is justified.
//
// CommentBytes rose a further 24051 bytes for owner decision method G, the
// smoothed-text presentation overlay (DIV-1385/DIV-1386). New files, each
// carrying its own doc comments: pkg/render/text/capture.go and its
// capture_test.go (the recorder/deferral hook Font.Draw now opens a window
// into), pkg/render/textsmooth/textsmooth.go and its textsmooth_test.go (the
// bicubic resize, contrast remap and compositor), pkg/ui/textsmoothing.go
// and pkg/ui/missiontextsmoothing.go (the App/Viewer wiring and the
// markCapture/shiftCapture and beginTextCapture/endTextCapture helper
// pairs), and pkg/ui/textsmoothing_test.go (the offset-paste placement and
// smoothing-off witnesses). Existing files gain doc comments on genuinely
// new code: pkg/render/text/text.go (Draw's capture-window paragraph),
// pkg/ui/app.go and pkg/ui/viewer.go (the new fields and the capture calls
// at each offset-paste and HUD site this story wired), pkg/ui/tooltip_runtime.go,
// pkg/game/tipstore.go and pkg/game/frontend.go (the TextSmoothing
// preference, mirroring TipsMode's own shape), and
// internal/archtest/dag.go and composition_baseline.go (registering the new
// leaf package and the one new PersistenceContext field), plus this
// paragraph's own bytes.
//
// CommentBytes rose a further 333 bytes: pkg/ui/townshell.go's
// drawCharacterPaneBody now shifts the statistics card's captured glyphs by
// its own paste offset (capture.go's ShiftCaptured doc already named this
// site; the wiring itself was the gap), with a doc comment explaining why.
//
// CommentBytes rose a further 1310 bytes:
// TestDrawCharacterPaneBodyShiftsStatisticsCardGlyphs
// (pkg/ui/textsmoothing_test.go), the regression test for the paste-offset
// fix above, plus its own doc comment on why a non-empty PaneRect and a
// per-call Color comparison are both load-bearing to the test.
//
// CommentBytes rose for three save hotfixes and the text-overlay kernel fix:
// U4C bit 3 carries live off-map presence both ways, an authored corpse can
// take a saved dead tuple, a mission SAVE writes each companion's PC_ row,
// and the glyph overlay keeps Draw's shade under painted coverage.
//
// CommentBytes rose for the text-overlay visibility hotfix: DrawCall.Under,
// text.Record/Append/Capturing, textsmooth.Settle and settleTextFrame gained
// doc comments on new code, the cached tavern inspection, mission panel,
// mission card and menu glyph fields explain why their glyphs are re-captured
// each frame, and three new tests carry their own doc comments.
//
// CommentBytes rose for the book school hotfix: shop.go's school-order
// comment names the Spells rows it follows, and the book release test
// explains its per-school "Protection from" check.
//
// CommentBytes rose for the restored-hero hover hotfix: inspectionUnitPicture
// says why a recorded figure wins over the class test, and its release test
// states the overwritten human type id it covers.
//
// CommentBytes rose 4157 bytes for the generated hero health-maximum fix:
// chargenProfile's own doc comment (pkg/game/hero.go) explains the added
// HealthColumn field and its scope; two existing test comments
// (pkg/game/chargen_test.go) explain why their fixture Profile now also
// carries it; the two new files pkg/game/herohealthmaximum_test.go and
// pkg/game/herohealthmaximum_release_test.go carry doc comments on the
// regression, the release proof for both missions, and the ground-truth
// resave witness, plus this paragraph's own bytes.
//
// CommentBytes rose a further 1126 bytes closing the same gap in
// unitPicture, the sibling reader pushPortrait calls: portrait.go's doc
// comment explains why a recorded figure must win over the class test there
// too, and the new mission131roundtrip_release_test.go states what each of
// its two regressions covers.
//
// CommentBytes rose a further 2018 bytes in two files: savroundtrip1195_corpus_test.go's
// sav1195OriginalBaseline gained a paragraph tracing a later corpus
// addition's own AGS-export aggregate-cap refusal to three legitimately
// large Snapshot fields, none of them an engine-side duplicate, disclosing
// the one new refusal reason its own refusals map now carries; plus this
// paragraph's own bytes.
//
// CommentBytes rose a further 9834 bytes over a correction pass fixing three
// reported round-trip defects the prior text had wrongly closed. resume.go
// gained a doc comment on why its own Snapshot no longer restates the entry
// party into CurrentRoster (the actual cause of heroes drawing as NPCs after
// this engine's own SAVE+LOAD, and of the AGS aggregate-cap refusal that
// paragraph above disclosed). pkg/sim/sack.go and its own doc comment gained
// SackTokenValue, the ITEM-SACK-010 constructor this build's ground-sack
// writer had never called; savedobjectground.go, savcurrentitems.go and
// ground.go each gained a short comment on where it is now wired in.
// savapplication.go gained viewOriginFloor and originalViewOrigin with their
// own corpus-census doc comments (the original's own flat 8-cell scroll
// floor, and why an exact (0, 0) view is exempt). The new
// savapplication_viewfloor_test.go, and updated comments in
// legacyprojection_test.go, savapplication_projection_test.go and
// savroundtrip1195_corpus_test.go's own baseline paragraph (correcting its
// prior mechanism guess now that the true cause is fixed), state what each
// covers.
//
// CommentBytes rose a further 884 bytes fixing a SAV save refusal after a
// Group member's full decay: savdocument.go's snapshotSavedDocument gained a
// comment on why it now calls rootDeadActors before projectSavedGroups can
// rebuild live-only Group membership and drop a newly terminal actor's only
// root, and the new savgroupmemberdecay_test.go states what its regression
// covers.
//
// CommentBytes rose 2743 bytes over three files fixing the roster-duplicate
// reader hotfix, a file already carrying an entry-party id in Start.Roster
// from before the prior paragraph's writer fix existed: 404 in
// pkg/game/world.go's missionAppearanceArt doc, on why a current party
// member's own body now wins over a Start.Roster entry sharing its id; 511
// in pkg/game/resume.go, on why Snapshot's CurrentRoster loop now excludes
// an id already in CurrentPartyIDs so the duplicate does not survive a
// resave; 1185 in the new pkg/game/rosterduplicate_hotfix_release_test.go,
// whose own doc comment states the mechanism its regression covers; and 643
// in this paragraph.
//
// CommentBytes recomputed at 7809268 reconciling three independent lines
// landing on top of each other: the generated hero health-maximum fix
// (7793026), the sack/resume round-trip fix (7801334), and the
// roster-duplicate reader hotfix immediately above (7804077). As before,
// the merge keeps every paragraph in full; the measured total comes from
// internal/storyguard/cmd/measure's own fresh scan of the fully merged
// tree, not from adding the three branches' deltas by hand.
//
// CommentBytes rose 1421 bytes in the new
// pkg/game/hiredmercenaryroster_hotfix_release_test.go, whose own doc
// comment states why a tavern-hired mercenary's own class is unaffected by
// the roster-duplicate fix above: its id is never a Start.Roster key on a
// healthy load, and the same Start.IDs exclusion that protects a hero's own
// body also leaves a mercenary's own class alone on a duplicate-carrying
// one; 455 of the total is this paragraph.
//
// CommentBytes rose a further 1519 bytes merging the save-refusal and
// roster-reader hotfix branches: no code from either side moved, and this
// paragraph is the only new text, added to satisfy the merged tree's own
// re-measured total below.
//
// CommentBytes recomputed at 7812796 merging that combined hotfix line
// (7807017, itself reconciling the save-refusal fix, the mercenary
// regression test, and the roster-duplicate reader hotfix) into this
// branch, which already carried its own three-way reconciliation above
// (7809268, generated hero health-maximum plus sack/resume round-trip plus
// roster-duplicate reader). No code moved in this step either; every
// paragraph stays, and the number is internal/storyguard/cmd/measure's own
// fresh scan of the fully merged tree, not an arithmetic sum of the two
// sides' totals.
//
// CommentBytes rose for the periodic-lag hotfix: pkg/ui/pixellog.go is new
// (the pixel log that decides glyph visibility without a GPU readback) with
// its test pixellog_test.go; pkg/ui/textsmoothing.go documents the retained
// overlay, the eraser and the readback fallback; pkg/render/textsmooth
// documents the resample cache and Decide; pkg/game's new
// textsettle_release_test.go states what its regression covers. This
// paragraph is included.
//
// CommentBytes recomputed at 7820444 merging the periodic-lag hotfix line
// (7814333) into this branch's own prior total (7812796). No code moved in
// this step; every paragraph stays, and the number is
// internal/storyguard/cmd/measure's own fresh scan of the fully merged
// tree, not an arithmetic sum of the two sides' totals.
//
// CommentBytes rose a further 3994 bytes over the sole-review correction
// pass that traced the 145-versus-137 gap to a SAV-writer defect rather
// than a formula gap: currentworldbuild.go's currentActorSource gained a
// doc comment on why it now writes actor+0x130 from the live SkillXP sum;
// the new savu130corpus_test.go states the corpus proof it runs; the
// retargeted TestReleaseHeroHealthMaximumGroundTruthResave and the new
// TestReleaseGeneratedHeroAggregateExperienceMatchesItsSkillSlots each carry
// a doc comment on what they now cover; recompute.go's and start.go's own
// HealthColumn comments were corrected rather than left stale once a
// constructed fallback, not only a shipped row, could set that bit; plus
// this paragraph's own bytes.
//
// CommentBytes rose for the command-panel hotfix. The stretched-plaque
// comments left with pkg/ui/buttonplaque.go and pkg/game/buttonplaque.go;
// the new text states the TOWN-260 draw rule in the shop and tavern, the
// label feedback in pkg/ui/buttonfeedback.go, the SHOP-052 grouping in
// pkg/ui/groupdigits.go, and the witnesses in the new shop panel test and
// the updated tavern and generator release oracles. This paragraph is
// included.
//
// CommentBytes rose for the terminal-actor LOAD hotfix: sav.ActorRecord's
// TerminalActor field, the planner's ownsOriginalEntity rule and the
// admission note on why a terminal root's Stage 1 is not a dying body, plus
// one doc line on each of the two new tests. This paragraph is included.
//
// CommentBytes rose for the terminal-actor SAVE hotfix: the LOAD note on why
// unbound authored constructors retire before terminal rows import in the
// saved ID namespace, the writer and admission notes on a terminal actor with
// no SAV record, RetireUnboundConstructors' doc, and the new release tests'
// doc lines. This paragraph is included.
//
// CommentBytes rose for the equipment definition-row hotfix: the new
// pkg/game/itemdefinitionrow.go states the row rule and its claims, two
// one-line notes explain the writer and seed-identity changes, and the new
// row tests state what they cover. This paragraph is included.
//
// CommentBytes recomputed merging the command-panel, terminal-actor LOAD/SAVE
// and equipment definition-row hotfixes into this story's own correction
// pass. No code moved in this step; every paragraph stays, and the number is
// internal/storyguard/cmd/measure's own fresh scan of the fully merged tree.
//
// CommentBytes rose for the release-time hotfix: the smacker oracle test's
// notes on its parallel payload and oracle subtests and their bound, and the
// notes on the generated-mission and mission-130 betrayal release tests split
// into parts. This paragraph is included.
//
// CommentBytes rose for the companion-detach hotfix: the doc of the new
// withoutDeparted in pkg/game/frontend.go, the new mission-130 witness
// pkg/game/companiondetach130_release_test.go, and this paragraph.
//
// CommentBytes rose again for the new
// pkg/game/savwritercensus_corpus_test.go, the writer census over changed
// worlds: its doc comments state each rule it applies, the claim behind it,
// and the cause of each baseline and debt row. The rise includes this
// paragraph. TestIdentCount fell 29 because the Sacks byte walker's three
// shared identifiers took digit-free names the census could reuse.
//
// CommentBytes moved again in the census's correction pass and its merge of
// engine main: the census states each exact rule, its parallel walk and its
// change floors; the town writer notes why a town with unavailable
// provenance still writes its loaded options; the constructed-sack allocator
// notes why item and effect ids are reserved; World.Entity has its doc line.
// The number is internal/storyguard/cmd/measure's fresh scan of the merged
// tree. This paragraph is included.
//
// CommentBytes rose for the death-sack SAVE hotfix: the note on why the
// current-world Sack record keeps its constructor shape, the mission-130
// witness helper's doc, and this paragraph.
//
// CommentBytes rose for the tick-cost hotfix: World.EntityView's read-only
// contract, the roster-joiner helpers, the pooled step scratch, the join-gate
// test and this paragraph.
//
// CommentBytes rose for the dead-actor order-state hotfix: livingOnlyState's
// doc, the SAV writer's note on a body's retained order, the reload witness's
// doc and this paragraph.
//
// CommentBytes rose for the tavern roster presentation. pkg/game/taverntalk.go
// is new and documents the talk object's resolution, sheet and statistics
// gate (TAVERN-TALKPIC-016, TAVERN-TALKSTATS-017); pkg/game/tavern.go names the
// stock walk and its fitted id phase (TAVERN-ORDER-015, DIV-1410);
// pkg/game/towntavernart.go the optional talk sheets (TOWN-470);
// pkg/ui/groupdigits.go the sign rule (TOWN-469); pkg/ui/townshell.go the
// paint guard (TOWN-468); and the new focused and release tests state what
// each pins. The removed two-sheet cycling comment offsets part of it. This
// paragraph is included. Its correction pass adds the stock-unit talk cast,
// the talk panel cache, the plaque price font and fit test, the purse
// marker, the Human term and the sight word, each with its own doc line.
//
// CommentBytes rose for the shop staff re-equip hotfix: the shop check's note
// on a worn weapon's owned Spell, its witness's doc and this paragraph.
//
// CommentBytes rose for the town-to-mission registry check, which names where
// the next mission's registry comes from, and this paragraph.
//
// CommentBytes rose for the city held-Item owner rebind: its doc in
// pkg/game/cityobjectprojection.go, its witness's doc and this paragraph.
// The fighter-training round-trip comparison's Player owner link added two
// docs in pkg/game/originalhuman_release_test.go.
//
// CommentBytes rose for the mission-entry quest-document follow in
// pkg/game/citymissionseed.go, mapload.QuestDocumentHolder's doc, the
// companion witness's doc and this paragraph.
//
// CommentBytes rose for the town player-list dword: the rule's note in
// pkg/game/nativecity.go, its witness's doc, the rewritten sale oracle's
// docs and this paragraph.
//
// CommentBytes rises again for the town SAV Player-groups carry: the
// doc of pkg/game/citygroupcarry.go naming DIV-1411, the rebind note in
// currentsave.go, its two witnesses' docs and this paragraph.
//
// It rises once more for the hire groups: the doc of the hire-group fields
// and the rewritten group writer, the hire witnesses' docs and this
// paragraph.
//
// It rises for the town SAV map name: the docs of Town.lastMap and
// Snapshot.LastMap, the three map-name witnesses' docs, the envelope pins'
// notes and this paragraph.
//
// It rises for the party class at original LOAD: the note on the party
// member's class in the actor registry, its lookup's doc, the witness's docs,
// the registry fixture's note on its unarmed class, the hover and
// push-portrait witnesses' rewritten docs with their hero-band fallback
// helper, and this paragraph.
//
// It rises for script patrols in a mission SAV: the docs of sim's shared
// patrol order fields and PatrolOrder, of projectCurrentPatrols, of the
// patrol witness, and this paragraph.
//
// It rises for the formation group term: the docs of aloneSpeed and
// World.RateSpeed in pkg/sim/world.go, the rewritten moverSpeed doc, the
// docs of pkg/sim/formationpenalty_test.go and
// pkg/game/formationspeed_hotfix_release_test.go, and this paragraph.
//
// It rises for the pickup publication: the docs of pkg/sim/itempublication.go,
// pkg/game/sessionpublication.go, publishSessionEntry in
// pkg/game/cityobjecttopology.go, Viewer.PickupRows in pkg/ui/pickup.go, the
// witnesses in pkg/game/pickuppublication_release_test.go and
// pkg/sim/itempublication_test.go, and this paragraph. TestIdentCount falls
// with the rename of the itemOperationsWorld fixture.
//
// It rises for the Catmull-Rom final-frame scaler: the docs of
// pkg/render/catmullrom/catmullrom.go and its tests, pkg/ui/framesmoothing.go
// and its tests, pkg/game/framesmoothing_test.go, the FrameSmoothing
// preference in pkg/game/tipstore.go, the smoothingOff field in
// pkg/game/frontend.go, the frameSmoothingOff fields in pkg/ui/app.go and
// pkg/ui/viewer.go, the new leaf in internal/archtest/dag.go, and this
// paragraph. The rewritten call-site comments in pkg/ui/app.go,
// pkg/ui/viewer.go and pkg/ui/cutscene.go fall.
//
// It rises for the Prismatic Spray victim selector: the docs of the new
// pkg/sim/prismatic.go and pkg/sim/prismaticvictims_test.go, the new
// preparePrismatic doc in pkg/sim/spelldelivery.go, the party-mage spray
// notes in pkg/game/spellcontinuation_release_test.go, and this paragraph.
// applyPrismatic's rewritten doc in pkg/sim/celleffect.go falls, and so does
// the refusal-table doc in pkg/sim/weaponspell_test.go.
//
// It rises for the staff Prismatic Spray rays: the docs of the new
// sprayLiveFrom and observeCasts' new victim-list paragraph in
// pkg/game/spellbolt.go, the four new tests and their helpers in
// pkg/game/spellpath_test.go, one note in pkg/game/actionvisual_test.go, and
// this paragraph.
//
// It rises for a body's order at SAVE and LOAD: the docs of
// projectSavedGroups and the new projectDeadActorOrders in
// pkg/game/savgroupdocument.go, the body note in pkg/game/savactionwire.go,
// the residue note in
// pkg/sim/originalactions.go, the witnesses in
// pkg/game/deadpatrolsave_hotfix_release_test.go and
// pkg/sim/originaldeadorder_test.go, and this paragraph.
//
// It rises for silent test audio: the docs in pkg/ui/deviceoutput.go and
// pkg/ui/deviceoutput_test.go, and this paragraph.
//
// It rises for a dead unit's script answer: the docs of departedEntity in
// pkg/sim/script.go, controlSpiritCorpseHP in pkg/sim/world.go and
// CurrentTerminalActor in pkg/sim/currentterminal.go, the new
// pkg/game/currentterminalscript.go, the witnesses in
// pkg/sim/scriptdeparted_test.go and pkg/game/scriptdeparted_release_test.go,
// and this paragraph.
//
// CommentBytes rose a further 6289 bytes for one new tagged test file,
// the SAV round-trip gate (pkg/game/savroundtripgate_corpus_test.go): its doc
// comments state what each census key and comparison means, which visible
// field a LOAD resets by construction and why, and this paragraph.
// It rises again for the same file's split into two tests on parallel
// workers with a decay sample and a dying route: the docs of savGateCensus,
// savGateCost, savGateDecaySampled, the two tests and the new constants, and
// this paragraph.
//
// It rises for the gate's town route, field-path World diff and per-case
// baseline: the docs of savGateDiffer, savGateWorldFields, savGateTownRun,
// savGateCauses and the baseline reader and writer in the same file, the
// dropped-kill note in savGatePlay, scripts/check-sav-roundtrip-gate.sh's
// header, and this paragraph.
//
// It rises for a cloud over a cell with no record: the doc of the new
// currentAreaCellRecord and its insertion note in
// pkg/game/savarea_current.go, the witness in
// pkg/game/cloudcellsave_hotfix_release_test.go, and this paragraph.
//
// It rises for a transferred ally's owner on SAVE and LOAD: the doc of the
// new projectCurrentActorOwner in pkg/game/savactorproject.go, the new
// partyBasis in pkg/formats/sav/party.go, the fixture fields in
// pkg/formats/sav/program_test.go, the new pkg/formats/sav/partyowner_test.go
// and pkg/game/partyownerload_hotfix_release_test.go, and this paragraph.
//
// It rises for the install data front ends share within one process: the
// docs of the new pkg/game/installshare.go and pkg/game/installshare_test.go,
// of savGateSameWorld in pkg/game/savroundtripgate_corpus_test.go, the
// own-copy note in pkg/game/townexterior_release_test.go, and this paragraph.
//
// It rises for headless placement and pick-up through the engine's routes: the
// docs of the new takeSackUnderfoot in pkg/game/world.go, pickUp and primary
// in pkg/game/scenario.go, the new pkg/game/headlessreach_test.go and the
// helpers in pkg/sim/headless_test.go, and this paragraph. The interim
// HeadlessTeleport keeps its doc from before, beside HeadlessPlace.
//
// It rises for a cutscene stream that no call waits on while Play fills:
// the doc of the new cutsceneSession in pkg/ui/cutsceneaudiodev.go, which
// replaces the longer note on the old asynchronous Play, the bounded device
// test helper in pkg/ui/deviceoutput_test.go, the new
// pkg/ui/cutsceneaudiodev_test.go, and this paragraph.
//
// It rises for an attack that ends a saved group's move: the doc of the new
// savedMoveGroup in pkg/sim/savedgroupcommands.go, the new test's doc in
// pkg/sim/structurecombat_test.go, and this paragraph. TestIdentCount falls
// with the rename of the producerBasis fixture.
//
// It rises for headless writers on the engine's routes: the docs of the new
// HeadlessDamage in pkg/sim/headless.go, Mission.PickUpUnderfoot in
// pkg/game/world.go and pickUpAt in cmd/missionrun/main.go, the new take and
// damage helpers in pkg/game/headlessreach_test.go, and this paragraph.
//
// It rises for the saved mover route cell: the docs of the new SavedMover in
// pkg/sim/currentmotion.go, savedMoverMatches in pkg/game/savmotion.go, the
// new pkg/game/savmovercell_test.go, and this paragraph. TestIdentCount falls
// with the rename of the group, stride and crossing fixtures that test uses.
//
// It rises for the doc on new clearRetiredWorldEffectCarriers in
// pkg/sim/savedcontinuation.go and this paragraph.
//
// CommentBytes fell by 92 before this note when the death-clock explanation
// in pkg/game/world.go replaced its older single-fall wording. This note is
// included in the recomputed count.
//
// CommentBytes rises for the fresh-process gob fixture explanation in
// pkg/game/save_test.go and this note.

// CommentBytes rises for the new ActorDeadSourceHealth doc in
// pkg/sim/currentvalues.go and this note; admission wording shrank.
//
// CommentBytes rises for the default-build sentence in Scan's doc in
// internal/gatedtests/scan.go and this note.
//
// CommentBytes rises for loading a hired Catapult or Ballista saved with Units
// row 0: the new comment in SourceActorSeed in pkg/mapload/sourcebinding.go,
// the docs of the new pkg/game/siegehire_release_test.go, and this note.
//
// CommentBytes rises for a raised Ghost's SAV row: the doc of the new ghostRow
// in pkg/mapload/ghost.go, its two uses in pkg/mapload/actorconstructor.go and
// pkg/mapload/sourcebinding.go, the new pkg/game/raisedghost_release_test.go,
// and this note.
//
// CommentBytes rises for the in-place attack action on a superseded motion
// carrier in syncCurrentActorCarriers in pkg/sim/currentactions.go and this
// note.
//
// CommentBytes rises for three release fixtures: the committed selection in
// pkg/game/cityquickspells1167_release_test.go, the mid-crossing order in
// pkg/game/milestone2_current_producers_release_test.go, the new sprayRowWalk
// in pkg/game/spellcontinuation_release_test.go, and this note.
//
// CommentBytes rises for Rood's fall, Heal, SAVE and cold LOAD lifecycle in
// pkg/game/roodthree_rescue_test.go and this note.
//
// CommentBytes rises for the installed skill-raise notice: the docs of the new
// Words.SkillRaised, authoredSkillRaised and SkillRaisedLine in
// pkg/ui/words.go, mainSlotSkillRaised in pkg/game/installtext.go,
// skillRaiseWords and skillRiseRows in pkg/game/world.go, the new test in
// pkg/ui/words_test.go, the new pkg/game/skillraise_test.go and
// pkg/game/skillraise_release_test.go, and this note.
//
// CommentBytes rises for a fallen character Heal can still raise: the doc of
// the new Entity.Restorable in pkg/sim/world.go, the loss-admission comment in
// pkg/game/world.go, the new pkg/game/fallenherodefeat_test.go and
// pkg/game/fallenhero_release_test.go, the finished-body fixture in
// pkg/game/dyingoutcome_test.go, and this note. The value is a fresh measure
// of the tree merged with the two notes above.
//
// CommentBytes rises for the placed-person portrait background: the docs of
// the new FigureIsHero in pkg/data/portrait.go and rosterFigureID in
// pkg/game/figures.go, the new pkg/game/placedherobackground_test.go and
// pkg/game/placedherobackground_release_test.go, and this note.
//
// CommentBytes rises for the selection reply: the docs of the new
// SetSelectionAcknowledgment, noteSelected and replySelection in
// pkg/ui/commandvoice.go, the selectionPicked field in pkg/ui/viewer.go, the
// second results of decide, clickSelection and rectangleSelection in
// pkg/ui/command.go, the reply call in pkg/ui/app.go, replyKind,
// replyRecordings, commandVoiceState and unitReplies in
// pkg/game/commandvoice.go, replyClockFrames and headlessReplyClock in
// pkg/game/commandvoice_witness.go, the new pkg/ui/selectionvoice_test.go,
// pkg/game/selectionvoice_test.go and pkg/game/selectionvoice_release_test.go,
// and this note.
//
// CommentBytes rises for the fallen-unit minimap mark: the docs of the new
// MapEntity.Restorable in pkg/ui/overlay.go, minimapMarks and the new
// minimapMarkRect in pkg/ui/minimap.go, App.HeadlessMinimapMarkAt in
// pkg/ui/headless.go and the Restorable projection in pkg/game/world.go, the
// new test in pkg/ui/minimap_test.go, the new
// pkg/game/minimapfallenmark_hotfix_release_test.go, and this note.
//
// CommentBytes rises for the item information price line: the comment on the
// positive-price gate and the file doc in pkg/game/iteminfo.go, the new test's
// doc in pkg/game/iteminfo_test.go, the new
// pkg/game/itempriceline_release_test.go, and this note.
//
// CommentBytes rises for the Shift drag's whole-stack move: the shift- edge's
// doc in pkg/ui/headlesspointer.go, the held origin in pkg/ui/app.go, the new
// pkg/game/shopshiftdrag_test.go and pkg/game/shopshiftdrag_release_test.go,
// and this note. The ShiftHeld, ShopControl.Shift and ShopDrag comments shrank.
//
// CommentBytes rises for a fallen band actor the mission-end cull raises: the
// NormalizeMissionSurvivors doc in pkg/sim/missionreturn.go, the tally-order
// comment in pkg/game/frontend.go, the new pkg/sim/missionreturnfallen_test.go,
// pkg/game/fallenherowin_test.go and pkg/game/fallenherowin_release_test.go,
// and this note. The value is a fresh measure of the merged tree.
//
// CommentBytes rises for the original's status bars: the docs of the new
// pkg/render/terrain/statusbar.go and pkg/ui/statusbar.go, their new tests
// pkg/render/terrain/statusbar_test.go and pkg/ui/statusbar_test.go, the new
// pkg/game/statusbars_release_test.go, net of the bar code and tests removed
// from pkg/render/terrain/overlay.go and overlay_test.go, and this note.
//
// CommentBytes rises for ranged shots drawn from their installed sprites: the
// new pkg/game/unitshot.go, indexedEffectFrames and sharedProjectileTable in
// pkg/game/projectiles.go, the shots field and the fallback mark in
// pkg/game/world.go, boltDraws in pkg/game/spellbolt.go, the class shot
// fields in pkg/render/terrain/units.go, SpritePath in pkg/data/projectile.go,
// the new pkg/game/unitshot_test.go and pkg/game/unitshot_release_test.go,
// the changed tests in pkg/game/projectiles_test.go, and this note.
//
// CommentBytes rises for the merchant's sale-stock descriptions: the doc of
// the new saleStockSpeech in pkg/game/townresponse.go, the new
// pkg/game/shopmerchantcomment_test.go with closeShopTip, the new
// pkg/game/shopmerchantcomment_release_test.go, and this note.
//
// CommentBytes rises for a scroll click during world-map travel: the
// WorldMapClick doc and the new travellingTo doc in pkg/game/worldmap.go, the
// new test's doc in pkg/game/worldmap_test.go, the new
// pkg/game/worldmaprepeatclick_release_test.go, and this note.
//
// CommentBytes rises for the loaded dialogue speaker: the docs of the new
// placedEntity and placedEntities in pkg/game/speakeractors.go, the new
// pkg/game/speakerload_test.go and pkg/game/placedspeaker_release_test.go,
// and this note.
//
// CommentBytes rises for the town equipment double click: the doc of the new
// shopDollTakes in pkg/game/shopview.go, the new
// pkg/game/shoppackdoubleclick_test.go and
// pkg/game/recoveredcap_release_test.go, the double-click wait in
// pkg/game/shopmerchantcomment_release_test.go, and this note.
//
// CommentBytes rises for the installed pickup line and the left-aligned top
// notices: the docs of the new pickupItemLine, pickupRowsForItems,
// pickupRowsForTake and reachedStacks in pkg/game/pickup_words.go, the new
// slots in pkg/game/installtext.go and fields in pkg/ui/words.go,
// pickupLeftMargin, pickupPen, PickupLine and PickupLog in pkg/ui/pickup.go,
// the new tests in pkg/ui/pickup_test.go, the new pkg/game/pickupline_test.go
// and pkg/game/topnotices_release_test.go, and this note. The takeSackFor doc
// in pkg/game/world.go and the tests replaced in pkg/game/pickup_test.go
// shrank.
//
// CommentBytes rises for a large unit's status bars: the docs of the new
// statusBarOverBox and of StatusBarFill, StatusBarRect and
// AppendStatusBarRuns in pkg/render/terrain/statusbar.go, the new Selection
// field in pkg/render/terrain/units.go, statusBars in pkg/ui/statusbar.go, the
// rewritten pkg/render/terrain/statusbar_test.go with its test of every
// installed wide class, the new pkg/ui/statusbar_wide_test.go and
// pkg/game/largeunitbars_release_test.go, the widened checker in
// pkg/game/statusbars_release_test.go, and this note.
//
// CommentBytes rises for a restored projectile drawn at its saved facing:
// savedProjectileFacing's doc in pkg/game/spellbolt.go, the new
// pkg/game/savedprojectilefacing_test.go and
// pkg/game/savedprojectilefacing_release_test.go, and this note.
//
// CommentBytes rises for shelves listed cheapest first: the new
// pkg/game/shoporder_release_test.go, the changed tests in
// pkg/game/shopsort_test.go and pkg/game/shopbook_release_test.go, net of the
// shorter order docs in pkg/game/shopsort.go and pkg/game/shop.go, and this
// note.
//
// CommentBytes rises for a party body at -10 or below that stays on the field:
// the doc of the new withoutFallenBodies in pkg/game/frontend.go, the new
// pkg/game/fallenbodywin_test.go and pkg/game/fallenbodywin_release_test.go,
// and this note. The citations in pkg/sim/missionreturn.go,
// pkg/sim/missionreturnfallen_test.go and pkg/game/fallenherowin_test.go
// moved off the closed row without growing.
//
// CommentBytes rises for the hurt voice that follows a human's sex and class:
// the docs of the new hurtEvent, VoiceBank, voiceMemory, playVoiceAt,
// placeAndPlay, ClaimVoice and playHurt in pkg/ui/sound.go, the new
// MapEntity.Voice and Blows in pkg/ui/overlay.go, the rewritten
// pkg/game/humansound.go, SoundBank.VoiceSample in pkg/game/sound.go, the
// blows field in pkg/game/world.go, the changed tests in pkg/ui/sound_test.go,
// the new pkg/game/hurtvoice_test.go and pkg/game/hurtvoice_release_test.go,
// and this note. pkg/ui/viewer.go and pkg/game/commandvoice.go fell. The
// value is a fresh measure on the merged tree.
//
// CommentBytes rises for the unit-cast flag of a cast order: the
// ProjectCastOrderOperands doc in pkg/sim/currentorder.go, the new
// pkg/game/currentcastorder_test.go and
// pkg/game/currentcastorder_release_test.go, and this note.
//
// CommentBytes rises for the dialogue picture backdrop: the doc of the new
// DialogFrame.PortraitBack in pkg/ui/menupanel.go, the rewritten portrait docs
// and the new drawNoticeWindow in pkg/ui/notice.go, the loader note in
// pkg/game/menupanel.go, the new pkg/ui/portraitback_test.go and
// pkg/game/dialoguebackdrop_release_test.go, and this note. The value is a
// fresh measure on the merged tree.
//
// CommentBytes rises for the pre-create name field: the docs of the new
// HeroNames, Unnamed, lastPressed, enterPreCreate, defaultName, nameCap and
// encodeName and of SelectPreChoice, Back and EditName in pkg/ui/chargen.go,
// the new pkg/ui/chargen_caret.go, NameFont, the field rect and the prompt
// and name origins and inks in pkg/ui/chargen_page.go, the HeadlessType doc in
// pkg/ui/chargen_headless.go, chargenUnnamed, chargenPictureNameEntries and
// HeroNames in pkg/game/chargenassets.go, the seed note in
// pkg/game/chargen.go, the order and name docs in
// pkg/game/headlesschargen.go, the new pkg/ui/chargen_name_test.go and
// pkg/game/precreatename_release_test.go, the synthetic font in
// pkg/game/chargenassets_test.go, the synthetic names and name font in
// cmd/againrom/main_test.go, the moved hover checks in
// pkg/ui/frontend1184_test.go and pkg/game/frontend1184_release_test.go, and
// this note. The value is a fresh measure on the merged tree.
//
// CommentBytes rises for mission 20's own AutoGetMission: the docs of
// campaignProgressFromSAV and the announce latch in
// pkg/game/campaignprogress.go, Won's latch and the Available doc in
// pkg/game/town.go, the Announced sentence in pkg/game/nativecity.go net of
// its two dropped DIV-881 citations, the new
// pkg/game/autogetmission_test.go and pkg/game/autogetmission_release_test.go,
// the frozen fixture's AutoGetMission in pkg/game/legacyprojection_test.go,
// and this note. The value is a fresh measure on the merged tree.
//
// CommentBytes rises for character generation's chrgen sounds: the new
// pkg/ui/chargensound.go, the docs of namedSounds, stepChargenStat,
// chargenDoubleClick, preCreateForward, preCreateBack and the press notes in
// pkg/ui/app.go and pkg/ui/ending.go, TownSurfacePresser in pkg/ui/town.go,
// NamedSample in pkg/game/sound.go, TownSurfacePress in
// pkg/game/townshell.go, the schoolSounds field in pkg/game/townscreen.go,
// the new pkg/ui/chargensound_test.go, pkg/ui/chargensoundsites_test.go,
// pkg/game/schoolskillsound_test.go and
// pkg/game/chargensound_release_test.go, the restated press tests in
// pkg/ui, and this note. The value is a fresh measure on the merged tree.
//
// CommentBytes rises for equal items sharing one pack cell: the docs of the
// new FoldItems in pkg/sim/carry.go, JoinForm in pkg/sim/item.go,
// firstMergeableHeld, joinForm and mergeIntoHeld in pkg/sim/actorload.go,
// absorbValue in pkg/sim/savedobjects.go, and cityPackRootsCurrent,
// cityJoinForm, cityPackJoins and cityShelfJoins in
// pkg/game/cityshopmutation.go; the new pkg/sim/packmerge_test.go and
// pkg/game/packstack_release_test.go; the new tests' docs in
// pkg/game/cityshopmutation_test.go, pkg/game/nativecityitems_test.go and
// pkg/mapload/currentcarry_test.go; and this note.
//
// CommentBytes rises for Weapon bytes 22 and 23 leaving the item comparison:
// the doc of the new sameRetainedOperands in pkg/sim/item.go; the new tests'
// docs in pkg/sim/iteminstance_test.go, pkg/sim/packmerge_test.go and
// pkg/game/cityshopmutation_test.go; the new
// pkg/game/packstack_owner_release_test.go; and this note. The value is a
// fresh measure on the merged tree.
//
// CommentBytes rises for the game window's icon: the package doc and the
// decoder docs of the new pkg/formats/winicon (doc.go, winicon.go and
// winicon_test.go), WindowIcon and deepestPerSize in the new
// pkg/game/windowicon.go and pkg/game/windowicon_test.go, SetWindowIcon in
// pkg/ui/app.go, the desktop seam and installWindowIcon in cmd/againrom/main.go
// and the new cmd/againrom/windowicon_test.go, the new synthetic builders
// internal/synth/pe.go and internal/synth/icon.go, the two registrations in
// internal/archtest/dag.go, and this note. The value is a fresh measure on the
// merged tree.
//
// CommentBytes rises for the pickup text and the message line: the docs of the
// new pkg/ui/messageline.go and pkg/ui/messageline_test.go, Font.DrawFlat in
// pkg/render/text/text.go and its test in pkg/render/text/draw_test.go, the
// rewritten pkg/game/pickup_words.go, the two gold words in pkg/ui/words.go and
// pkg/game/installtext.go, the rewritten pkg/game/pickupline_test.go and
// pkg/game/pickup_gold_test.go, and pkg/game/messageline_release_test.go, which
// replaces topnotices_release_test.go, and this note. pkg/ui/pickup.go and
// pkg/ui/pickup_test.go fell. The value is a fresh measure on the merged tree.
//
// CommentBytes rises for the order of the action supplement's bindings: the
// visiting note in pkg/game/savactioncurrent.go, the new
// pkg/game/savbindingorder_test.go and pkg/game/savbindingorder_release_test.go,
// and this note.
//
// CommentBytes rises for the pre-create page's empty-name gate: the doc of
// preCreateForward in pkg/ui/app.go, chargenNamedSetup in
// pkg/ui/chargen_app_test.go, the new pkg/ui/chargen_emptyname_test.go and
// pkg/game/precreateemptyname_release_test.go, and this note. The value is a
// fresh measure on the merged tree.
//
// CommentBytes rises for the default hero's installed name: the docs of
// defaultHeroName and MissionPartyAs in pkg/game/hero.go, heroPictureNames in
// pkg/game/chargenassets.go, LoadDefinitions' note in pkg/game/table.go,
// Table.HeroNames in pkg/mapload/spawn.go, the new pkg/game/heronames_test.go
// and pkg/game/heronames_release_test.go, and this note. The value is a fresh
// measure on the merged tree.
//
// CommentBytes rises for a ranged withdrawal that lets its loaded cycle
// finish: the comments on the changed tail in pkg/sim/withdraw.go, the held
// move skip and the cycle-end drop in pkg/sim/step.go, the flyer clause in
// pkg/sim/route.go, the new pkg/sim/withdraw_cycle_test.go and
// pkg/game/withdrawcycle_release_test.go, and this note. The value is the
// measure of the tree without this note plus this note's own bytes.
//
// CommentBytes rises for the world-map travel inputs: the docs of
// WorldMapMove and WorldMapClick in pkg/game/worldmap.go, the wheel note in
// pkg/ui/headlesspointer.go, the new pkg/game/worldmaptravelinputs_test.go and
// pkg/game/worldmaptravelinputs_release_test.go, and this note.
//
// CommentBytes rises for the class-voiced fall and the one voice bank: the
// restated doc of playHurt and stepSound in pkg/ui/sound.go, voiceBank in
// pkg/game/humansound.go, unitReplies in pkg/game/commandvoice.go, the new
// tests and their docs in pkg/ui/sound_test.go, pkg/game/hurtvoice_test.go
// (hurtVoiceFightPlays and the class-voiced fight test),
// pkg/game/replybank_test.go and pkg/game/labyrinth_release_test.go, the
// hearing test in pkg/game/hurtvoice_release_test.go, and this note.
// TestIdentCount falls by one with the retired family test in
// pkg/game/acknowledgments_test.go.
//
// CommentBytes rises for the Labyrinth fall witness in
// pkg/game/labyrinth_release_test.go, whose doc now states that it stops once
// each creature class has fallen in hearing and targets the classes not yet
// heard, and this note. The value is a fresh measure on the merged tree.
//
// CommentBytes rises for the statistics pane taking a released pack item: the
// doc of the new heldItemBox in pkg/ui/missionpane.go, the amended doc of
// HeadlessDollBoxPoint in pkg/ui/headlesspointer.go, the new tests' docs in
// pkg/ui/missionpane_drop_test.go, the new pkg/game/panedrop_release_test.go,
// and this note.
//
// CommentBytes rises for the tavern speaker cast: the doc of the new
// tavernSpeakerCast in pkg/game/taverntalk.go, the rewritten cast and resolver
// notes in pkg/game/speakeractors.go and pkg/game/townscreen.go, the new
// pkg/game/tavernspeaker_test.go and pkg/game/tavernspeaker_release_test.go,
// and this note. The value is a fresh measure on the merged tree.
//
// CommentBytes rises for an attacked enemy group fighting as a group: the docs
// of the new pkg/sim/groupengaged.go and pkg/sim/groupengaged_test.go, the new
// pkg/game/groupfight_release_test.go, the closedOn paragraph in
// pkg/sim/combat.go, two short notes in pkg/sim/engage.go, and this note. The
// value is a fresh measure on the branch tree.
//
// CommentBytes rises for the hired squad's place in the town pickers: the docs
// of the new pkg/game/townpicker.go, the new pkg/game/townpicker_test.go and
// pkg/game/townpicker_release_test.go, the rewritten docs of shopMemberIndex
// and shopStepMember in pkg/game/shopview.go, the two retargeted witnesses in
// pkg/game/nativecity_release_test.go, and this note. The value is a fresh
// measure on this branch at its base.
//
// CommentBytes rises for the item information stating no price: the docs of
// priceLabels, statesPrice, unpricedInfo, shopNumberDrawn and the two release
// tests in pkg/game/itempriceline_release_test.go, the new
// TestItemInstanceInfoLinesComposeNoPriceLine in pkg/game/iteminfo_test.go and
// the PriceFont doc in pkg/ui/shopscreen.go, net of the price comments dropped
// from pkg/game/iteminfo.go, and this note. TestIdentCount and TestFileCount
// stay where they were. The value is a fresh measure on this branch.
//
// CommentBytes rises for a player's attack order keeping its victim: the docs
// of the new holdsOrderedVictim and stanceMembers and the note in decide in
// pkg/sim/engage.go, the note in savedDecision in pkg/sim/savedgroupsai.go, the
// new pkg/sim/orderedvictim_hotfix_test.go and
// pkg/game/orderedvictim_release_test.go, the docs of the waiting subject in
// pkg/game/hurtvoice_release_test.go and of the answered fire in
// pkg/game/unitshot_release_test.go, and this note. The value is a fresh
// measure on the merged tree.
//
// CommentBytes rises for the shield worn alone and over a weapon that fills
// both hands: the restated docs of WeaponAllowsShield and
// NormalizeShieldLoadout in pkg/mapload/shield.go, the wear notes in
// pkg/game/shoproom.go and pkg/game/world.go, the new
// pkg/sim/shieldalone_test.go, pkg/game/shieldwear_test.go and
// pkg/game/shieldwear_release_test.go, the rewritten docs in
// pkg/mapload/wear_test.go and pkg/mapload/spawn.go, and this note. The docs
// in pkg/sim/equip.go, drop.go, step.go and sourceequipmove.go and in
// pkg/mapload/fromalm.go fall.
//
// CommentBytes rises for the hero name witness's shield case: the note on the
// dropped name index in pkg/game/heronameindex_hotfix_release_test.go, and
// this note. The value is a fresh measure on the merged tree.
//
// CommentBytes rises for the room exits and conversation ends that post no
// line: the shop Exit comment in pkg/game/shopview.go, the doc of talk in
// pkg/game/townscreen.go, the new pkg/game/roomexitlines_test.go and
// pkg/game/roomexitlines_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the tavern speaker who stays after the conversation:
// the docs of Town.TavernRoster and the TownOffer note in pkg/game/town.go,
// keepTavernSelection and the amended tavernCandidates doc in
// pkg/game/townshell.go, the kept speaker clause in pkg/game/townscreen.go,
// the new pkg/game/tavernspeakerroster_test.go and
// pkg/game/tavernspeakerroster_release_test.go (renamed at the merge beside
// the tavern speaker cast's files of the same first names), and this note.
//
// CommentBytes rises for the music request at a completed load: the doc of
// RequestScene in pkg/ui/music.go, the counter note in pkg/ui/load_window.go,
// the sync note in App.syncMusic and the musicLoads field note in
// pkg/ui/app.go, the new pkg/ui/music_load_test.go and
// pkg/game/musicload_release_test.go, and this note.
//
// CommentBytes rises for the front-end cursor standing on the frame's pixel
// lattice: the docs of the new CursorPlacement and cursorPlacement and the
// amended drawCursor doc in pkg/ui/app.go, the amended cursorPresent doc in
// pkg/ui/flow.go, the new HeadlessCursor doc in pkg/ui/headless.go, the new
// pkg/ui/menucursor_test.go and pkg/game/menucursor_release_test.go, and this
// note.
//
// CommentBytes rises for the selection lines at no selection and on the lower
// card: the zero case of characterPaneView and the restated doc of
// missionCardPresent in pkg/ui/missionpane.go, the amended doc of panelPresent
// in pkg/ui/panel.go, the doc of drawSelectionStatus in pkg/ui/townshell.go,
// the docs of the changed and the new tests in pkg/ui/missionpane_test.go, the
// new pkg/game/selectionstatus_release_test.go, the reference notes in
// pkg/game/placedherobackground_release_test.go, and this note. The value is
// the measure of this branch with this note in place. At the merge it rises
// for the doc of noticeCalls in pkg/ui/notice_capture_test.go: the notice
// tests count the notice's own calls, because the zero-selection lines share
// the capture window.
//
// CommentBytes rises for the placed speaker's class: the class clause in
// missionSpeakers (pkg/game/speakeractors.go), the new
// pkg/game/placedspeakerclass_test.go and
// pkg/game/placedspeakerclass_release_test.go, and this note. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the mage figure's own paint order: the amended doc of
// FigureDrawOrder and the new doc of mageFigureDrawOrder in
// pkg/data/itemcode.go, the amended doc of composeUnitFigure in
// pkg/game/figures.go, the new test docs in pkg/data/itemcode_test.go, the new
// pkg/game/magefigure_test.go and pkg/game/magefigure_release_test.go, and this
// note. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the original's non-mage colour pass: the docs of
// FigureStep, mageFigureSteps and FigureDrawSteps in pkg/data/itemcode.go, the
// new docs in pkg/data/itemcode_test.go, pkg/game/figures_test.go and
// pkg/game/fighterfigure_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the world-map arrival that waits for the Cross
// animation: the docs of worldMapState.cross, crossPassedEnd and finishCross
// and the amended docs of WorldMapClick, WorldMapTick and arriveWorldMap in
// pkg/game/worldmap.go, the new pkg/game/worldmapcross_test.go and
// pkg/game/worldmapcross_release_test.go, and this note. The value is the
// measure of this branch with this note in place.
//
// CommentBytes rises for the attack move walking on to its commanded cell: the
// doc of the new resumeSwarm in pkg/sim/engage.go, the new
// pkg/sim/swarmresume_test.go and pkg/game/attackmove_release_test.go, the doc
// paragraph on the changed witness in pkg/game/quest81_release_test.go, and
// this note. The value is a fresh measure on the merged tree.
//
// CommentBytes rises for a player's attack order beside a book cast and a
// cast at a unit by a unit fighting an ordered victim: the docs of the new
// attachAttack in pkg/sim/engage.go and scrollCastPending in
// pkg/sim/actionguard.go, the attack arm's note in pkg/sim/step.go, the
// ordered-victim clause in autoCastTarget in pkg/sim/spell.go, the resume note
// in pkg/sim/manualcast.go, the new pkg/sim/mageorders_hotfix_test.go and
// pkg/game/mageorders_release_test.go, the three pins reworded in
// pkg/sim/orderguard1001_test.go, pkg/sim/savedengagement_test.go and
// pkg/sim/spell_test.go, and this note.
//
// CommentBytes rises for the synthesised Hero speaker's backdrop: the clause in
// SpeakerFace (pkg/game/speakers.go), the new pkg/game/speakerbackdrop_test.go
// and pkg/game/speakerbackdrop_release_test.go, and this note. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the shop grid's price figure at its claimed place: the
// new price constants and the amended docs of drawShopPlaque and
// ShopPricePlacement in pkg/ui/shopscreen.go, the new pkg/ui/shopprice_test.go
// and pkg/game/shopprice_release_test.go, the amended docs of
// pkg/game/shopplaquefit_release_test.go and
// pkg/game/itempriceline_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the shop pane taking an item released on any of its
// rectangles: the doc of the new shopPaneTakesItemAt in pkg/ui/shopscreen.go,
// the new pkg/ui/shoppanedrop_test.go and pkg/game/shoppanedrop_release_test.go,
// and this note. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the placed Lancers' information portrait: the amended
// doc of rosterFigureID in pkg/game/figures.go, the new
// pkg/game/placedportraithorse_test.go and
// pkg/game/placedportraithorse_release_test.go, and this note. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the automatic withdrawal whose flee cell no route
// serves: the docs of the new answerRefusedFlee and fleeRefused and the two
// clauses on withdrawFrom and withdrawFromAny in pkg/sim/withdraw.go, the new
// pkg/sim/withdraw_refused_test.go and pkg/game/withdrawflee_release_test.go,
// and this note. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for a loaded actor's own character sheet: the amended doc
// of placedCharacters in pkg/game/panelchars.go, the new
// pkg/game/creaturesheetload_test.go and
// pkg/game/loadedsheets_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the shop grid's plaque and figure on an item priced 0
// or below: the new doc of shopGridPrice in pkg/game/shopview.go, the two
// paragraphs added to the docs of drawShopCell and shopPlaqueIndex in
// pkg/ui/shopscreen.go, the new pkg/ui/shopunpriced_test.go,
// pkg/game/shopgridprice_test.go and pkg/game/shopunpriced_release_test.go, the
// amended docs in pkg/ui/shopscreen_test.go, pkg/game/shopprice_release_test.go
// and pkg/game/shopplaquefit_release_test.go, the note added to
// pkg/ui/grouping_test.go, and this note. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the town guards' rest frame on entry to the square:
// the doc of the new townGuardRestFrame in pkg/game/townexterior.go, the new
// pkg/game/townguards_test.go, and this note. The value is a fresh measure on
// the tree with this note.
//
// CommentBytes rises for the release witness of the room figures and the
// guards after a room exit: the docs of the new
// pkg/game/townfigures_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the shop grid choosing its plaque from the stored
// price on both sides of the deal: the new doc of shopClientPrice and the
// amended docs of shopGridPrice and shopPlaceCell in pkg/game/shopview.go, the
// new PlaquePrice doc and the amended docs of ShopCell, shopPlaqueIndex and
// drawShopPlaque in pkg/ui/shopscreen.go, the new pkg/ui/shopplaque_test.go,
// pkg/game/shopplaquesize_test.go and pkg/game/shopplaquesize_release_test.go,
// the amended docs in pkg/ui/shopunpriced_test.go, pkg/game/shopgridprice_test.go
// and pkg/game/shopplaquefit_release_test.go, and this note. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for an explicit Retreat whose flee cell no route serves:
// the amended docs of armRetreat in pkg/sim/playerretreat.go, of
// answerRefusedFlee in pkg/sim/withdraw.go and of the Retreat boundary clause
// in pkg/sim/step.go, the new pkg/sim/retreat_refused_test.go and
// pkg/game/retreatrefused_release_test.go, and this note. The value is a fresh
//
// CommentBytes rises for the gate press with nothing on offer: the docs of the
// new dialogueRoom and townGateTextPath in pkg/game/townscreen.go, of TownGate
// in pkg/game/town.go, the new pkg/game/gatenooffer_test.go and
// pkg/game/gatenooffer_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for a loaded attack cycle kept through a move, a group
// move, a guard's walk home and an escort's close: the doc of the new
// clearAttackBetweenCycles in pkg/sim/combat.go, the clauses on the move arm in
// pkg/sim/step.go, on issueSavedGroupDestination in pkg/sim/group.go, on
// guardWalkHome in pkg/sim/guardarm.go and on escortClose and escortStepAway in
// pkg/sim/escort.go, the new pkg/sim/loaded_cycle_test.go and
// pkg/game/loadedcycle_release_test.go, the amended docs in
// pkg/sim/orderguard1001_test.go and pkg/sim/structurecombat_test.go, the
// clause on the move closure in pkg/game/quest81_release_test.go and the new
// doc of finishLoadedCycle in pkg/game/headlessreach_test.go, and this note.
// The clause in pkg/sim/withdraw.go moved into the new doc. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for a placed person's figure following the face byte his
// spawner writes: the new docs of PlacedFigure, placedFigure and
// spawnerFaceByte in pkg/mapload/placedfigure.go, the new doc of
// FigureForFaceByte and the rewritten arm block in pkg/data/portrait.go, the
// new pkg/mapload/placedfigure_test.go, pkg/game/placedfigurearm_test.go and
// pkg/game/placedfacebyte_release_test.go, and this note. The value is a fresh
//
// CommentBytes rises for the far search spending the budget of its mover's
// owner: the docs of budgetRule, humanParticipantUnit, farBudgetFor and
// generationBudget in pkg/sim/route.go, the rewritten clause on the far search
// in pkg/sim/step.go, the new pkg/sim/farbudget_test.go and
// pkg/game/farsearch_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the Labyrinth fall witness answering the creatures
// that strike its mace-bearers: the docs of the new labyrinthBeside and
// labyrinthStriker and the amended doc of
// TestReleaseClassVoicedFallsPlaySoundFourInTheLabyrinth in
// pkg/game/labyrinth_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the reacquisition after a refused flee: the docs of
// the new reacquireWithinReach, reacquisitionVictim, suppressedAcquirer and
// retreatKeepsPursuit in pkg/sim/reacquire.go, the amended doc of
// answerRefusedFlee and the new docs of fleeRefusedAt and fleeCell in
// pkg/sim/withdraw.go, the amended Retreat boundary clause in pkg/sim/step.go,
// the new pkg/sim/reacquire_test.go and pkg/game/reacquisition_release_test.go,
// and this note. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for a loaded attack cycle kept through the Defend and
// Patrol orders, the Swarm walk-on, the stop of a script or saved group and
// structure use: the clause on commandDefend in pkg/sim/playerdefend.go, the
// docs of beginStructureUse and the new usingStructure in
// pkg/sim/structureuse.go, the clauses on the movement and attack passes in
// pkg/sim/step.go, the amended doc of clearAttackBetweenCycles in
// pkg/sim/combat.go, the new pkg/sim/cycle_writers_test.go and
// pkg/game/cyclewriters_release_test.go, the docs of the new
// openLoadedCycleArenaBeside, clickGround and openGroundSquareBeside in
// pkg/game/loadedcycle_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for Hold Position's stand-down: the docs of the new
// standDown, commandStandGround and syncSavedStandGround in
// pkg/sim/standground.go, the paragraph on the participant's stand-down in
// decide's doc in pkg/sim/engage.go, the comments in pkg/sim/savedgroupsai.go
// and pkg/sim/step.go, the new pkg/sim/standground_test.go and
// pkg/game/holdposition_release_test.go, the note on the renamed
// TestAStandGroundMemberDropsAVictimThatStepsPastReach in
// pkg/sim/release_test.go, and this note. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the dialogue window's single command and the inn's
// queue: the four new test files pkg/game/dialoguekeys_test.go,
// pkg/game/innqueue_test.go, pkg/game/dialoguekeys_release_test.go and
// pkg/ui/menucapture_test.go, the amended docs of the seven tests that encoded
// the old accept and decline rows (pkg/game/roomexitlines_test.go,
// roomexitlines_release_test.go, tavernspeakerroster_test.go,
// tavernspeakerroster_release_test.go, gatenooffer_release_test.go,
// townfigures_release_test.go and townreturn1168_release_test.go), and this
// note. The docs in pkg/game/townscreen.go, townshell.go, town_test.go and
// pkg/ui/app.go fell. TestIdentCount falls by one: townReturnTalk1168 became
// townReturnHear1168 and the new townReturnLeaveTavern carries no number. The
// value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the town controls witness's way out of the tavern:
// the one comment in pkg/game/owner_fidelity_witness.go and this note. The
// value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the selection lines standing centred in the body of
// both mission boxes: the amended doc of drawSelectionStatus and the new docs
// of selectionLinePitch and centredOffset in pkg/ui/townshell.go, the amended
// docs in pkg/ui/missionpane_test.go, the new pkg/ui/selectionstatus_test.go,
// the amended docs in pkg/game/selectionstatus_release_test.go, and this
// note. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the dialogue panel drawn and its text laid out as the
// original does: the docs of the new pkg/ui/noticedialogue.go (the nine-piece
// panel, the portrait surface, the button, the wrap and the justified line),
// the new pkg/ui/noticedialogue_test.go and
// pkg/game/dialoguelayout_release_test.go, the amended geometry docs in
// pkg/ui/notice.go and pkg/ui/noticepage.go, the amended witness doc in
// pkg/game/dialoguebackdrop_release_test.go, small clauses in
// pkg/ui/halt_test.go, pkg/ui/notice_test.go, pkg/ui/noticepage_test.go and
// pkg/ui/portraitback_test.go, and this note. The value is a fresh measure on
// the tree with this note.
//
// CommentBytes rises for the world-map Enter key hastening travel as a click
// that misses every scroll does: the docs of the new skipTravel and the
// rewritten WorldMapChoose in pkg/game/worldmap.go, the new
// pkg/game/worldmapenter_test.go and pkg/game/worldmapenter_release_test.go,
// the docs of openCrossWitness and clickScroll, split out of startCrossWitness
// in pkg/game/worldmapcross_release_test.go, the clauses on the departure in
// pkg/game/victoryworldmap1088_test.go and
// pkg/game/mission_handoff_mappoint_release_test.go, the docs of the new
// headlessWorldMapArrivalFrames and the rewritten headlessActivateWorldMap in
// pkg/ui/headless.go, the amended fake in pkg/ui/headless_worldmap_test.go,
// and this note. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the numpad's Enter key reading as Enter: the new docs
// of enterKeys, enterPressed and onlyKey in pkg/ui/app.go, the amended doc of
// HeadlessKey in pkg/ui/headless.go, the four new test files
// pkg/ui/numpadenter_binding_test.go, pkg/ui/numpadenter_test.go,
// pkg/game/numpadenter_test.go and pkg/game/numpadenter_release_test.go, and
// this note. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the shelf groups run cheapest first: the amended
// witness doc and the new docs of shopOrderShelfNames, shopOrderCompare,
// shopOrderShow and shopOrderReturn in pkg/game/shoporder_release_test.go, the
// amended docs in pkg/game/shopsort_test.go and
// pkg/game/shopbook_release_test.go, the amended order docs in
// pkg/game/shopsort.go and pkg/game/shop.go, and this note. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the far search of a creature larger than one cell
// spending the n x n arm of the budget: the docs of budgetRule, farBudgetFor and
// generationBudget in pkg/sim/route.go, the amended clause on the far search in
// pkg/sim/step.go, the new pkg/sim/farbudgetlarge_test.go, the new witness in
// pkg/game/farsearch_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the party's hired Catapult and Ballista spending the
// n x n arm of the far search's budget: the amended docs of flatGenerations,
// budgetRule, humanParticipantUnit, farBudgetFor and generationBudget in
// pkg/sim/route.go, the amended clauses in pkg/sim/step.go and pkg/sim/world.go,
// the amended pkg/sim/farbudgetlarge_test.go, the amended docs in
// pkg/sim/farbudget_test.go and pkg/game/farsearch_release_test.go, the new
// pkg/game/hiredfarsearch_release_test.go, and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for glyph capture and visibility settlement in
// pkg/render/text/{capture,text}.go, pkg/render/textsmooth/textsmooth.go,
// pkg/ui/{pixellog,textsmoothing,messageline,numeral}.go, and their new tests
// pkg/render/text/{draw,tint,underlay}_test.go,
// pkg/render/textsmooth/explain_test.go and
// pkg/ui/{textsmoothing_routes,grouping,notice_capture}_test.go. The new release witnesses are
// pkg/game/{precreatetextsmoothing,textsmoothing,worldmaptextsmoothing}_release_test.go.
// The value includes this note and is a fresh measure on this tree.
// CommentBytes rises for the Hold/Retreat witnesses in
// pkg/sim/holdretreat_test.go and pkg/game/holdretreat_release_test.go.
// TestIdentCount falls when the shared saved tactical registry fixture drops
// its numeric suffix; the fixture body and its callers keep the same behaviour.
// The value is a fresh measure after both hotfix merges and includes this note.
//
// CommentBytes rises for the world-map Enter progress helper in
// pkg/game/worldmap.go. The value is a fresh measure on this tree and
// includes this note.
//
// CommentBytes rises for the cornered seed-route shortcut and the town
// refusal filters with their witnesses in pkg/sim and pkg/game. The value is a
// fresh measure after both story merges and includes this note.
//
// CommentBytes rises for the speed modifier projection in
// pkg/game/savactorproject.go and its witness pkg/game/hastesave_hotfix_release_test.go.
// The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the modifier words of the other timed stat effects in
// pkg/game/savactorproject.go and its witnesses.
//
// CommentBytes rises for the pending-row replacement doc in pkg/sim/standground.go
// and the Hold witnesses pkg/sim/holdstrike_test.go and
// pkg/game/holdstrike_release_test.go. The value is a fresh measure on the tree
// with this note.
//
// CommentBytes rises for the live city group membership in
// pkg/game/citygroupcarry.go and its witnesses pkg/game/citygroupslive_test.go
// and pkg/game/citygroupscycles_release_test.go, net of the unused loaded-town
// member writer removed from pkg/game/originalcity_current.go. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the current-health admission in
// pkg/game/currenthealthadmission.go, the width-restoration docs in
// pkg/sim/currentvalues.go and pkg/sim/originalpools.go, and the wide-domain
// witnesses pkg/game/currenthealthadmission_test.go,
// pkg/game/widepools_release_test.go and pkg/sim/savedgroups_shape_test.go.
// The value is a fresh measure on the tree with this note.
//
// CommentBytes fell with the retired AGS codec and then rose by about 1000 bytes
// for new code: the save-list name helpers in pkg/ui/save_dialog.go, the
// converter's label and comparator docs, and the test-only historical-name copy
// in pkg/game/agshistorical_test.go. The value is a fresh measure on the tree
// with this note.
//
// CommentBytes rises for the placed rider's own mounted art: the doc of
// rosterBodyArt in pkg/game/world.go and the two comments of
// pkg/game/placedriderart_release_test.go. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the shop's status echoes: the doc of ui.TownAction.Info and
// ui.TownInfo. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the town room under the game menu: the doc of
// townRoomMessage. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the shop pack strip: the doc of packScrollMax and the
// ShopScreenView arrow flags. The value is a fresh measure on the tree with this
// note.
//
// CommentBytes rises for the plain Beard exclusion: the doc of beardItemCode. The
// value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the walkable-area minimap: the docs of minimapWindow,
// minimapWindowed and the Origin field, and the margin witnesses in
// pkg/ui/minimapinput_test.go. The value is a fresh measure on the tree with this
// note.
//
// CommentBytes rises for a saved Human with no definition row: the docs of
// data.HumanDefaults and mapload.sourceHumanDef, and the comments of the two
// new tests. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the city hire purse witness: the doc of
// pkg/game/cityhiregold_release_test.go. The value is a fresh measure on the tree
// with this note.
//
// CommentBytes rises for the city hire cost word: the doc of cityHireCost and the
// witness in pkg/game/cityhiregold_release_test.go. The value is a fresh measure
// on the tree with this note.
//
// CommentBytes rises for the projectile kit: the three comments in
// pkg/game/kitprojectile_release_test.go and this note. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the any-unit spellbook correction: the docs of
// Viewer.SetSpellbook and Viewer.ClearSpellbook in pkg/ui/spellbook.go and the
// comments of pkg/game/spellbookanyunit_release_test.go and the entity-zero
// witness in pkg/ui/spellbook_test.go. The value is a fresh measure on the tree
// with this note.
//
// CommentBytes rises for the TestKitProjectileBuild doc, which now names its
// temporary-directory fallback. The value is a fresh measure on the tree with
// this note.
//
// CommentBytes rises for the hired siege Unit actor: the docs of
// pkg/mapload/siegehire.go, pkg/game/nativecitysiege.go, the siege hire
// builder in pkg/game/tavern.go and the witnesses in
// pkg/game/siegeunit_release_test.go. The value is a fresh measure on the tree
// with this note.
//
// CommentBytes rises for the mission 120 resave witness, which now names why
// its party holds 14. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the strike distance measured from body centres: the doc
// of strikeDistance in pkg/sim/combat.go and the comments of
// pkg/sim/bodyreach_hotfix_test.go and pkg/game/ogrereach_release_test.go. The
// value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the staffless healer left out of a group attack order:
// the docs of stafflessHealer and attackedTogether in pkg/sim/autohealing.go and
// the comments of pkg/sim/healerattackorder_hotfix_test.go and
// pkg/game/healerattackorder_release_test.go. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the creature spell slots: the docs in
// pkg/sim/creaturespell.go, pkg/sim/creaturespellbinary.go,
// pkg/game/savcreaturespells.go and the new tests in
// pkg/sim/creaturespell_test.go and pkg/game/creaturespells_release_test.go, with the
// order slot window docs in pkg/sim/creaturespell.go. The value is a fresh measure
// on the tree with this note.
//
// CommentBytes rises for the carried order fix: the comments of
// projectSavedActorActions and SnapshotSAVDocument.ActionTick, and the docs of
// pkg/game/savactorrestoredorder_test.go and
// pkg/game/kitprojectilev2_release_test.go. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the ranged weapon's range line: the doc of
// weaponRangeLine in pkg/game/iteminfo.go and the comments of
// pkg/game/weaponrangeline_release_test.go and the two range tests in
// pkg/game/iteminfo_test.go. The value is a fresh measure on the tree with this
// note.
//
// CommentBytes rises for the thrown item's visible landing: the docs of
// World.DropLanding, dropWindowCell and dropRing in pkg/sim/drop.go, the
// enqueueDrop and dropAim notes in pkg/game/world.go, the binding-order note in
// pkg/game/currentworldspatial.go and the witness in
// pkg/game/thrownitem_hotfix_release_test.go. The value is a fresh measure on
// the tree with this note.
//
// CommentBytes rises for the hero fallen body and the target bearing: the docs
// in pkg/game/heroart.go and pkg/sim/facing.go and the new tests in
// pkg/game/fallenherobody_release_test.go and pkg/sim/facingbearing_test.go. The
// value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the Heal cast at a hostile unit: the apply-arm note in
// pkg/sim/spell.go and the comments of pkg/sim/hostileheal_hotfix_test.go and pkg/game/hostileheal_release_test.go, net
// of the two admission notes removed from pkg/sim/spell.go. The value is a fresh
//
// CommentBytes rises for the text batch hotfix: the docs of structureCardName in
// pkg/ui/inspection.go, characterCardBodyHeight in pkg/ui/character_card_style.go,
// the new Words fields and worldMapTextDrop, and the comments of the witnesses in
// pkg/ui/panel_test.go and pkg/game/eventaudience_test.go. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the launcher and its supporting packages, all genuinely
// new code: the doc comments of cmd/starter, pkg/ini, pkg/mod, pkg/game/installinfo.go,
// pkg/ui/menulabel.go, internal/buildinfo, internal/versionres and the tests of
// each, plus the -mods and -version notes in cmd/againrom. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the native projectile registry: the docs of the new
// pkg/sim/unitshot.go (ReleaseUnitShot and its helpers), the rewritten header
// and the new release, direction, offset and trail functions in
// pkg/game/unitshot.go, savedProjectileTrail in pkg/game/spellbolt.go,
// bindProjectileDrivers and projectProjectileCounter in
// pkg/game/savworldeffects_project.go, pinNativeProjectileTargets in
// pkg/game/savruntimeids.go, onlyNativeProjectiles in
// pkg/game/savworldeffects.go, and the new tests in pkg/sim/unitshot_test.go
// and pkg/game/freshshot_release_test.go. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes falls for the selection caption docs in pkg/ui. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the stale duplicate map unit rule: the note in
// planOriginalActors, the one-of-several-claimants test and the doc of the release witness in
// pkg/game/staleduplicatemapunit_release_test.go, and the hired party
// convention note in milestone2_acceptance_units_test.go. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// TestIdentCount rises by two and CommentBytes by the rest for the knowledge pin
// moving to snapshot 112, whose test files carry two identifiers with a story
// number, and for the projectile resave witness and the unbound-consumer note
// in pkg/game/savworldeffects.go. The values are a fresh measure on the tree
// with this note.
//
// CommentBytes rises for the creature magic hotfix: the doc of the new
// mission 51 witness and its helpers in pkg/game/creaturemagic_release_test.go,
// the experience value note in pkg/game/savactorproject.go, the reworded
// importedCreatureSpells and OriginalActorSpellbook docs, and the new test in
// pkg/sim/originalspellbooks_test.go, plus the creature caption rule's note in
// pkg/ui/panel.go and its test in pkg/ui/panel_test.go and the witness helpers.
// The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the unit shadow level hotfix: the level note in
// pkg/ui/shadow.go, the UnitShadow docs, and the comments of the new
// pkg/game/unitshadow_release_test.go. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the first-tick skill raise notice: the doc of
// seedSkillBaseline in pkg/game/world.go and the comments of the new tests in
// pkg/game/skillraise_test.go and pkg/game/skillraise_firsttick_release_test.go.
// The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the mod interpreter: the docs of the new pkg/rules,
// pkg/mod, pkg/modrt and pkg/game/modmark.go, the Rules plumbing in pkg/sim,
// pkg/data and pkg/mapload, the mod-set flags in cmd/againrom, the per-mod
// settings panel in cmd/starter, the AgainromMods leaf in
// pkg/formats/sav/native_mods.go, their tests and this note. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the refused pursuit hotfix: the PursuitIdle field note
// in pkg/sim/world.go, the idleRefusedPursuit doc in pkg/sim/combat.go and the
// comments of pkg/sim/refusedpursuit_test.go. TestIdentCount falls by two for
// the escort grid helper's rename. The values are a fresh measure on the tree
// with this note.
//
// CommentBytes rises for the monster spell hover witness: the comments of the new
// pkg/game/monsterspellhover_release_test.go and this note. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the Prismatic Spray turn term and selector preamble:
// the docs in pkg/sim/prismatic.go, pkg/sim/prismaticturn_test.go, the release
// witness note and this note. The value is a fresh measure on the tree with
// this note.
//
// CommentBytes rises for mod items: the docs of the new pkg/mod/items.go,
// pkg/modrt/data.go, pkg/data/rowoverlay.go, pkg/game/moditems.go and
// moditemsprite.go, the mark's item notes, the release witness and this note.
// The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for area-layer movement cost: the docs of the new
// pkg/sim/areacost.go, areacostbinary.go and areacost_test.go, the savedCellCost
// and RecomputeLayeredCosts docs in savedcellplanes.go, the stepRate note in
// step.go, the release witnesses in pkg/game/areacost_release_test.go and this
// note. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises 261 for the campaign-end release witness
// pkg/game/heisdead_release_test.go and this note.
//
// CommentBytes rises for mod clothing layers: the docs of the new
// pkg/mapload/layers.go, pkg/game/modlayers.go, the layered figure composers in
// inventory.go and portrait.go, the mark's layer notes, the release witness and
// this note. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the special hover texts: the docs of the new
// pkg/game/spellcaption_test.go, pkg/game/tooltipspecial_release_test.go,
// pkg/ui/tooltip_special_test.go, the spellPopupLines and spellCaptions docs in
// pkg/game/spell.go, the map list column docs in pkg/ui/picker.go and
// tooltip_targets.go, and this note. The value is a fresh measure on the tree
// with this note.
//
// CommentBytes rises for base profiles: the docs of the new pkg/base/base.go
// and base_test.go, pkg/game/base.go, base_release_test.go and
// demobase_test.go, cmd/againrom/base_test.go and cmd/starter/base_test.go,
// plus the doc comments the profile adds to installinfo.go, the ui App and
// flow, cmd/againrom/main.go, cmd/starter/app.go and this note. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for mod character edits: the docs of the new
// pkg/mod/characters.go, pkg/data/rowedit.go, pkg/game/modcharacters.go,
// pkg/mapload/modcharacters.go, pkg/modrt/characters.go, the tests and release
// witnesses of those, and this note. The value is a fresh measure on the tree
// with this note.
//
// CommentBytes rises for the Fire_Ball burst record: the docs of the new
// pkg/sim/burst.go and its test, the restored-run seeding note in
// pkg/game/spellbolt.go, and the two release witnesses
// pkg/game/projectileflight_release_test.go and
// pkg/game/fireballburst_release_test.go, all genuinely new code. The bound-id
// follow, the burst owner note and the loaded-save burst witness in
// pkg/game/projectilesave_release_test.go add the same kind.
//
// CommentBytes rises for mod screens: the docs of the new pkg/mod/screens.go,
// pkg/ui/screenhandler.go, pkg/ui/modscreens.go, pkg/game/modscreens.go and the
// screen tests and release witnesses, plus the doc comments the registry adds
// to app.go, flow.go, gamemenu.go and save.go. The value is a fresh measure on
// the tree with this note.
//
// CommentBytes rises for the mod mission actions: the docs of the new
// pkg/game/missionleave.go, the action parsing in pkg/mod/screens.go, the
// action rows and confirming page in pkg/ui/save.go and modscreens.go, and the
// new pkg/ui/modaction_test.go and pkg/game/modabandon_release_test.go, and the
// carry of a restored mission's party in missionleave.go, frontend.go and
// world.go with its release witnesses in modabandonloadparty_release_test.go.
// The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the companion join condition: the docs of the new
// pkg/mod/companions.go and pkg/game/modjoin.go, the split of
// addChapterCompanions into carryTownCompanion and settleCompanionArrival, the
// held-grant takes in town.go and campaignprogress.go, and the tests and the
// release witness pkg/game/modjoin_release_test.go. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the mod item name hotfix: the install-alphabet note in
// pkg/game/moditems.go. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the mod item debt hotfix: the docs of modStockChance,
// modStockOffered and refuseLayer and the notes of the new release witnesses. The
// value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the book-chosen spell hotfix: the doc of spellModeLive and
// the new witness. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the F1 help panel and the C key: the docs of the new
// pkg/ui/help.go, the castKey and armCast docs in quickspells.go, the help
// handlers in app.go, and the notes of help_test.go and the release witness
// pkg/game/help_release_test.go. The value is a fresh measure on the tree with
// this note.
//
// CommentBytes rises 1479 for the cheap defect batch: the docs of
// World.GroupRateTerm, World.ScriptPassJustRan, World.ScriptTriggerBlocked and
// Town.permanentLoaded, the announcer's repeating-trigger notes and the notes of
// the new tests and corpus witnesses. The value is a fresh measure on the tree
// with this note.
//
// CommentBytes rises 4879 for the loaded-cycle order routes and Retreat geometry:
// the docs of retainCycleForState, fleeCell and moverFinePoint and the notes of the
// new sim tests and the release witnesses pkg/game/cyclewriters_release_test.go and
// pkg/game/loadedcycle_corpus_release_test.go. The value is a fresh measure on the
// tree with this note.
//
// TestIdentCount falls by six and CommentBytes rises for the town square
// families: the shared installed-square oracle drops its story number from its
// type and loader, and the docs of pkg/game/townfamilies.go, the ui family
// frame and the new tests carry the rise. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the area-layer cost timing and Prismatic Spray group
// sight: the docs of pkg/sim/layercostwindow.go, strideCrossing and
// ResetLoadedAreaCosts, and the new tests. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the refused attack pursuit answering with the nearest
// hostile: the docs of attackerRefusable, aimsAtDestination and
// answerRefusedPursuit and the notes of the new sim tests. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the help panel scroll bar hotfix: the docs of the new
// helpPointer and its hold parts in pkg/ui/help.go, the headless bar point, and
// the new tests. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the k121 adoption: the docs of scriptGroupOwner, the
// off-map regeneration skip, the script unit binding, itemSpellFragment and the
// notes of the new sim tests and the release witnesses
// pkg/game/scriptgroupowner_release_test.go, pkg/game/offmapregen_release_test.go
// and pkg/game/bookcard_release_test.go (with its armour card witness and
// wornCardLines in iteminfo_test.go), and the resolved-group oracle in
// cmd/scriptcoverage/controlled.go. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the shop grid quantity: the doc of drawShopQuantity and
// the note of the release witness in pkg/game/shopquantity_release_test.go. The
// value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the wide load witness: the comments of the new
// pkg/sim/currentwideload_test.go. The value is a fresh measure on the tree with
// this note.
//
// CommentBytes rises for the mission hover cursor rows: the docs of
// structureHoverHit and HeadlessMapCursor and the notes of the new hover and
// slot census tests, and the click test with the headless ctrl- pointer prefix. The
// value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the map list description block: the field note in
// alm.Info. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the unit-shot build tick and the Fire_Ball transport
// target: the docs of unitShotSwingTick and spellDeliveryRemaining, the notes
// of pkg/sim/fireballaim_test.go and the release witnesses in
// pkg/game/projectileflight_release_test.go and
// pkg/game/projectilesave_release_test.go. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the restored delay 0 shot record: holdsDelayZeroRecord
// and the notes in seedRestoredRuns and its unit test. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises for the scripted Stand Ground and the refused pick-up
// walk: the docs of standScriptedMembers and answerRefusedPickupWalk in
// pkg/sim, the notes of pkg/sim/scriptedstand_test.go,
// pkg/sim/refusedpickup_test.go, pkg/sim/walkoffset_test.go and the release
// witnesses pkg/game/scriptstandground_release_test.go and
// pkg/game/refusedpickup_release_test.go. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the footprint-centre and teardown rules: the notes of
// pkg/sim/footprintstep_test.go, pkg/game/footprint_release_test.go and
// pkg/sim/teardownrelease.go, and the docs of requestCellCast. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the settings key notices: the docs of
// pkg/ui/settingnotice.go, Words.SettingNotice and PostMessageUnlessNewest, and
// the notes of pkg/ui/settingnotice_test.go and
// pkg/game/settingnotice_release_test.go. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the unit voices: the docs of speakerOf, unitReplies
// and commandRecording in pkg/game/commandvoice.go, VoiceGesture and the reply
// helpers in pkg/ui/commandvoice.go, voiceTier in pkg/game/humansound.go, the
// drawnMoving field in pkg/game/world.go, the new pkg/game/commandvoice_test.go
// and pkg/ui/voicegesture_test.go, and this note. The value is a fresh measure
// on the tree with this note.
//
// CommentBytes rises for the creature spellbook casts: the docs of
// pkg/sim/creaturecast.go, the notes of pkg/sim/creaturecast_test.go and the
// release witnesses pkg/game/creaturebooks_release_test.go. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the closed route cell correction: the subGoal note and
// the arrival test's doc. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for cutscene presentation: the docs of the new
// pkg/video pacer, sidecar and presenter files and their tests. The value is a
// fresh measure on the tree with this note.
//
// CommentBytes rises for the ending route: the docs of endingSeams, endingLeave,
// leaveCampaignEnding and the hall-entry and bind notes, and the new ending tests.
// The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the unit shadow arms: the doc of the unit shadow pass in
// pkg/ui/shadow.go, UnitClass.Boundary and BoundaryOf, boundaryFrames and the two
// new shadow tests. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the keyboard remainder: the docs of the frame-rate
// readout in pkg/ui/fpsreadout.go, suppressAltLetters, showSpellBar and the new
// keyboard tests. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the held move's first walk call: the doc of
// endHeldCycleForWalk and the walk-offset tests. The value is a fresh measure on
// the tree with this note.
//
// CommentBytes rises for the footprint-centre aim and heading: the docs of
// headingOf and headingBetween, the aim note in spellDeliveryRemaining and the
// new fire ball, heading and unit shot offset tests. The value is a fresh
// measure on the tree with this note, and the merge with main.
// CommentBytes rises for the shop and school presentation: the docs of the
// teacher speech latches in pkg/game/townresponse.go, the idle shine clock in
// pkg/game/schooltraining.go and the notes of the new quantity, pack readout,
// idle shine and teacher speech tests. The value is a fresh measure on the tree
// with this note.
//
// CommentBytes rises for the shadow writer witnesses: the docs of the new
// hero-pair and translated-pair release tests in pkg/game. The value is a fresh
// measure on the tree with this note, and the merge with main.
// CommentBytes rises for the town square families: the docs of the position
// roll helpers, TownViewOrigin and their tests. The value is a fresh measure
// on the tree with this note.
//
// CommentBytes rises for the credits roll and hall input: the docs of the roll's
// step constants and pitch in pkg/ui/media_menu.go, the hall rectangle in
// pkg/ui/ending.go and the notes of the new roll, hall and reset tests. The
// value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the cutscene sound clock: the docs of the new
// pkg/video/soundclock.go, the played-bytes note in pkg/ui/cutsceneaudiodev.go
// and the notes of the new sound-wait tests. The value is a fresh measure on the
// tree with this note.
//
// CommentBytes rises for the hurt voices: the docs of the zero-strike report in
// pkg/sim/castevent.go, the strike, bleed and speech-device rules in
// pkg/ui/sound.go and the notes of the new sim, ui and game hurt tests. The
// value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the effective skill level: the docs of
// rules.EffectiveSkill, the skill lift in pkg/sim/sourceactor.go, the SAV level
// projection in pkg/game/savskilldomain.go and the notes of their tests.
//
// CommentBytes rises for spell power above 100: the docs of lastingTicks,
// segmentedTicks, spellLastingTicks, clampByte and the power tables, and the
// notes of the new power tests and release witnesses, and bookRootValueMatches.
//
// CommentBytes rises for the border-ring placement refusal: the doc of
// mapload.WithdrawBorderPlacements, the notes of its tests and of the event
// dialogue witnesses.
//
// CommentBytes rises for the purse gold drop: the docs of the new
// pkg/sim/dropgold.go, pkg/ui/goldmodal.go and pkg/game/purse.go and the notes of
// their tests. TestIdentCount falls by 26 because two saved-objects test
// helpers lose their numeric suffix in the five test files that use them.
// CommentBytes rises for the panel-set and pause hotfix: the docs of
// barePauseKey, packBarShown, endCastOnce and drawNoHeroText, the Words field
// note, and the C key and empty-panel notes; all new code.
//
// CommentBytes rises for the tester batch 2 hotfixes: the notes of the new
// crowd-front, teleport-approach, worn-item and unworn-item tests, the settle
// and approach docs in pkg/sim/route.go, pkg/sim/manualcast.go and
// pkg/game/rearm.go, and the card name wrap in pkg/ui/character_card_style.go.
// The release fixtures that keep clear of the changed crowd front add notes in
// pkg/game/spellcontinuation_release_test.go and
// pkg/game/minimapfallenmark_hotfix_release_test.go. The value is a fresh
// measure on the tree with this note, and the merge with main.
//
// CommentBytes rises 5375 for the autosave worker: the docs of the new
// pkg/game/autosavequeue.go, the ordering and exit notes in savedialog.go,
// timedautosave.go and pkg/ui/timedautosave.go, and the notes of the new
// autosave tests. The value is a fresh measure on the merge with main.
//
// CommentBytes rises for the cast cursor by spell kind: the docs of
// pkg/ui/castcursor.go and gameAreaAt in pkg/ui/cursor.go, spellSelfOnly in
// pkg/game/spell.go and the new viewer and release tests; all new
// code. The value is a fresh measure on the tree with this note.
//
// CommentBytes rises for the generated mission release test's staged batch
// helpers in generatedmissions_release_test.go; test code only.
//
// CommentBytes rises for the Drop Gold editor's Alt+Backspace rule: the note in
// stepGoldModal and the two new tests; all new code.
//
// CommentBytes rises for the shop pane source release witness: the notes of
// the new pkg/game/shoppanesource_release_test.go and this one; test code only.
// CommentBytes rises for the campaign lifecycle split: the doc comments of
// the new pkg/game/campaignsession.go, missionentry.go, missionaudio.go and
// missionart.go, and the tests' docs in campaignsession_test.go. The comments
// that described the mission-entry sequence moved with their code from
// frontend.go and are not new.
// CommentBytes rises for the return and restore split: the docs of the new
// pkg/game/campaignreturn.go, missionports.go and restoredecode_test.go, the
// port docs in missionentry.go, decodeRestore and decodeOriginalCampaign, and
// the tests' docs in campaignreturn_test.go and campaignsession_test.go. The
// comments that described the moved operations moved with their code.
// CommentBytes rises for the Fire Sacrifice self kind: the docs of spellSelfKind
// and fireSacrificeSpellID and the comments of the new cursor tests.
//
// CommentBytes rises for the original-SAV restore split: the docs of the new
// pkg/game/originalrestore.go, the world-map data and marker functions in
// pkg/game/worldmap.go, the plain squad builder in pkg/game/tavern.go, the
// routing and leave operations in pkg/game/campaignreturn.go, the install-level
// art methods in pkg/game/menupanel.go, characterpane.go and tips.go, and the
// tests' docs in originalrestore_test.go. The comments of restoreOriginal moved
// with their code.
//
// CommentBytes rose with the hand-over group record's doc comment in
// pkg/sim/script.go, the matching comment in pkg/sim/engage.go and the
// regression witnesses in pkg/sim/handovergroup_test.go and engage_test.go,
// including this note.
//
// CommentBytes rises 2551 for the recovery hold: the docs of cycleLoaded and
// holdsCycleFor in pkg/sim/pendingorder.go and the comments of the new
// pkg/sim/recoveryorders_test.go and the changed lost-target check in
// playerdefend_test.go, all new code.
// CommentBytes rises for the doc comment of TestIndexListsEveryAreaFile in
// internal/divledger/divledger_test.go, new code, net of this paragraph.
// The comment forms acclause, scclause and barestorynumber (a zero-padded
// four-digit run) are ratcheted at their measured counts and may only fall.
// Counts holds the directory-name, non-.go file-name, struct-tag,
// string-literal and long-comment-group ratchets, read by the filesystem
// walker and the go/ast walker in scan.go. The long-group count is the number
// of comment groups over MaxCommentGroupLines lines.
// CommentBytes rises for the new walker's, counters' and ceiling's own doc
// comments in scan.go, check.go and storyguard_test.go, new code, including the
// git-ignore filter of the name walker.
// CommentBytes rises for the doc and step comments of the new campaign chain
// test in pkg/game/campaignchain_release_test.go, new code, net of this
// paragraph. TestIdentCount falls 68 with the removal of the 1176 suffix from
// seven shared test helpers.
// CommentBytes rises for the doc comments of the document payload codec in
// pkg/formats/sav/docpayload.go, the carrier fields in campaign_tail.go and
// city_data.go, pkg/game/docpayload.go and the new tests in
// pkg/formats/sav/docpayload_test.go and pkg/game/docpayload_test.go, new code,
// including this paragraph.
// CommentBytes falls for the sweep that replaced the executable locators in
// comments with opaque aliases; the value is a fresh measure including this
// paragraph.
// CommentBytes rose for the enemy card knowledge level and the Diary kill writer:
// pkg/sim/diaryknowledge.go, pkg/mapload/diaryunits.go, pkg/formats/alm/alm.go, pkg/game/townknowledge.go,
// the panel gates in pkg/ui/panel.go and the new tests in
// pkg/sim/diaryknowledge_test.go, pkg/ui/cardknowledge_test.go and
// pkg/game/enemycard_release_test.go, pkg/game/enemycardtown_release_test.go, and the full-knowledge notes in
// pkg/game/creaturemagic_release_test.go, pkg/game/cardload_release_test.go and
// pkg/game/loadedsheets_release_test.go, all new code, including this paragraph.
// CommentBytes rises for the tavern statistics card's offset: the docs of the
// new cardRect, cardBoard, shiftedCardBackground and tavernCandidateCardOffset
// in pkg/ui/townshell.go, new code, including this paragraph.
// CommentBytes rises for the bank pursuit hotfix: the docs of pursuitRings and
// pursuitGoalIsVictim in pkg/sim/route.go and the test comment, new code,
// including this paragraph.
// CommentBytes rises for the skill level hotfix: the docs of RepairSkillLevel,
// SightWord, skillrepair.go, snapshotHumanFields and the writer census test
// file, new code, including this paragraph.
// CommentBytes rises for the school shine hotfix: the doc of schoolShineDisplaySlot
// and the test comments, new code, including this paragraph.
// CommentBytes rises for the pack no-hero line and the wheel over the pack: the
// docs of drawPackNoHeroText and the two new tests, including this paragraph.
// New constructor Base documentation is in pkg/mapload/nativeinitialbasis.go
// and its test. The comment count includes this baseline explanation.
// New local constructor Modifier tests and UpdateNativeEquipmentBasis document
// independent initial history. This measure includes this explanation.
// New native subject comparison documents its independent metadata boundary.
// The measure includes this explanation and neutral fixture names.
// Native live-block and Item-record state adds new-code comments in sim/item.go,
// game/currentnativeitem.go, and the new raw, item and route controls. Shared
// acceptance fixture names now use their subjects instead of story numbers.
// SetNativeTraining in pkg/sim/training.go documents the new selected-byte
// repair seam. The comment count includes this explanation.
// Native scalar state, legacy name membership and their independent loss controls
// add new-code documentation, including this measured baseline explanation.
// New native effect-mask controls document their attachment and expiry inputs in
// sim/nativeeffectmask_test.go and game/savactoreffectcurrent_test.go. This measure
// includes that new-code documentation and this explanation.
// CommentBytes rises for the World-built SAVE: the docs of the unknown-span
// graft, the departed-actor and departed-member bodies, the original world
// spell effects, the restored and frozen order carriers, the joined actor
// identity, the town option carrier and the loaded-document census and
// loss-control tests, and of the World's held orders, the graft's held
// bases and the consumed-corpse held-byte control, including this paragraph.
// The pursuit search adds new code documented in pkg/sim/pursuitsearch.go,
// pickerb.go, pursuitsearchbinary.go and their tests, and the release drives
// it moved document their new set-ups, as do the structure-order, remap and
// head-removal fixes and their tests, net of the removed stand-in docs,
// including this explanation. Reconciled with the melee facing turn, the
// spell graph check names the area drivers it guards, and the release drives
// that moved again document their set-ups and a new witness, in new code.
// The acquisition drive documents the Defend arm's release past reach.
// The mission 10 heal drives document the wound they now set near the mage.
// CommentBytes rises for the melee facing hotfix: the docs of the new melee
// walking witness, the Kadagan release test's pack note and the skill
// trainee's heal note, new code, including this paragraph.
// CommentBytes rises for the shared widget kit: the docs of the new push
// button, bar, list, slider, radio and checkbox, edit field and hover box
// builders in pkg/ui, of their focused and per-screen tests, of the notice
// picture accessor, the hover corner accessor, the Load double-click
// release and the EN/RU widget screen witness, and the release witnesses
// restated to the shared bar, new code, including this paragraph.
// CommentBytes rises for the widget kit's input corrections: the widget
// latch reset on focus loss, checkbox press and Space, radio arrows and Tab,
// the Save caret paint order and their tests, new code, including this paragraph.
// CommentBytes rises for the Skrakan portal-part release witness's comments,
// new code, including this paragraph.
// CommentBytes rises for the second game's campaign support census: the docs
// of pkg/game/secondcensus.go, its focused and release tests, the
// cmd/campaigncensus package doc and archtest row, and the four helpers it
// reads (secondContinues, completeBank, secondCompletionMovie,
// sim.SpellRuleApplicable), new code, including this paragraph. It rises
// again for the docs of each departure exit and its bank gates.
// CommentBytes rises for the second game's spell arms: the docs of
// SpellRule.Arm and Second, the pkg/sim/secondspell.go helpers, the mapload
// arm assignment and their focused and release tests, new code, including
// this paragraph. It rises again for the attached-effect lookups by arm and
// their tests.
// CommentBytes rises for the second game's ordinary departure: the docs of
// completeBank's output, secondAux and its decoder, the stage table, the
// later-town departure and the movie chosen at acknowledgement, and the
// focused and installed tests of every case body, new code, including this
// paragraph. It rises again for the town inn options and TALK: the docs of
// EnterInn, the speakers and TalkTo, and the focused and installed tests of
// the stage-30 inn route, new code, including this paragraph.
// CommentBytes rises for the remaining inn stage bodies: the docs of the
// stage entry table and its predicates, and the focused and installed tests
// of the stage 40 to 110 entries, new code, including this paragraph.
// CommentBytes rises for the staff projectile light: the note in
// objectLightStamps and the doc of its new focused test, new code, including
// this paragraph.
// CommentBytes rises for the fresh party member's dying time: the docs of
// pkg/mapload/partyhumanrow.go and its focused and installed tests, and the
// corpse-entry fixture's served-countdown note, new code, including this
// paragraph.
// CommentBytes rises for the town square trace instrument: the docs of its
// recorder and script, new code, including this paragraph.
// CommentBytes rises for the town composer: the docs of pkg/town's description
// format, actor programs and view, which replace the square's per-actor code,
// new code, including this paragraph.
// CommentBytes rises for the town room pages: the room trace instrument, the
// reader's refusals, pkg/town's own tests and its page programs, new code,
// including this paragraph.
// CommentBytes rises for the frame builder: the docs of the frame rows, the
// tiling and the shadow tone in the new widget kit file, which replace the
// separate window, tip, panel and border painters, new code, including this
// paragraph.
// CommentBytes rises for the one press latch: the latch package docs and
// the notes on the dialogue capture and the generation page latch, new code,
// including this paragraph.
// CommentBytes rises for the widget kit scan in internal/archtest and the
// latch site witness, new code, including this paragraph.
// CommentBytes rises for the latch drop on a left screen: its doc and the
// leave witness, new code, including this paragraph.
// CommentBytes rises for the SAV byte producer list in internal/archtest's
// saveproducer_list.go and its guard doc, new code, including this paragraph.
// CommentBytes rises for the random service: the docs of pkg/random, of the
// World stream's two modes and of every consumer moved onto a named stream,
// new code, including this paragraph.
// CommentBytes rises for the game-profile scan's new forms, its mutation
// cases and debt list in internal/archtest, the edition's Rooms field and
// the room description reader in pkg/game, new code, including this
// paragraph.
// CommentBytes rises for the tip witnesses: the docs of the new release test
// file for the generator cycles and the tip renders, the scroll witness's
// note that the mission start tip covers its click, the room tip witnesses,
// the edition's mission tip field and the tip list rectangle's doc, new code,
// including this paragraph.
// CommentBytes rises for the engine words: the docs of the new pkg/words and its
// tests and the engine-words scan in internal/archtest, new code, including
// this paragraph.
// CommentBytes rises for the game-profile scan's further forms and the widget
// kit scan's push button rule in internal/archtest, their mutation cases and
// debt lists, new code, including this paragraph.
// CommentBytes rises for the slot-0 resistance witnesses: the docs of the
// focused strike tests in pkg/sim and of the corpus release test in
// pkg/game, new code, including this paragraph.
// CommentBytes rises for the random service remainder: the AI range idiom,
// the count reseeds, the generator restart, the edition's generator name and
// the wider randomness scan, new code, including this paragraph.
// CommentBytes rises for the unit-shot tracking release witness and its
// helpers, new code, including this paragraph.
// CommentBytes rises for the Human speed derive: the shared derive in
// pkg/rules, the native speed modifier, its byte-form section and LOAD split
// in pkg/sim, the spawn speed word in pkg/mapload, and the overload order
// tests in pkg/rules, pkg/sim and pkg/game, new code, including this
// paragraph.
// CommentBytes rises for the one chat command parser, its two game adapters,
// the second game's completion win predicate and the second game's command
// witnesses, new code, including this paragraph.
// CommentBytes rises for the world-map task flag and held Cross: the frame
// pickers, the flag's counter and placement, and their focused and release
// tests, new code, including this paragraph.
var Committed = Baseline{
	TestIdentCount: 4694,
	TestFileCount:  264,
	CommentForms: map[string]int{
		"specclause":      0,
		"storymention":    0,
		"calendardate":    0,
		"rom1address":     0,
		"funaddr":         0,
		"expmention":      0,
		"acclause":        1877,
		"scclause":        506,
		"barestorynumber": 1741,
	},
	Counts: map[string]int{
		"dirnames":               0,
		"nongofilenames":         45,
		"structtags.nontest":     0,
		"structtags.test":        0,
		"stringliterals.nontest": 1,
		"stringliterals.test":    34,
		"longcommentgroups":      1767,
	},
	CommentBytes: 8591275,
}
