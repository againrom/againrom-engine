# Town buttons and inspection fidelity

The owner reported flat or unresponsive plaques, disabled empty trades, missing
magic headings, anonymous spell books and understated siege statistics.

Tavern, shop, school and detailed-generator command plaques now show their
raised face at rest, brighten labels/numbers on hover, depress while a press
remains over its origin, and restore on release or departure. The shared
renderer uses the installed school raised/depressed pair with preserved corners.
Existing plaque wells, command keys, item dragging and hire/fire routing remain.
Shop release must match the button where the press began. Buy and Sell remain
clickable when empty and return without changing money, items or original-city
transaction provenance. Nonempty transactions keep their existing checks.

Item inspection adds one localized magic heading before enchantments and keeps
Value last. The heading comes from installed main.txt[189]. Spell books use a
Book of / Russian book prefix and the installed spell.txt name; their teaching
effect no longer produces an extra cast/teach row. Both common inventory and
shop inspection use this producer. Siege candidate and party cards resolve the
same creature template as mission spawning: the installed catapult has 250 HP
and 40-80 damage, the ballista 170 HP and 30-50 damage. Active potion bonuses
remain folded once above that baseline and disappear after expiry. Simulation
is unchanged.

Authority is the owner's requested behavior and installed resources. The public
MERC-LEVEL-005 claim distinguishes the two siege hires from generated humans.
The fitted artwork and authored book prefix remain disclosed in DIV-1306/1307.

Proof: TestReleaseTownFidelity1192 covers both installed languages, all 28 book
names, enchanted/plain items, candidate/party/spawn siege statistics, empty trade
state preservation, actual potion double-click/consumption and expiry, and App pointer dispatch in all four screens. Existing
shop tests retain insufficient-funds refusal, item dragging and statistics mode;
the latter now sends a complete press/release gesture. The paired release gate
must run all 306 registered tests before landing. Seat evidence records final
commit, executable witnesses and gate outcomes; no install bytes enter Git.

The reported orange cloak loss is not closed by this story. The inspected owner
mission111 save has plate cuirasses on its fighters; its dolls show those layers.
The proposed world-palette removal contradicted PAL-BLIT-024 and was rejected
without a code change. The exact affected portrait/equipment still needs a
reproducible witness. No cloak restoration or original pixel parity is claimed.
