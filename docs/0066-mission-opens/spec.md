# 0066 — the mission opens in the game

**Intensity: spec-first / static. Terrain: brownfield** for the text renderer, the map screen and
the map driver; greenfield for the event-text reader and the announcement derivation.

## Why

A campaign mission can be built and stepped, and no part of the game can open one. The map screen
opens a *map*; the mission — the thing with a start cell, a script, words and an ending — is
reachable only from a test. This story makes a mission openable and legible from the game itself:
a player starts a campaign mission, sees what the mission says, and sees how it ended.

"Legible" is the point. A mission that runs correctly and shows nothing is indistinguishable, from
the seat that matters, from one that does not run at all.

## Vocabulary

- **Mission number** — the campaign's own number for a mission. Its map is the entry
  `scenario/<number>.alm`.
- **Address** — the string that names an entry across the opened archive set, carrying the
  container's identity first: `main/text/battle/m10/event01.txt`. A failure "names the address"
  when its message contains that string.
- **Announcement** — an authored script action that raises an **event number**; the mission number
  and the event number together name the text to show.
- **Event text** — the resource an announcement names, at the address
  `main/text/battle/m<mission>/event<NN>.txt`, where `<NN>` is the event number in decimal padded
  to two digits. It is markup divided into numbered **parts**, of which one is shown at a time.
- **Notice** — a box of text drawn over the map screen. It has two kinds: a **dialogue** notice
  showing one part of an event text, and an **outcome** notice reporting how the mission ended.
- **Outcome** — what the mission has come to: undecided, won, or lost.
- **Script pass** — one evaluation of the whole authored script. It happens on a fixed phase of the
  world's tick cycle, not on every tick.
- **Design space** — the 640x480 coordinate space the front end already lays its screens out in.
  All geometry below is stated in it.

## Functional requirements

### The door

**FR-1.** The game starts a campaign mission by number: `-mission <n>` on the command line opens
that mission's map screen at startup. The main menu and the map list remain behind it, unentered.
Without `-mission`, every existing path behaves exactly as it does today.

Three failures are distinguished, each exiting non-zero with a message naming the address it failed
at, and each distinguishable from the other two by its message: a number that names no mission, a
mission whose map entry is absent, and a map entry that will not decode. The existing headless
check mode accepts `-mission` too and reports the same outcome without opening a window.

**FR-2.** The mission is started with a party of exactly one member, standing where the mission's
own start puts it. The member carries a class key fixed by this project and nothing else; it is not
a hero record and this spec does not pretend it is one. Where the mission's start puts that member
is contracted elsewhere and is not altered here.

### The words

**FR-3.** When an announcement fires, the game reads the event text at the address the mission
number and the event number name (see Vocabulary), from the archive set the front end holds, at the
moment it fires. Nothing reads any event text when a map is loaded.

**FR-4.** An event text that does not ship produces **nothing at all**: no notice, no placeholder,
no message, no error, and no change to anything else. A mission that raises a number for which no
entry ships is not a broken mission and is not reported as one.

**FR-5.** A part is located by scanning the payload for tags delimited by `<` and `>`, lowercasing
each tag's body, and taking the first tag in file order whose body **contains** `part=<n>`. The
part's content is the bytes between that tag's `>` and the next `<`, or the end of the payload when
no `<` follows. A file yielding no part 1 shows nothing.

The substring test is reproduced, not corrected — AC-19 is the case it decides.

**FR-6.** An announcement is raised when a trigger fires whose action list carries the
raise-message action; the event number is that action's first parameter. A trigger carrying several
such actions raises each of them, in action-list order. A trigger that can never fire raises
nothing.

**FR-7.** A notice is drawn over the map screen, in the game's own lettering, and does not stop the
map being drawn or advanced. Its geometry, in the design space:

| | Dialogue notice |
|---|---|
| Box | 580 x 240, top-left at (30, 120) |
| Text area | 380 x 136, top-left at (48, 36) within the box |
| Button | 80 x 26, top-left at (200, 172) within the box |

The text is wrapped to the text area's width at a line pitch of the font's height plus two. The
number of lines drawn is the smaller of the lines the text produced and the lines the text area
holds; the rest is not drawn and the notice does not scroll. A line is broken at the last space
that fits; a single word wider than the text area is broken within the word.

An outcome notice is drawn in the same manner and is visibly distinct from a dialogue notice —
different words, and a different frame colour — but its own box size and placement are this
project's choice.

Where the front end has no font, no notice is drawn and the map screen is otherwise unaffected.

**FR-8.** A notice's lifecycle is driven by input alone; nothing closes it on a timer.

- Its button clicked, the RETURN key pressed and the ESCAPE key pressed are three routes to one
  action, **advance**. Each acts on the key's or button's press edge, so holding a key advances once.
- Advancing a dialogue notice shows part `n+1` if the file has one, and closes the notice if it
  does not.
- While any notice is open, ESCAPE advances it and does **not** leave the map screen.
- An announcement raised while any notice is open is **discarded** — not queued, not deferred, and
  not shown when the open notice closes.
- An open notice does not otherwise take input away from the map screen.

**FR-9.** Every string byte the game draws is converted before it selects a glyph, by a rule the
install's own language selector chooses:

- Under selector 1, a byte in `0x80..0xAF` has `0x30` added and a byte in `0xE0..0xEF` has `0x10`
  added; every other byte is unchanged.
- Under every other selector, every byte is unchanged.

The selector is the trailing ASCII digit of the entry `main/id`, taken as a number. An install with
no such entry, or one whose entry ends in no digit, takes selector 0 — the identity rule — so a
missing or unreadable selector can only leave drawing exactly as it is.

The glyph record is then the converted byte less `0x20`, taken as a byte. The existing bound is
**kept**: a record at or past the font's record count, and a converted byte below `0x20`, both
select the font's first record, the space. This is a **deliberate divergence** — the original
applies no bound here and reads past its own table.

### The outcome

**FR-10.** When a mission becomes decided, an outcome notice reports which way it went, dismissible
by the same three inputs FR-8 names. The world **keeps running** while it is up: the notice stops
nothing, and the mission's clock, its script and its units carry on exactly as they would have
without it.

A mission is decided once and the notice is shown once. An outcome notice **replaces** an open
dialogue notice, because the mission is over and its words no longer matter.

**FR-11.** Advancing an outcome notice ends the mission. Where it leads is where this story stops:

- A **lost** mission returns to the main menu.
- A **won** mission has nowhere to go: the campaign leads to a town and there is no town. The win
  path returns to the map list and states there, **in the running program**, that the destination
  is not built.

This is an **authored** boundary, not an open question. Nothing further is claimed about it.

*Folded from hotfix `dbd5c89` — see `docs/hotfix/ARCHIVE.md#dbd5c89`.* A dismissed win names the
mission that follows it, and the party that survived is carried into that mission rather than
re-minted.

**FR-12.** A started mission's world **runs that mission's own compiled script**: as the world is
stepped its triggers are evaluated, their latches set, and its outcome follows from them. Without
this nothing above raises anything and no mission can ever be decided, so FR-6 and FR-10 are
unreachable. A mission whose map authors no script, or whose script will not decode, starts anyway
and its world runs none; that is not an error. Starting a map with no mission is unchanged and
still builds a world running nothing.

## Acceptance criteria

| ID | Given | When | Then |
|---|---|---|---|
| AC-1 | an install and a number naming a mission | the game starts with `-mission <n>` | that mission's world is built and its map screen is shown |
| AC-2 | a number naming no mission; a mission whose map entry is absent; a map entry that will not decode | the game starts with each | each exits non-zero, names the address, and its message differs from the other two |
| AC-3 | no `-mission` | the game starts | the main menu opens, exactly as before |
| AC-4 | a started mission | the world is inspected | it holds one party member more than the map's own placements, carrying the fixed class key, at the mission's start cell |
| AC-5 | a fired trigger carrying a raise-message action, and a shipped event text | the pass that fires it completes | a dialogue notice shows that file's part 1 |
| AC-6 | a fired raise-message action and **no** shipped event text | the pass that fires it completes | no notice appears, nothing is reported, and no other state changes |
| AC-7 | an event text with several parts | the notice is advanced | part `n+1` is shown; advancing past the last part closes it |
| AC-8 | an open notice | a second announcement is raised | it is discarded, and closing the open notice does not show it |
| AC-9 | an open notice | ESCAPE is pressed | the notice advances and the map screen is not left |
| AC-10 | a text longer than one line | it is laid out | it breaks at the last space that fits, a word wider than the area breaks within the word, and no more lines are produced than the area holds |
| AC-11 | selector 1 | a byte from each moved block, and a byte from outside both, are drawn | the two moved bytes select their converted records and the third is unchanged |
| AC-12 | selector 0, no `main/id` entry, or an entry ending in no digit | a string is drawn | every byte selects the record it selected before this story |
| AC-13 | a byte whose record lies past the font's records | it is drawn | the first record is used and the pen advances; nothing reads past the font |
| AC-14 | a running mission | its outcome becomes won, or lost | one outcome notice appears, and the two are distinguishable from each other |
| AC-15 | a notice on screen | ticks elapse | the world's tick advances and its script keeps being evaluated |
| AC-16 | an outcome notice for a **lost** mission | it is advanced | the game returns to the main menu |
| AC-17 | an outcome notice for a **won** mission | it is advanced | the game returns to the map list and states there that the town is not built |
| AC-18 | a trigger that can never fire, carrying a raise-message action | any number of passes run | no announcement is raised |
| AC-19 | a payload whose first tag is `<part=10>` | part 1 is requested | the part-10 body is returned |
| AC-20 | a payload with no part-1 tag | part 1 is requested | nothing is returned and nothing is shown |
| AC-21 | an open notice | its button is clicked, or RETURN is pressed | it advances, once per press |
| AC-22 | a front end whose font failed to load | a mission runs and raises an announcement | no notice is drawn and the map screen is otherwise unaffected |
| AC-23 | a loaded map | the load completes | no event text was read |
| AC-24 | an open dialogue notice | the mission becomes decided | the outcome notice replaces it |
| AC-25 | a started mission whose script carries a trigger whose conditions hold | one whole script pass elapses | that trigger's latch is set, its instants run, and its announcement is raised |
| AC-26 | a mission started with no script | its world is compared with one started the same way before this clause | the two are the same world, digest for digest |

## Properties

- **P-1.** No file under `pkg/sim` changes. The simulation gains no field, its byte form gains no
  version, and its digest for any given world is what it was before this story.
- **P-2.** An announcement is raised at most once per firing, and exactly once for every trigger
  that fires once. A **repeating** trigger that fires on two consecutive passes raises once rather
  than twice — a disclosed divergence, bounded to that case.
- **P-3.** Under selector 0 every byte selects the record it selected before this story, so an
  English install renders what it rendered, everywhere text is drawn.
- **P-4.** No notice reaches the simulation. With the map screen driven identically, a world's tick
  and digest sequences are the same whether or not a notice was shown.
- **P-5.** Nothing this story adds reads a clock, a file or a device on the drawing path. A
  notice's picture is a function of its text, its font and its geometry alone.

## Out of scope

Combat, autonomous behaviour, saves, the town, and the campaign beyond the single mission opened.
The portrait pane and the speaker tag. Every markup tag other than the part tag. The rule-glyph
markup character. Scrolling a part past the clamp. Reading the original's own panel art or its
string table. Choosing a mission from a menu rather than from the command line. Changing where a
mission start places a party.

## Error cases

**FR-1**'s three startup failures are the only new error paths, and each names its address. FR-4,
FR-7, FR-9 and FR-12 each name a condition that is explicitly **not** one.
