package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckRewardPolicyNamesThePlantedRead(t *testing.T) {
	const predicate = "package sim\nfunc (w *World) paysNoReward(e Entity) bool { return w.game.suppressedRewards && e.SuppressRewards }\n"
	const copied = "func (w *World) copied(e Entity) bool { return w.game.suppressedRewards && e.SuppressRewards }\n"
	cases := []struct {
		name, file, src, want string
	}{
		{"caller", "pkg/sim/skill.go", "package sim\n" + copied, "pkg/sim/skill.go:2"},
		{"same file", rewardPolicyFile, predicate + copied, rewardPolicyFile + ":3"},
		{"profile alias", "pkg/sim/step.go", "package sim\nfunc f(p gameProfile) bool { return p.suppressedRewards }\n", "pkg/sim/step.go:2"},
		{"test source", "pkg/sim/skill_test.go", "package sim\n" + copied, ""},
		{"prose", "pkg/sim/skill.go", "package sim\n// w.game.suppressedRewards\nvar text = `w.game.suppressedRewards`\n", ""},
		{"caller uses predicate", "pkg/sim/skill.go", "package sim\nfunc f(w *World, e Entity) bool { return w.paysNoReward(e) }\n", ""},
		{"missing read", rewardPolicyFile, "package sim\n", "exactly one"},
		{"extra predicate read", rewardPolicyFile, strings.Replace(predicate, "e.SuppressRewards", "w.game.suppressedRewards", 1), "found 2"},
		{"parse failure", "pkg/sim/skill.go", "package sim\nfunc", "source does not parse"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{rewardPolicyFile: predicate}
			files[tc.file] = tc.src
			vs := CheckRewardPolicy(files)
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("violations = %v, want none", vs)
				}
			} else if len(vs) != 1 || !strings.Contains(vs[0].String(), tc.want) {
				t.Fatalf("violations = %v, want one naming %q", vs, tc.want)
			}
		})
	}
	if vs := CheckRewardPolicy(nil); len(vs) != 1 {
		t.Fatalf("empty scan = %v, want one violation", vs)
	}
}

func TestRewardPolicyHasOnePredicate(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadSimSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckRewardPolicy(files) {
		t.Error(v)
	}
}
