# Provenance — multi-archive resolution by identity prefix

Pinned at research `e7602bb`, frozen for the story. `claims/retracted.md` was read first: it carries
`RES-PATH-025`'s lower-casing clause, retracted at High. Only that row's separator half is cited.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| FR-4, P-4, P-5 — an archive answers only an address whose leading segment is its identity, that segment consumed | `RES-IDENT-034` | **High** — the compare, its skip conditions and the identity's derivation each read at instruction level with anchors; the path-literal census (335 `graphics\`, 95 `sfx\`, 44 `main\`, …) corroborates and is not the basis |
| FR-1 — identity = the basename, cut before the first `.`, first 15 bytes | `RES-IDENT-034` | High — `@L07659..@L07660` the basename walk, `@L07661` the cut, `@L07662` the clamp |
| FR-5 — among archives sharing an identity, list order decides and the first match answers | `RES-ORDER-033` | **High** — read from a raw listing, with five named rivals each killed separately |
| FR-3 — the whole address is folded, identity segment included, ASCII `A`–`Z` alone | `RES-CASE-036` | **High** — the gate's value read from the image with one reference image-wide, the transform read at instruction level, and 95 shipped literals showing the fold is *required*, not merely present |
| FR-3 — `\` and `/` interchangeable at every position, neither rewritten | `RES-PATH-025`, separator half | High — tested symmetrically at every position |
| FR-8 — a loose-file tier on disk after the archives, first readable file answering, addressed by the whole folded address | `RES-ORDER-033`, `RES-DIR-037` | High for the code — both tiers and the tier-2 path build read at instruction level |

`RES-COLL-038` **corroborates and does not back**: 0 identity and 0 cross-archive path collisions
over 8 live archives and 3980 EN / 3976 RU paths — **Medium, capped there deliberately**, since a
census shows no collision *occurs*, never that one is *impossible*. What forbids them is
`RES-IDENT-034`. So FR-5 cites `RES-ORDER-033` alone, and this row is why FR-5 reads as a corner case.

`RES-IDENT-034` and `RES-COLL-038` are **inherited** assignments, argued **into** this tier by
`docs/0001-res-archive/provenance.md` under Divergence: a single-container reader cannot know the
identity of the file it was opened from, and holds no set for a dispatch rule or a census to be about.

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **A failed archive open fails the whole open** (FR-2, P-6) | The original does the opposite: `RES-SET-032` (High) has a failed open throw, swallowed by named catch funclets so the launch resumes and a missing archive is merely absent — which, with `RES-DIR-037`'s first-directory-only rule, is why 3 of 10 registered names never open. Ours refuses an install the original runs degraded on |
| **The caller supplies both lists; nothing is discovered** (FR-1, C-4) | `RES-SET-032`: the set is ten literal names registered unconditionally at startup plus `World.res` on demand, each opened lower-cased under the first registered directory. `RES-DIR-037`: that directory is the working directory captured before `main`, with three more appended from the registry and the environment. A search of that shape is not testable in a library, and the asset root already arrives from a flag |
| **Both escapes from disjointness are refused** — a leading separator on the address, an empty derived identity on an archive (FR-1, FR-3, AC-6, AC-9) | `RES-IDENT-034` skips the identity compare in both cases (`@L07657`, `@L07658`), so each turns list order into a real priority. No shipped name or literal is shaped either way, and research carries the first as its own open item — so building them would exercise a mechanism only our own tests could reach |
| **The leading segment must equal an identity, not merely prefix it** (FR-4) | `RES-IDENT-034` has the compare stop both when the identity is consumed and when the path reaches a separator, leaving equality and prefix acceptance undistinguished. Equality is the narrower rule; the ambiguity is filed under *Open* |
| **Enumeration ascends by folded address, archive tier only** (FR-9) | Nothing in the original enumerates: the manager exposes no listing and `res.Entries()` is registry order |
| **The loose tier looks up the folded address** (C-3) | `RES-CASE-036` folds every lookup path, and `RES-COLL-038` notes the RU containers are upper-cased on disk while the engine opens the lower-cased name, Win32 resolving that for it. A host that does not fold has no such backstop |
| **A listed directory is not validated at open** (FR-2) | Undecoded: the original's directory-existence check is not on this path. A missing directory simply never answers |
| **A read yields bytes the caller owns** (FR-11) | Nothing decoded bears; `res.ReadFile` already copies, and the rule extends that to the loose tier |
| **The callers migrate in this same story** (FR-12) | No claim bears: which of our packages resolves what is our own structure |

## Open — assigned no meaning by the spec

- **Whether a leading segment that is a strict prefix of an identity resolves.** No shipped literal
  separates it — a question for research, not a default to pick.
- **Whether any runtime-constructed lookup path begins with a separator.** Research's own open item,
  and the only route by which archive order could become an observable priority in the original.
- **What a launch's directory list holds.** `RES-DIR-037` is **Medium** there — the environment's,
  not the image's. Nothing depends on it: the list is an argument.

## Removed from the baseline and why

- **The `patch.res` override, and "the last argument has highest priority".** `RES-IDENT-034` makes
  the archives disjoint namespaces, so no override is expressible; `RES-COLL-038` finds `patch.res`
  holds one entry, `patch\patch.txt`, and registers after `graphics` and `main` in any case.
- **The override mechanism itself — a genuine absence, not something declined.** `RES-MASK-035`
  (High) reads `update.lst` end to end: the engine's only config-driven input here and its only
  override, it **masks rather than adds** — bit 29 on the matched node makes the resolver return a
  stop scalar and fall through to disk — and **neither shipped install contains one**. The engine has
  an override; this install does not use it. That is why the loose tier is built and the mask is not.
- **Close-all with joined errors, and the no-leaked-handle clause.** `res.Archive` has no `Close` and
  holds no host handle past `res.Open`, which is `os.ReadFile` then `OpenBytes`: both describe an
  operation this tier cannot have and failures that cannot occur. The atomicity half is kept (FR-2).
- **A non-not-found archive failure that must not fall through**, as an *archive-tier* rule:
  `res.ReadFile` fails only with `fs.ErrNotExist`. FR-7 keeps the rule for the loose tier, which can.
