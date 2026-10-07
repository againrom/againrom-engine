# Analysis — the dialogue window shows who is speaking

## What was reported

The product's author played the current build and reported that the dialogues carry no picture of
whoever is talking.

## What the baseline actually is

The omission is not "a feature was left out". `pkg/ui/notice.go` names the portrait **once, in a
comment**, and the paragraph is accurate: it quotes the panel rect, the button rect and the text
control's rect. But the text rect it quotes and ships is `48,36-428,172` — the rect the original
uses **only when there is no portrait**. The constructor holds two text rects and picks between
them; this tree holds one, and it is the other branch's.

So the file contains the right geometry as prose and the wrong geometry as behaviour, which is why
the gap survived a landing, a review and eleven subsequent stories: every reader who checked the
comment against the claim found the two in agreement.

The second half of the same shape: the layout this tree ships is the one **no shipped file uses**.
Measured here over both preserved roots, every event file mentions a speaker — so on shipped data
the original always takes the portrait branch, and this tree always takes the other one.

```
                              EN    RU
event*.txt under text/battle  225   228
containing "npc="             225   228
containing "npc"              225   228
```

## What is not established, and where the wall is

The window's **geometry** is settled and the **two tests** are settled. What no published claim
settles is where the picture comes from.

The chain a speaker id would have to travel is visible and its last hop is missing:

1. a part's tag carries `npc=<n>` — settled as vocabulary;
2. `<n>` is a `scenario.res::npc.reg` `[npc<n>]` subscript — every id the shipped corpus uses names a
   section of that registry, and the registry's own id space is published;
3. that section carries some combination of `Face`, `Picture`, `PortraitX1/X2/Y1/Y2` and a `Flags`
   token list — published, with the domains measured;
4. **those keys index something, and what they index is not identified.** `REG-NPC-058` grades that
   Unknown in as many words.

Two further things were established here, and both make the fourth step harder rather than easier.
Some speakers carry no picture key at all — their `Flags` name the player rather than a picture, so
their face would be a function of character-generation state this tree does not hold. And the archive
census over both roots finds **no node named for an NPC face**: `graphics.res` has no `npc/` or
`faces/` tree, `scenario.res` holds three registries and the maps, and the `.16a` art that *is* filed
per NPC number lives under the **inn's** tree and covers six of the thirty-six speakers the mission
corpus uses — missing the four most common ones. That last measurement is what rules out the one
guess a builder would otherwise have made.

## What was considered and rejected

**Waiting for the decode.** Everything except the picture is settled, and the pane, the two tests and
the per-part selection are the larger part of the work. Holding all of it for one unresolved hop
would leave the shipped tree drawing a layout the game never draws, for the sake of not drawing an
empty rectangle.

**Guessing the art.** `inn/unit<n>/sprites.16a` is the tempting candidate and it is measurably wrong.
Choosing among the remaining candidates would be deciding what an undecoded byte layout means, which
is the one thing golden rule 4 refuses.

**Reproducing the speech playback.** The pager also composes a `speech\...\....wav` per part. No
`battle` subtree exists in `speech.res` on either root, so it resolves to nothing: there is nothing
to build and nothing to hear.
