# Dialogue renderer match

## Result and authority

Owner goal item 9 matches the selected public dialogue wrapper, justified placement, frame/shadow, portrait copy and OK state contracts in town and mission. The released base is `1b0e8d06ea0d51ef4f31be65b9e9b0c6888e5b35`. The exact public pin is `c91bb3972d7ec1184ae49fae885691591d29337d` (k102). Authority is DIALOGUE-060..065, DIALOGUE-068..070, amended DIALOGUE-062 and the amended DLG-PANEL-035, DLG-PORTRAIT-036, DLG-LINE-038 and DLG-BUTTON-039, including their narrowed entries. DIALOGUE-055 supplies disabled level-3 remapping.

## As built

The actual town and mission resource callers deliver the accepted interval after header LF, before the CR preceding a next tag or at NUL/end. They preserve boundary/interior bytes and remove the header exactly once. Missing LF fails; a backward range before body start is safely rejected. Generic EventPart remains a raw tag-body API.

The wrapper retains a leading CRLF empty piece and interior LF data. Later pieces and suffixes use the supplied six-byte ASCII TrimLeft/TrimRight service. Strict fit, whole over-wide words and ordinary space/CR markers remain. A nonprogressing remainder is emitted once whole. Measurement, visible line count, drawing and disclosure paging use this wrapper. The finite remainder/backward-range policy and supplied classification are disclosed in DIV-1619; native locale remains Unknown.

Justification adds integer width to the stored accumulator, then the stored gap, and stores the result as double after both additions. `NoticeLayout.DialogueArithmetic` supplies PC24, PC53 or PC64 and rounding; zero chooses nearest-even PC53. Native FPU configuration remains Unknown (DIV-1620).

Frame stream masks remap the actual scene through packed level 6 before normal body replacement. Nine shadow requests precede 24 body requests at the shipped geometry. RGB565/RGB555 and full/reduced lookup remain supplied policies. The frame clip is separate from the backdrop clip. CPU composition and GPU submission/log retain skip positions, alpha customization and method-C underlays. Before selective shadow lookup, each visible captured glyph cell and raw underlay folds its preceding tint stage, including cells outside the mask; clipped cells remain excluded. The capture then clears Tint for the whole glyph. Reached custom-frame/translucent-backdrop Viewer.Draw controls retain both partially shadowed cells and their exact logged colors without a second tint. Native format/table/clip and repaint remain Unknown (DIV-1621).

Portrait composition separates opaque background and keyed canvas copies, descending physical source rows, reversal, default/record windows, border and final keyed copy. The engine chooses an RGBA canvas adapter explicitly; native upstream canvas contents, orientation and exposed rows remain Unknown (DIV-1622). Speaker construction and equipment remain game-owned.

The OK state keeps hover, press, membership and enabled flags separate. Packed idle/hover ramps and bevels reach App town and mission input. Press plus membership swaps bevels and changes shadow 2 to 4; press outside keeps shadow 2. The modal cache includes visual state and stable content revision retains matching-down/up ownership. These flags remain client presentation and do not enter SAV or hashed simulation.

## Proof

Initial reached regressions preserve failures for long-word splitting, CRLF/LF and empty-piece distinctions, markers, repeated-remainder loss and retained-accumulator placement. Supplied precision controls distinguish double spill from a retained accumulator and PC64 two-addition spill from per-add double. Frame and portrait tests compare whole buffers and guards, masks, packed losses, zero/high-bit words, reversal, clipping and opaque/keyed copies. State tests reach App input/cache. The partial-shadow controls include no-wash partial and washed complete controls; additional scalar CPU controls discriminate captured Tint, source-over, clipping, opaque overwrite, later wash and second shadow across full/reduced RGB565/RGB555. Inexact GPU blend values remain Unknown in pixelLog; the scalar controls use an explicit CPU oracle and do not claim native output.

`TestReleaseDialogueSpilledCorpus` declares the DIALOGUE-062/070 candidate extraction and supplied acceptance/font/string premises. Production accepted tails equal the independent extraction on all 1420 candidates. Reached PC53/PC64 positions match independent per-add double: EN 688/2542/1854/11415 and RU 732/2650/1906/9550 blocks/lines/justified lines/word positions. Every projected source word survives the visible control. The raw tag-body input remains a separate leading-empty/clamp witness; its historical census and clipped-tail failures are preserved as distinct evidence.

The installed layout, backdrop town routes and mission routes retain shop, inn, mercenary, training, page/close, SAV cold-load and next Talk witnesses. Their pointer controls check hover, pressed-inside/outside and unarmed release. Existing native/wide/detached/row-fallback, custom menu/outcome, lifecycle and method-C tests pass in the ordinary touched packages. Focused and ordinary touched-package checks, the full `go test -trimpath -count=1 ./...`, clean gofmt, asset exclusion and claim-citation checks pass. The new env-gated corpus test is registered in `internal/gatedtests/testdata/population.txt`.

Installed quest and portrait witnesses derive the accepted LF/CR/NUL interval from resource bytes without whitespace trimming. Town remap comparisons freeze the presentation clock so both frames use the same cauldron phase. Reward, world hash, portrait, party and cold-SAV checks remain.

## Open bounds

No native executable, window, synthetic desktop input or GPU framebuffer readback is used. GPU proof is submitted operations and the CPU pixel log. Native accepted-tag population, loose overrides, locale/configuration, FPU, table/clip selection, canvas exposure, parent repaint, physical delivery, presented pixels and cadence remain Unknown. The sole adversarial pass and one correction are complete; main release gates and build promotion belong to coordination.
