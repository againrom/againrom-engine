# 1087 — Mission census

Source: `49f60ffd`, reconciled with master `2fae001a`. Full behavior and gate
evidence remain in `story.md`.

Worktree-built `.local/missionrun-1087.exe`, preserved EN assets,
`-mission 10 -trace -ticks 1` and `-mission 20 -trace -ticks 1`:

| Mission | Baseline unsupported | Candidate unsupported | Script checks/instants/triggers |
|---|---:|---:|---|
| 10 | 0 | 0 | 16/27/12 |
| 20 | 0 | 0 | 14/15/11 |

Baseline: seat `pipeline/milestone-baseline.txt`; neither mission has an
unsupported line. Counts are unchanged. The observable player result is the
installed App Defend route and native continuation described in `story.md`,
not a lower census. No desktop window was driven; no whole-mission win is claimed.
