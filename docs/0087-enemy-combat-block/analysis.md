# Analysis — the people on the map fight for nothing

## The report, and where it actually lands

The owner reports that enemies deal `0-0` and cannot hurt him. That string is `pkg/ui/panel.go`'s
damage field: `fmt.Sprintf("%d-%d", DamageBase, DamageBase+DamageSpread)`. So the report is about
entities whose damage pair is `(0, 0)`, and the question is which placements those are.

A handed hypothesis said the loader spawns a placed actor without applying its `Data.bin` combat
columns, and that `0-0` then follows from the unarmed law below Body 32. It named a `Man_Club` row
carrying Body 19. Both halves were checked before anything was built on them.

**The `Man_Club` row is not on this map.** It is index 10 of the *Humans* collection — a row index,
not a mission number. Mission 1 is `scenario.res:10.alm`.

**The loader is not uniformly empty.** `classdump -databin` against the lawful EN root, on `10.alm`
at difficulty 2, prints one row per placement off the built world:

```
      3 0x0045/0x0001       8      4     40       50       2        1          7   false  "Ghost"
      6 0x004a/0x0001       8      4     40        0       0        3          3   false  "Squirrel"
     11 0x0049/0x0001       6      2     30       10       0        3          2    true  "Bee"
      0 0x000a/0x0000       8      4      0        0       0        0          0   false  none - the server-id arm reached no unit definition
  arm npc taken 2   arm server-id taken 14   arm humans taken 0   arm units taken 19
```

Nineteen placements — every creature — already carry their own numbers. **Sixteen carry nothing:**
the fourteen that resolve through the definition-id arm and the two through the NPC arm. Those are
the *people*, and they include placements 0 and 1, humans entry **203**, which is `M10_Brigands` —
the two the win chain's second trigger requires dead.

So the hypothesis' mechanism is right and its family was one out. The defect is the **humans band**,
not the spawner as a whole, and it is not an arithmetic bug: `definitionFor` returns no definition at
all off any of the three humans rungs, and `FromALMWith` then substitutes `data.UnitDefaults()`,
which sets no damage, no to-hit, no defence and no absorption. Health is `SpawnHP`, our provisional
100, for all sixteen.

## Why the fix is not "read the columns"

The `Humans` collection **has no damage column** — no `physicalMin`, no `physicalMax`, no
`attackKind`, no `absorbtion`, no protection and no resistance. Twenty-three slots, and five column
families that exist only in `Units` are absent from all of them. A loader that read this collection
the way the `Units` loader reads its own would still ship a monster at `0-0`, and would look
finished.

A human's damage comes from the **derived-stat graph** instead: `ftol(1.1^Body / 20)` into both
halves of the pair, plus whatever an equipped weapon adds. That is the same recompute the party's own
hero already goes through, and this tree already has all of it — `data.Hero.Derive` and
`data.ResolveWeapon`, built for 0078 and 0082. Nothing new has to be decoded to arm a brigand; the
arithmetic is in the tree and the humans band simply never reaches it.

At Body 5, `ftol(1.1^5 / 20)` is 0. So `M10_Brigands` is predicted to swing for nothing **bare** —
which is the hypothesis' own mechanism arriving at the right row by a different road — and the whole
of the difference is his `Wood Club`.

## What was looked at and rejected

- **Reusing 0082's character generator.** A non-hero unit's `vt+0x50` derives nothing; the *Units*
  template's numbers are the stats. That is the arm 0067 already built, and it is not this one. The
  generator's point-buy, its authored spread and its skill choice are a screen's rules and belong to
  no map placement.
- **A single code path for both collections.** Two collections, two streamers, two field maps, two
  sources of damage. Merging them is the mistake this story most invites.
- **The `Units` arm's `EquipItem` column.** Twenty-six of fifty-six classes carry one; none of the
  three on this map does, so it moves nothing here and is a story of its own.
- **Applying the difficulty setting to a human.** The spawner's humans arm jumps over the whole
  three-way block. Never adjusted — so this stays as it is by construction, not by omission.

## Two things measured on the way that are not this story's

- The **NPC arm reaches no entry** in `classdump`'s report. That is the tool's own table, which
  carries no NPC registry; the front end's does. Not a defect, and not evidence either way.
- `resolveBlow` subtracts absorption unconditionally. The auto-hit mark is decoded to skip it, and
  eleven of this map's twenty-one hostiles carry that mark. It changes nothing while the party's own
  absorption is zero, which it is, so it is named in the spec's out-of-scope list rather than fixed
  here.
