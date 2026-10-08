package sim

import "testing"

func TestGhostTemplateNativeBasisValidationAndLegacyAbsence(t *testing.T) {
	caster, corpse := spEnt(1, 1, 1), spEnt(2, 2, 1)
	corpse.NativeBasis = (NativeActorBasis{}).WithBody(503).WithModifier([64]byte{18: 199})
	legacy := hlGhostWorld(t, 47, nil, hlGhostTemplate(), caster, corpse)
	raised, ok := legacy.raisedGhost(0, 1, 3)
	if !ok || raised.NativeBasis.HasValues() {
		t.Fatal("legacy template invented or copied corpse native history", raised.NativeBasis)
	}
	bad := []NativeActorBasis{
		{BaseKnown: 1},
		{Base: [24]byte{0: 1}},
		{ModifierKnown: 1},
		{Modifier: [64]byte{0: 1}},
		{BodyKnown: true},
		{Body: 1},
		{BasePresent: true, BaseKnown: 1 << 24},
	}
	for _, basis := range bad {
		template := hlGhostTemplate()
		template.NativeBasis = basis
		if _, err := NewSummoningWorld(47, Bounds{Width: 16, Height: 16}, ModeCanonical,
			Terrain{}, []Entity{caster, corpse}, nil, Relations{}, nil, nil, nil, template); err == nil {
			t.Fatal("constructor accepted invalid Ghost history", basis)
		}
		before := legacy.Hash()
		policy := legacy.CurrentPolicy()
		policy.Ghost.NativeBasis = basis
		if err := legacy.RestoreCurrentContinuation(&policy, nil, legacy.Actions(), nil); err == nil || legacy.Hash() != before {
			t.Fatal("invalid current Ghost history accepted or changed World", basis, err)
		}
		if legacy.Ghost().NativeBasis.HasValues() {
			t.Fatal("policy mutation aliased live template")
		}
	}
}
