package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/sim"
)

// The bodies of a game's script rules. compileScriptFrom reads each rule only
// through its index here, so a rule's two bodies never sit as branches on a
// profile flag.

// emptyBodies reports whether a script compiles to no script at all.
var emptyBodies = [...]func(src alm.Script) bool{
	scriptprofile.EmptyCompiles: func(alm.Script) bool { return false },
	scriptprofile.EmptyNil:      func(src alm.Script) bool { return src.Empty() },
}

// dropBody is how a drop action is kept: whether its parameters bind, and the
// record a kept drop writes.
type dropBody struct {
	bind   bool
	record func(rep *ScriptReport, n alm.ScriptNode, args [10]int32)
}

var dropBodies = [...]dropBody{
	scriptprofile.DropCells: {bind: true, record: func(rep *ScriptReport, _ alm.ScriptNode, args [10]int32) {
		rep.DropCells = append(rep.DropCells, ScriptCell{X: args[0], Y: args[1]})
	}},
	scriptprofile.DropPacked: {record: func(rep *ScriptReport, n alm.ScriptNode, _ [10]int32) {
		rep.PackedCoordinates = append(rep.PackedCoordinates, uint16(uint8(n.Value[0]))|uint16(uint8(n.Value[1]))<<8)
	}},
}

// omittedBodies records an omitted action: a build-time node, or an unbuilt
// one that a trigger's roster check skips.
var omittedBodies = [...]func(buildTime, unbuilt map[uint32]bool, id uint32){
	scriptprofile.OmittedUnbuilt:   func(_, unbuilt map[uint32]bool, id uint32) { unbuilt[id] = true },
	scriptprofile.OmittedBuildTime: func(buildTime, _ map[uint32]bool, id uint32) { buildTime[id] = true },
}

// constBody is how a node binds: its arguments and references, and whether an
// unbuilt reference omits it.
type constBody struct {
	bind      func(n alm.ScriptNode, refs ScriptRefs, rep *ScriptReport, p scriptprofile.Profile) ([10]int32, bound)
	omittable bool
}

// constBodies binds a constant node only; every other node binds through
// constBodies[scriptprofile.ConstBound].
var constBodies = [...]constBody{
	scriptprofile.ConstBound: {omittable: true,
		bind: func(n alm.ScriptNode, refs ScriptRefs, rep *ScriptReport, p scriptprofile.Profile) ([10]int32, bound) {
			return bindScriptParams(n, refs, rep, true, p)
		}},
	scriptprofile.ConstValue: {
		bind: func(n alm.ScriptNode, _ ScriptRefs, _ *ScriptReport, _ scriptprofile.Profile) ([10]int32, bound) {
			var args [10]int32
			args[0] = int32(n.Value[0])
			return args, bound{}
		}},
}

// omitBodies reports whether a node's references omit it.
var omitBodies = [...]func(b bound, refs ScriptRefs) bool{
	scriptprofile.OmitByRole:       func(b bound, refs ScriptRefs) bool { return refs.Roster && b.roleFailed },
	scriptprofile.OmitByAnyFailure: func(b bound, _ ScriptRefs) bool { return b.failed },
}

// triggerOmitBodies reports whether a trigger is omitted for its checks.
var triggerOmitBodies = [...]func(t alm.ScriptTrigger, omittedChecks map[uint32]bool) bool{
	scriptprofile.TriggersKept: func(alm.ScriptTrigger, map[uint32]bool) bool { return false },
	scriptprofile.TriggersOmitOnCheck: func(t alm.ScriptTrigger, omittedChecks map[uint32]bool) bool {
		omitted := false
		for k := range t.Pairs() {
			if left, right, _ := t.Pair(k); omittedChecks[left] || omittedChecks[right] {
				omitted = true
			}
		}
		return omitted
	},
}

// unitBodies resolves a unit value.
var unitBodies = [...]func(v uint32, refs ScriptRefs) (sim.EntityID, bool){
	scriptprofile.UnitDirect: resolveUnit,
	scriptprofile.UnitByRole: resolveROM2Unit,
}

// lateBodies reports whether a value is a late reference.
var lateBodies = [...]func(p scriptprofile.Profile, t, v uint32) bool{
	scriptprofile.LateKept: func(scriptprofile.Profile, uint32, uint32) bool { return false },
	scriptprofile.LateReferences: func(p scriptprofile.Profile, t, v uint32) bool {
		return (t == typeGroup || t == typePlayer || t == typeUnit) && v >= p.LateLow && v <= p.LateHigh
	},
}
