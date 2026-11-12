package game

import (
	"testing"

	"againrom/pkg/sim"
)

// unlockCheats gives the Player the chat unlock privilege.
func unlockCheats(mw *mapWorld) { mw.cheats.privilege[sim.SelfSlot] = 255 }

func TestCheatsUnlockedFollowsThePlayersPrivilege(t *testing.T) {
	var none *FrontEnd
	if none.cheatsUnlocked() || (&FrontEnd{}).cheatsUnlocked() {
		t.Fatal("a session with no mission reports the cheats unlocked")
	}
	f := &FrontEnd{}
	f.live = &mapWorld{}
	if f.cheatsUnlocked() {
		t.Fatal("a fresh mission reports the cheats unlocked")
	}
	for _, p := range []byte{50, 51, 255} {
		f.live.cheats.privilege[sim.SelfSlot] = p
		if got, want := f.cheatsUnlocked(), p > 50; got != want {
			t.Fatalf("privilege %d: unlocked %v, want %v", p, got, want)
		}
	}
}
