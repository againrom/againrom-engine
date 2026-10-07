# 0169 — the campaign script's broadcast message

## Subject

A campaign map's script can raise a numbered announcement. The action is instant opcode 2. The
number it carries names a text file inside the install, and the text is shown to the player in
the dialogue panel this build already draws for mission events.

Before this story the simulation reported opcode 2 as an action it cannot run. Every remaining
`UNSUPPORTED` line in the campaign census is that opcode: 13 on mission 10, 11 on mission 20.
The text half — the address, the read, the panel, the paging, the portrait and the silence of a
file that does not ship — landed in earlier stories. What is missing is the opcode itself, the
conditional part selection the text markup asks for, the reserved message number, and a way to
observe an announcement with no window.

## Contract

### FR-1 — instant 2 is an action this build runs

Instant opcode 2 is a supported script action. A trigger whose action slot names it fires and
advances exactly as a trigger naming any other supported action does. The compiled script
reports no gap for it, so the census counts it nowhere.

### FR-2 — running it changes no simulation state

The action's whole effect is outside the simulation. Running it writes no register, no relation,
no entity field, no counter and no latch beyond the trigger's own. Two worlds that differ only
in whether an instant-2 node ran produce the same tick, the same digest and the same serialized
bytes, and the serialized form's version does not change.

### FR-3 — a raised announcement resolves to the mission's own event text

An announcement raising number `e` in mission `m` resolves to the entry
`main\text\battle\m<m>\event<NN>.txt` of the install's main archive, with `<NN>` the number
written to two digits and `<m>` the mission's own number and never a value carried by the
announcement. An entry that does not ship resolves to nothing at all: no panel, no placeholder,
no message and no change to any other state.

### FR-4 — the eight conditional tags select which part is shown

An event text is a sequence of tagged parts. A tag naming the wanted part may additionally carry
any of eight conditional literals. Each is a **substring** test over the tag body, each either
rejects the tag or falls through to the next literal, and a rejected tag does not end the search:
the scan continues to the next tag naming the same part, and the part is absent only when no tag
naming it survives.

The four `iam*` literals test the **player's own hero**:

| Literal | The tag is rejected unless the hero is |
|---|---|
| `iamfemale` | female |
| `iammale` | not female |
| `iammage` | a spellcaster |
| `iamfighter` | not a spellcaster |

The four bare literals test the **speaker** — the person the tag's `npc=` number names — with
the same four questions: `female`, `male`, `mage`, `fighter`.

The literals are evaluated in the order given, `iam*` first. Because the test is a substring
test and a literal that does not reject falls through, `iamfemale` also satisfies `female`,
`iammale` satisfies `male`, `iammage` satisfies `mage` and `iamfighter` satisfies `fighter`. One
guard exists and is reproduced: the `male` arm is skipped for any tag body that contains
`female` anywhere.

The consequence is authored behaviour and is not corrected here: where the speaker arms are in
force, a part tagged `iamfemale` additionally requires a female speaker, and an
`iamfemale`/`iammale` pair read to a female player by a male speaker satisfies neither tag, so
that part is absent.

### FR-5 — the four speaker arms are gated

The four bare literals are skipped whole — neither rejecting nor selecting — unless both hold:
the tag's `npc=` number names a record that carries a key named `Start`, and that record
resolves to a speaker. Either failing skips all four, so a tag body carrying `female` alone is
accepted by any hero.

### FR-6 — message number 255 is reserved

A script raising message number 255 does not resolve an event text. It opens the mission-lost
notice, which is the panel the original's mission-lost path posts on the same window message
with the same value. No shipped campaign map raises it: the shipped range is 0..25.

### FR-7 — an announcement is observable with no window

The no-window mission runner records every announcement a drive raises, in the order raised:
the message number, whether it resolved to a shipped text, how many parts that text holds, and
the part the drive's own hero receives after FR-4 and FR-5 have been applied. A scenario file
can assert the numbers raised. Nothing recorded reaches the simulation: a drive that records
and a drive that does not produce the same ticks and the same digests.

## Acceptance

- **AC-1** — Driving each of the 28 campaign maps reports zero unrunnable script **instants**.
  The scope is instants alone: the sweep's `unsupported` column also counts unimplemented
  **check** opcodes, which no part of this story implements, so that column falls to what the
  checks leave and not to zero.
- **AC-2** — Driving a mission whose script raises an announcement produces that announcement's
  number and its text in the no-window runner's own output, with no window opened and no
  install-reading test added to `go test`.
- **AC-3** — For every combination of hero sex and class, a synthetic event text carrying an
  `iamfemale`/`iammale` pair yields exactly one part, and the one that matches the hero.
- **AC-4** — A synthetic event text whose only tag for part 1 is rejected yields no part 1, and
  the panel does not open.
- **AC-5** — Two nodes raising the same message number produce two announcements, not one. The
  shipped campaign authors 254 nodes over 242 distinct (map, number) pairs, so 11 pairs are
  authored more than once and one is authored three times; nothing de-duplicates them.

## Properties

- **P-1** — The announcement pass reads the world and writes nothing to it. A world stepped
  with an announcer attached and one stepped without it are equal in every field and hash.
- **P-2** — Part selection is a function of the payload, the part number and the audience alone.
  It opens nothing, reads no clock and allocates no state that outlives the call.
- **P-3** — Selection is decided once per window, when it opens, and paging re-applies it per
  part with the same audience facts, so a page cannot show a part the same audience would have
  been refused.

## Cut list

- **SC-1** — `sound=` is parsed to nothing. The literal is recognised as a tag body like any
  other and no wave is composed or played. Audio on this surface is out of scope for this
  story; the tag is unused by every shipped `text/battle` file, so nothing shipped is affected.
- **SC-2** — The speaker gate (FR-5) is implemented as a lookup of a **key** named `Start` in
  the record. No shipped record carries such a key, so the four speaker arms are inert over
  shipped data. Lifting this is one predicate at one call site.
- **SC-3** — The speaker's own sex and class are taken from the record's flag tokens, with the
  record's `MySex`, `MyClass` and `Me` tokens taking the player's hero. This is authored: it is
  reachable only when SC-2's gate is lifted.
- **SC-4** — `npcalive=` and `npcdead=` are not evaluated. They are unused by every shipped
  `text/battle` file and what they do once matched is not established.
- **SC-5** — The two-digit format of FR-3 is not widened. A message number above 99 composes a
  wider name, exactly as the original's `%02d` does, and nothing shipped reaches it.
- **SC-6** — An open panel does not survive a save. The snapshot carries the simulation's bytes
  and the front-end's per-map residue, and an open notice is in neither, so a save taken while
  a panel is up resumes with no panel. The announcement is **not** re-raised on resume: the
  announcer seeds its memory from the world's own latch array when the mission opens, so a
  trigger that had already latched is not a rising edge. The message is therefore lost rather
  than repeated, which is the same trade the outcome notice already makes.
