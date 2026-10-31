package archtest

import (
	"strings"
	"testing"
)

// TestLiveSpellArmSitesAreNone measures the real tree: no production file
// outside the spell-arm table tests an arm's identity.
func TestLiveSpellArmSitesAreNone(t *testing.T) {
	files := liveProductionSources(t)
	if !strings.Contains(files[SpellArmTableFile], "var spellArms = ") {
		t.Fatalf("%s declares no spellArms; the scan reads the wrong file", SpellArmTableFile)
	}
	for _, v := range SpellArmSites(files) {
		t.Error(v)
	}
}

// spellArmFindings is the file:line and shape of each finding.
func spellArmFindings(files map[string]string) []string {
	var out []string
	for _, v := range SpellArmSites(files) {
		shape, _, _ := strings.Cut(v.Reason, ":")
		out = append(out, v.From+" "+shape)
	}
	return out
}

func wantSpellArmFindings(t *testing.T, src string, want ...string) {
	t.Helper()
	wantSpellArmFindingsIn(t, "pkg/sim/a.go", src, want...)
}

func wantSpellArmFindingsIn(t *testing.T, path, src string, want ...string) {
	t.Helper()
	files := map[string]string{
		SpellArmTableFile: "package sim\nconst lastBookSpellID = 28\nconst ArmNone = 0xffff\nconst PrismaticSpellID = 14\nvar spellArms = [...]spellArm{{arm: 1}}\n" +
			"func f(rule SpellRule) bool { return rule.arm() == 14 }\n",
		path: src,
	}
	got := spellArmFindings(files)
	if strings.Join(got, ", ") != strings.Join(want, ", ") {
		t.Fatalf("findings %v, want %v", got, want)
	}
}

func TestSpellArmCompareIsFound(t *testing.T) {
	wantSpellArmFindings(t, `package sim
const teleport = 26
func a(w *World, rule SpellRule, id uint16) bool {
	return rule.arm() == 14 || 26 != w.spellArm(id) || uint32(rule.ArmID()) == teleport || w.SpellArm(id) == uint16(5)
}
`, "pkg/sim/a.go:4 compare", "pkg/sim/a.go:4 compare", "pkg/sim/a.go:4 compare", "pkg/sim/a.go:4 compare")
}

func TestSpellArmVariableCompareIsFound(t *testing.T) {
	wantSpellArmFindings(t, `package sim
func a(w *World, id uint16, n int) bool {
	kind := w.SpellArm(id)
	if n == 3 {
		return kind != ArmNone
	}
	return kind == 6
}
func b(arm uint16) bool { return arm == 3 }
`, "pkg/sim/a.go:5 compare", "pkg/sim/a.go:7 compare", "pkg/sim/a.go:9 compare")
}

// TestNonSpellArmsAreClean: outside the simulation a name arm, a table
// named ...Arms and an Arm field of a value not named for a spell are other
// kinds' arms, not findings.
func TestNonSpellArmsAreClean(t *testing.T) {
	wantSpellArmFindingsIn(t, "pkg/game/b.go", `package game
var eventArms = map[int]bool{}
func b(arm int, resolved mapload.Resolved) bool {
	return arm == 3 || resolved.Arm == mapload.ArmUnits
}
`)
}

func TestSpellArmSwitchIsFound(t *testing.T) {
	wantSpellArmFindings(t, `package sim
func a(rule SpellRule, n int) int {
	switch rule.arm() {
	case 13, 14:
		return 1
	}
	switch n {
	case 13:
		return 2
	case lastBookSpellID:
		return 3
	}
	return 0
}
`, "pkg/sim/a.go:4 switch", "pkg/sim/a.go:4 switch", "pkg/sim/a.go:10 switch")
}

func TestSpellArmLookupIsFound(t *testing.T) {
	wantSpellArmFindings(t, `package sim
func a(w *World, e EntityID, arm uint16) bool {
	return w.HasEffectArm(e, 20) || w.armSpellID(15) == 0 || SpellArmTeleports(26) || w.HasEffectArm(e, arm) ||
		w.autoCastAt(0, 0, 6, true)
}
`, "pkg/sim/a.go:3 lookup", "pkg/sim/a.go:3 lookup", "pkg/sim/a.go:3 lookup", "pkg/sim/a.go:4 lookup")
}

func TestSpellIDConstCompareIsFound(t *testing.T) {
	wantSpellArmFindings(t, `package sim
func a(rule SpellRule, i int) bool {
	return rule.ID == sim.PrismaticSpellID || i+1 != lastBookSpellID || rule.ID > lastBookSpellID
}
`, "pkg/sim/a.go:3 id-const", "pkg/sim/a.go:3 id-const", "pkg/sim/a.go:3 id-const")
}

func TestSpellArmConstNameIsFound(t *testing.T) {
	wantSpellArmFindings(t, `package sim
const healSpellID = 6
func a() int {
	const teleportArm = 26
	return teleportArm
}
`, "pkg/sim/a.go:2 const-name", "pkg/sim/a.go:4 const-name")
}

func TestSpellArmTableIsFound(t *testing.T) {
	wantSpellArmFindings(t, `package sim
var magnitudeArms = []bool{true}
var level = 3
var pointMagnitudes = [...]func() int{5: half, 10: half}
var order = [...]int{1, 2}
var dirs = [8]struct{ x int }{0: {1}, 1: {2}}
`, "pkg/sim/a.go:2 arm-table", "pkg/sim/a.go:4 arm-table")
}

func TestSpellArmFieldMatchIsFound(t *testing.T) {
	wantSpellArmFindings(t, `package game
func a(spell sim.WeaponSpell, rule sim.SpellRule) bool {
	switch spell.Arm {
	case 20:
		return true
	}
	return rule.Arm == sim.ArmNone
}
`, "pkg/sim/a.go:4 switch", "pkg/sim/a.go:7 field-match")
}

// TestSpellArmAllowListAdmitsItsCount: an entry admits its count of one
// file's shape; one more site, or one fewer, is a finding.
func TestSpellArmAllowListAdmitsItsCount(t *testing.T) {
	src := "package sim\nfunc a(rule SpellRule) bool { return rule.arm() == 14 || rule.arm() == 26 }\n"
	files := map[string]string{"pkg/sim/a.go": src}
	if vs := spellArmSitesAllowing(files, map[string]int{"pkg/sim/a.go compare": 2}); len(vs) != 0 {
		t.Fatalf("admitted sites reported: %v", vs)
	}
	if vs := spellArmSitesAllowing(files, map[string]int{"pkg/sim/a.go compare": 1}); len(vs) != 2 {
		t.Fatalf("an excess site: %v, want both sites", vs)
	}
	if vs := spellArmSitesAllowing(files, map[string]int{"pkg/sim/a.go compare": 3}); len(vs) != 1 {
		t.Fatalf("a fallen count: %v, want one finding", vs)
	}
	if a, ok := SpellArmAllowList["pkg/sim/creaturecast.go arm-table"]; !ok || a.Count != 1 || len(SpellArmAllowList) != 1 {
		t.Fatalf("allow list %v, want the second game's creature aim table alone", SpellArmAllowList)
	}
}

func TestSpellRowIDCompareIsFound(t *testing.T) {
	wantSpellArmFindingsIn(t, "pkg/game/a.go", `package game
const fireBall = 2
func a(rule sim.SpellRule, spell sim.SpellRule, other sim.Entity) bool {
	return rule.ID == 24 || 14 != spell.ID || rule.ID == fireBall || rule.ID == 0 || other.ID == 5
}
`, "pkg/game/a.go:4 id-match", "pkg/game/a.go:4 id-match", "pkg/game/a.go:4 id-match")
}

func TestSpellRowIDListIsFound(t *testing.T) {
	wantSpellArmFindingsIn(t, "pkg/game/a.go", `package game
func a(rule sim.SpellRule, ids []uint16) bool {
	return slices.Contains([]uint16{24, 7, 28}, rule.ID) || slices.Contains(ids, rule.ID) ||
		slices.Contains([]uint16{0}, rule.ID)
}
`, "pkg/game/a.go:3 id-match")
}

func TestBookSpellLiteralIDIsFound(t *testing.T) {
	wantSpellArmFindings(t, `package sim
func (w *World) a(e Entity, i int, t EntityID, id uint32) bool {
	_, ok := w.bookSpell(e, 6)
	return ok && w.bookSpellRefusal(i, t, 6, false, false) == "" && w.beginBookSpellOnce(i, t, 6) &&
		w.beginBookSpellOnce(i, t, id)
}
`, "pkg/sim/a.go:3 book-id", "pkg/sim/a.go:4 book-id", "pkg/sim/a.go:4 book-id")
}

func TestSpellIDTableIsFound(t *testing.T) {
	src := `package game
var captions = []caption{{value: byID([]uint16{5, 16}, func(rule sim.SpellRule) bool { return true })}}
var marks = map[uint16]mark{5: {1, 2}}
var kinds = []caption{{kind: sim.SpellCaptionSpeed, value: func(rule sim.SpellRule) bool { return true }}}
var keyed = map[uint16]func(sim.SpellRule) int{12: light}
`
	wantSpellArmFindingsIn(t, "pkg/game/a.go", src, "pkg/game/a.go:2 id-table", "pkg/game/a.go:5 id-table")
	wantSpellArmFindingsIn(t, "pkg/game/sub/a.go", src)
}

// TestSpellArmTableFileAndCapabilitiesAreClean: the table file is exempt and
// a consumer asking a capability is no finding.
func TestSpellArmTableFileAndCapabilitiesAreClean(t *testing.T) {
	wantSpellArmFindings(t, `package sim
func a(w *World, rule SpellRule, id uint16) bool {
	arm := spellArmOf(rule.arm())
	return arm.prismatic || SpellArmTeleports(w.SpellArm(id)) || w.HasEffectArm(0, w.stoneCurseSpell())
}
`)
}
