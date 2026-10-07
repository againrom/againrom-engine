package sim

import "testing"

func TestPickupPublicationClearsOnlyGatedHolders(t *testing.T) {
	for _, c := range []struct {
		name   string
		owner  uint32
		typeID int32
		want   uint32
	}{
		{"recipient Human", SelfSlot, HumanTypeID, 0},
		{"recipient non-band", SelfSlot, 0, 1},
		{"other owner Human", SelfSlot + 1, HumanTypeID, 1},
	} {
		w := itemOperationsWorld(t)
		w.carried[0] = nil
		w.recomputeLoad(0)
		w.entities[0].Owner, w.entities[0].TypeID = c.owner, c.typeID
		picked := w.savedObjects.NextID
		if err := w.TakeSack(7, 10, 20); err != nil {
			t.Fatal(c.name, err)
		}
		if got := w.savedObjects.item(picked).Token.T08; got != c.want {
			t.Fatalf("%s: picked Item flag %d, want %d", c.name, got, c.want)
		}
		if n := w.PublishSessionEntry(SelfSlot); c.want == 1 && n != 0 || w.savedObjects.item(picked).Token.T08 != c.want {
			t.Fatalf("%s: session entry cleared %d outside its gates", c.name, n)
		}
	}
}
