package game

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// hireArchiveCounts are the per-voice file counts of the sound archive's
// characters/ keys (R2-ASSET-185), in the order select, attack, move, swarm,
// retreat and defend (one count), pickup. Only the second root holds them.
var hireArchiveCounts = map[string][7]int{
	"xbow":   {2, 1, 3, 2, 1, 1, 0},
	"knight": {3, 1, 3, 2, 1, 1, 0},
	"druid":  {2, 1, 2, 3, 1, 1, 0},
	"troll":  {3, 1, 1, 3, 1, 1, 0},
	"orc":    {3, 1, 2, 3, 1, 1, 0},
	"mage":   {3, 2, 3, 3, 1, 1, 0},
}

var hireGestureNames = [7]string{"select", "attack", "move", "swarm", "retreat", "defend", "pickup"}

// hireKeys lists the characters/ keys of one root: every voice has easy, hard
// and dead in the second root and in the first only the six hire voices do
// (R2-ASSET-185); the second root alone holds the hire gesture files.
func hireKeys(ru bool) []string {
	var keys []string
	voices := []string{"xbow", "knight", "druid", "troll", "orc", "mage"}
	if ru {
		voices = append(voices, "m_fighter", "m_mage", "m_peasant", "f_fighter", "f_mage", "f_peasant")
	}
	for _, v := range voices {
		for _, cry := range []string{"easy", "hard", "dead"} {
			keys = append(keys, "characters/"+v+"/"+cry+".wav")
		}
		if ru {
			for i, g := range hireGestureNames {
				for n := 1; n <= hireArchiveCounts[v][i]; n++ {
					keys = append(keys, fmt.Sprintf("characters/%s/%s/%d.wav", v, g, n))
				}
			}
		}
	}
	return keys
}

// hireSubject is hire id's actor: the person its class draws, or a creature.
func hireSubject(id uint8) voicePerson {
	switch id {
	case 1:
		return voicePerson{typeID: 14, dir: data.FigureDirManFighter, weapon: true, owner: sim.SelfSlot, hp: 10}
	case 6:
		return voicePerson{typeID: 0x17, dir: data.FigureDirWomanMage, owner: sim.SelfSlot, hp: 10}
	}
	return creature
}

// hireWantVoice is the voice directory a hire's word selects (R2-ENGINE-718,
// R2-ENGINE-914): the six hire voices, then the catapult's generic voice.
var hireWantVoice = [8]string{"", "xbow", "knight", "druid", "troll", "orc", "mage", "m_peasant"}

// TestSecondGameHireRepliesAndCriesFollowTheArchive checks, for hires 1 to 7
// of each root's key set, the recording each gesture and each hurt event
// requests, against the composed table of R2-ENGINE-914: the first root's
// hires answer only with easy, hard and dead, the second's also with every
// gesture of their own voice but pickup, and the catapult, in the second
// root, with the cries of its generic voice only.
func TestSecondGameHireRepliesAndCriesFollowTheArchive(t *testing.T) {
	for _, root := range []struct {
		name string
		ru   bool
	}{{"first root", false}, {"second root", true}} {
		keys := hireKeys(root.ru)
		for id := uint8(1); id <= 7; id++ {
			voice := hireWantVoice[id]
			for _, off := range []bool{false, true} {
				name := fmt.Sprintf("%s hire %d (%s) acknowledgment off=%v", root.name, id, voice, off)
				mw := voiceFixture(t, hireSubject(id))
				secondVoiceMission(mw, []mapload.PartyMember{voiceHireMember(id, id != 1 && id != 6)}, nil)
				f, voices := voiceFront(mw, secondVoiceBank(keys...))
				f.speech.acknowledgmentsOff = off
				script := &scriptedDraw{}
				command, selection := f.runtimeAudio().unitReplies(mw, script.draw)
				now := time.Unix(100, 0)
				code := func(path string) int16 { return int16(slices.Index(keys, path) + 1) }
				for g, gesture := range hireGestureNames {
					now = now.Add(time.Minute)
					mark := len(voices.samples)
					// The speaker draw 0, then the last file of the set.
					script.set(0, 32767)
					switch g {
					case 0:
						selection([]uint32{1}, now)
					default:
						command([]ui.VoiceGesture{0, ui.VoiceAttack, ui.VoiceMove, ui.VoiceSwarm, ui.VoiceRetreat, ui.VoiceDefend, ui.VoicePickup}[g], []uint32{1}, now)
					}
					var want []int16
					wantDraws := 0
					if !off {
						wantDraws = 1
						n := 0
						if root.ru && id <= 6 {
							n = hireArchiveCounts[voice][g]
						}
						switch {
						case n == 0:
						case g <= 3:
							want = []int16{code(fmt.Sprintf("characters/%s/%s/%d.wav", voice, gesture, n))}
							wantDraws = 2
						default:
							want = []int16{code(fmt.Sprintf("characters/%s/%s/1.wav", voice, gesture))}
						}
					}
					if got := heard(voices, mark); !slices.Equal(got, want) || script.used != wantDraws {
						t.Errorf("%s: %s heard %v after %d draws, want %v after %d", name, gesture, got, script.used, want, wantDraws)
					}
				}
				// The cries take the voice bank the wound sender names and
				// follow no setting.
				e, _ := mw.world.Entity(1)
				bank := mw.voiceBank(e)
				if bank != "characters/"+voice {
					t.Errorf("%s: wound voice %q, want characters/%s", name, bank, voice)
					continue
				}
				leaves := *mw.unitVoiceRules().wounds
				for k, cry := range map[int]string{1: "easy", 2: "hard", 3: "dead"} {
					if leaves[k] != cry {
						t.Errorf("%s: hurt event %d plays %q, want %q", name, k, leaves[k], cry)
						continue
					}
					_, ok := f.SoundBank.VoiceSample(bank + "/" + leaves[k] + ".wav")
					if want := id <= 6 || root.ru; ok != want {
						t.Errorf("%s: cry %s found=%v, want %v", name, cry, ok, want)
					}
				}
			}
		}
	}
}
