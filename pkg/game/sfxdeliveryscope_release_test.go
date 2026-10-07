package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type deliveryFrozenCut struct {
	Name     string
	World    uint64
	Tick     uint64
	SAVHash  string
	SAVBytes int
	raw      []byte
}

func deliveryScopeCut(w *deliveryWitness, name string) deliveryFrozenCut {
	w.t.Helper()
	snapshot, _, err := w.front.Snapshot(true)
	w.check(err)
	raw, err := w.front.ExportCurrentSave(snapshot, "Lifecycle")
	w.check(err)
	world := w.front.live.world
	form, err := world.MarshalBinary()
	w.check(err)
	w.check(os.WriteFile(filepath.Join(w.dir, name+"-current.sav"), raw, 0600))
	w.check(os.WriteFile(filepath.Join(w.dir, name+"-world.bin"), form, 0600))
	return deliveryFrozenCut{name, world.Hash(), uint64(world.Tick()), fmt.Sprintf("%x", sha256.Sum256(raw)), len(raw), raw}
}

func deliveryScopeEqual(t *testing.T, before, after deliveryFrozenCut) {
	t.Helper()
	if before.World != after.World || !bytes.Equal(before.raw, after.raw) {
		t.Fatalf("%s changed frozen World or full current SAV: World %x/%x SAV %s/%s", after.Name, before.World, after.World, before.SAVHash, after.SAVHash)
	}
}

func deliveryScopeJSON(w *deliveryWitness, name string, value any) {
	w.t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	w.check(err)
	w.check(os.WriteFile(filepath.Join(w.dir, name+".json"), raw, 0600))
}

func deliveryScopePresent(snapshot audio.DeliverySnapshot, scope audio.ScopeID) bool {
	for _, id := range snapshot.Scopes {
		if id == scope {
			return true
		}
	}
	return false
}

func deliveryScopeProbe(w *deliveryWitness) (audio.DeliveryReceipt, *deliveryWitnessPlayer, int64) {
	w.t.Helper()
	before := w.owner.BackendState().Created
	w.front.live.view.PlayUISound(ui.UISoundOptionsTest)
	snapshot := w.owner.Service.Snapshot()
	for i := len(snapshot.Receipts) - 1; i >= 0; i-- {
		r := snapshot.Receipts[i]
		if r.Reason == audio.DeliveryAdmitted && r.Source == "fixed-interface" && r.Selector == "registry:100" {
			state := w.owner.BackendState()
			if state.Created != before+1 || r.Scope == 0 || !deliveryScopePresent(snapshot, r.Scope) {
				w.t.Fatal("instrument: Viewer fixed producer did not create a scoped retained buffer", r, state)
			}
			p := w.players[state.Created-1]
			p.position = 17 * time.Millisecond
			for _, ch := range w.owner.Service.Snapshot().Channels {
				if ch.Voice == r.Voice && ch.Playing && ch.Phase > 0 {
					return r, p, ch.Phase
				}
			}
			w.t.Fatal("instrument: retained Viewer buffer has no nonzero playing phase")
		}
	}
	w.t.Fatal("instrument: Viewer fixed producer did not admit registry:100")
	return audio.DeliveryReceipt{}, nil, 0
}

func deliveryScopeAlive(w *deliveryWitness, receipt audio.DeliveryReceipt, player *deliveryWitnessPlayer, phase int64) {
	w.t.Helper()
	snapshot := w.owner.Service.Snapshot()
	if !deliveryScopePresent(snapshot, receipt.Scope) || !player.playing || player.closes != 0 || player.position != 17*time.Millisecond {
		w.t.Fatal("outgoing scope or lowest retained handle changed before commit", receipt, snapshot.Scopes, player)
	}
	for _, ch := range snapshot.Channels {
		if ch.Voice == receipt.Voice && ch.Scope == receipt.Scope && ch.Playing && ch.Phase == phase {
			return
		}
	}
	w.t.Fatal("outgoing duplicate identity, generation or phase changed before commit", receipt, phase)
}

func deliveryScopeChoose(w *deliveryWitness, label string) {
	w.t.Helper()
	w.check(headlessOpenLoad(w.app))
	for i, row := range w.app.HeadlessRows() {
		if row.Choosable && (row.Text == label || strings.HasPrefix(row.Text, label+" - ")) {
			for range i {
				w.check(w.app.HeadlessKey("down"))
			}
			w.check(w.app.HeadlessKey("enter"))
			return
		}
	}
	w.t.Fatalf("instrument: ordinary LOAD did not list %q: %v", label, w.app.HeadlessRows())
}

func deliveryScopeReturn(w *deliveryWitness) {
	w.t.Helper()
	if w.app.Screen() == ui.ScreenLoad {
		w.check(w.app.HeadlessKey("escape"))
	}
	if w.app.Screen() == ui.ScreenGameMenu {
		w.check(w.app.HeadlessGameMenuAction("return"))
	}
	if w.app.Screen() != ui.ScreenMap {
		w.t.Fatal("ordinary LOAD cancellation did not return to its map", w.app.Screen())
	}
}

func deliveryScopeShutdown(w *deliveryWitness) {
	w.t.Helper()
	w.app.StopAudio()
	w.app.StopAudio()
	state := w.owner.BackendState()
	if !w.owner.Service.Snapshot().Closed || len(state.Buffers) != 0 || state.Created != state.Destroyed {
		w.t.Fatal("App shutdown left owned buffers or scopes alive", state)
	}
	for i, player := range w.players {
		if player.closes != 1 || player.playing {
			w.t.Fatal("App shutdown did not close each lowest retained handle exactly once", i, player)
		}
	}
	var handles []struct {
		Buffer   int
		Playing  bool
		Closes   int
		Seeks    int
		Position time.Duration
		Gain     float64
	}
	for i, player := range w.players {
		handles = append(handles, struct {
			Buffer   int
			Playing  bool
			Closes   int
			Seeks    int
			Position time.Duration
			Gain     float64
		}{i + 1, player.playing, player.closes, player.seeks, player.position, player.gain})
	}
	deliveryScopeJSON(w, "terminal-lowest-handles", handles)
	w.receipt("shutdown")
}

func deliveryScopeControlled(w *deliveryWitness) {
	f, app := w.front, w.app
	f.Carried = f.NextParty()
	f.arriveInTown()
	f.Town.announceMission(f.Town.currentMain())
	city, _, err := f.Snapshot(false)
	w.check(err)
	city.MercenaryPool[1], city.MercenaryHired[1] = 13, true
	if city.CampaignState {
		city.Campaign.MercenaryWorking[0], city.Campaign.MercenaryHired[0] = 13, true
	}
	bad, err := f.ExportCurrentSave(city, "Late city")
	w.check(err)
	badFile, err := sav.Open(bad)
	w.check(err)
	badCampaign, ok, err := badFile.Campaign()
	w.check(err)
	if !ok || badFile.Head.Mission != 0 || len(badCampaign.MercenaryWorking) == 0 || len(badCampaign.MercenaryHired) == 0 || badCampaign.MercenaryWorking[0] != 13 || !badCampaign.MercenaryHired[0] {
		w.t.Fatal("instrument: late CITY failure control was not written into SAV")
	}
	badStore := SaveStore{Dir: filepath.Join(w.dir, "late-saves")}
	_, err = badStore.WriteOriginal(f.Archives.Root, bad)
	w.check(err)
	w.check(app.OpenMission(f.NewGameOpener(20, ui.ChargenResult{Name: "Lifecycle", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}})))
	for n := 0; app.HeadlessNoticeOpen() && n < 32; n++ {
		w.check(app.HeadlessKey("enter"))
	}
	releasePauseMission(w.t, f, app)
	w.finishOneShots()
	baseline := deliveryScopeCut(w, "source")
	oldLive := f.live
	oldReceipt, oldPlayer, oldPhase := deliveryScopeProbe(w)
	admitted := w.owner.Service.Snapshot().Counters.Admitted
	w.owner.SetSettings(audio.EffectsChannel, audio.Settings{Master: 37, Muted: true})
	w.owner.SetSettings(audio.SpeechChannel, audio.Settings{Master: 61})
	if w.owner.Service.Snapshot().Counters.Admitted != admitted || oldPlayer.gain != 0 {
		w.t.Fatal("settings re-admitted a buffer or missed the lowest retained gain")
	}
	deliveryScopeAlive(w, oldReceipt, oldPlayer, oldPhase)
	afterSettings := deliveryScopeCut(w, "settings")
	deliveryScopeEqual(w.t, baseline, afterSettings)
	w.receipt("settings")
	store := SaveStore{Dir: filepath.Join(w.dir, "current-saves")}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	w.check(app.HeadlessKey("f2"))
	if app.Screen() != ui.ScreenSave {
		w.t.Fatal("ordinary F2 did not open SAVE", app.Screen())
	}
	w.check(app.HeadlessSaveEdit(store.Dir, "Lifecycle", ui.SaveSAV))
	w.check(app.HeadlessSaveAction("save"))
	if state, up := app.HeadlessSaveState(); up && state.Confirmation {
		w.check(app.HeadlessSaveAction("overwrite"))
	}
	good, err := os.ReadFile(filepath.Join(store.Dir, "Lifecycle.sav"))
	w.check(err)
	if !bytes.Equal(good, baseline.raw) {
		w.t.Fatal("ordinary SAVE did not write the complete current producer bytes")
	}
	for _, storeDir := range []string{store.Dir, badStore.Dir} {
		entries, err := os.ReadDir(storeDir)
		w.check(err)
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sav") {
				w.t.Fatal("ordinary save witness produced a non-SAV output", entry.Name())
			}
		}
	}
	deliveryScopeReturn(w)
	deliveryScopeAlive(w, oldReceipt, oldPlayer, oldPhase)
	afterSave := deliveryScopeCut(w, "ordinary-save")
	deliveryScopeEqual(w.t, baseline, afterSave)
	w.check(headlessOpenLoad(app))
	w.receipt("load-open")
	deliveryScopeReturn(w)
	deliveryScopeAlive(w, oldReceipt, oldPlayer, oldPhase)
	if f.live != oldLive {
		w.t.Fatal("ordinary LOAD cancel replaced the live session")
	}
	afterCancel := deliveryScopeCut(w, "cancel")
	deliveryScopeEqual(w.t, baseline, afterCancel)
	w.receipt("cancel")
	f.ConfigureSaveSeams(app, badStore, OriginalStore{}, nil)
	prior := w.owner.Service.Snapshot()
	deliveryScopeChoose(w, "Late city")
	if app.Screen() != ui.ScreenLoad || !strings.Contains(app.HeadlessMessage(), "over the 12-member roster cap") || f.live != oldLive {
		w.t.Fatal("late CITY draft failure did not preserve live session", app.Screen(), app.HeadlessMessage())
	}
	created, destroyed := map[audio.ScopeID]bool{}, map[audio.ScopeID]bool{}
	for _, r := range w.owner.Service.Snapshot().Receipts {
		if r.Sequence <= prior.Receipts[len(prior.Receipts)-1].Sequence {
			continue
		}
		if r.Reason == audio.DeliveryScopeCreated {
			created[r.Scope] = true
		}
		if r.Reason == audio.DeliveryScopeDestroyed {
			destroyed[r.Scope] = true
		}
	}
	if len(created) == 0 {
		w.t.Fatal("instrument: failure did not reach detached draft scope construction")
	}
	for scope := range created {
		if scope == oldReceipt.Scope || !destroyed[scope] || deliveryScopePresent(w.owner.Service.Snapshot(), scope) {
			w.t.Fatal("failed draft scope survived or disposed the outgoing scope", scope, created, destroyed)
		}
	}
	deliveryScopeAlive(w, oldReceipt, oldPlayer, oldPhase)
	w.receipt("late-failure")
	deliveryScopeJSON(w, "late-failure-control", map[string]any{"message": app.HeadlessMessage(), "createdScopes": created, "destroyedScopes": destroyed, "oldScope": oldReceipt.Scope})
	deliveryScopeReturn(w)
	afterFailure := deliveryScopeCut(w, "late-failure")
	deliveryScopeEqual(w.t, baseline, afterFailure)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	deliveryScopeChoose(w, "Lifecycle")
	if app.Screen() != ui.ScreenMap || f.live == oldLive || oldPlayer.closes != 1 || oldPlayer.playing || deliveryScopePresent(w.owner.Service.Snapshot(), oldReceipt.Scope) {
		w.t.Fatal("successful ordinary LOAD did not commit and dispose its outgoing scope", app.Screen(), app.HeadlessMessage(), oldReceipt.Scope, oldPlayer)
	}
	w.finishOneShots()
	committed := deliveryScopeCut(w, "committed")
	newReceipt, newPlayer, newPhase := deliveryScopeProbe(w)
	if newReceipt.Scope == oldReceipt.Scope || newReceipt.Voice == oldReceipt.Voice {
		w.t.Fatal("committed map reused outgoing scope or stale duplicate generation", oldReceipt, newReceipt)
	}
	deliveryScopeAlive(w, newReceipt, newPlayer, newPhase)
	afterCommitDelivery := deliveryScopeCut(w, "committed-delivery")
	deliveryScopeEqual(w.t, committed, afterCommitDelivery)
	w.receipt("committed")
	deliveryScopeShutdown(w)
	afterShutdown := deliveryScopeCut(w, "shutdown")
	deliveryScopeEqual(w.t, committed, afterShutdown)
	deliveryScopeJSON(w, "frozen-state", []deliveryFrozenCut{baseline, afterSettings, afterSave, afterCancel, afterFailure, committed, afterCommitDelivery, afterShutdown})
	coldDir := filepath.Join(w.dir, "cold")
	w.check(os.MkdirAll(coldDir, 0700))
	cold := newDeliveryWitness(w.t, coldDir, true)
	cold.front.ConfigureSaveSeams(cold.app, store, OriginalStore{}, nil)
	deliveryScopeChoose(cold, "Lifecycle")
	if cold.app.Screen() != ui.ScreenMap || cold.front.live == nil || !cold.front.live.mission.resumed {
		w.t.Fatal("cold ordinary SAV LOAD did not reach its resumed map", cold.app.Screen(), cold.app.HeadlessMessage())
	}
	coldBefore := deliveryScopeCut(cold, "cold-load")
	capacityChanges := map[sim.EntityID][2]int32{}
	for _, before := range oldLive.world.Entities() {
		if after, found := liveEntity(cold.front, before.ID); found && before.Capacity != after.Capacity {
			capacityChanges[before.ID] = [2]int32{before.Capacity, after.Capacity}
		}
	}
	deliveryScopeJSON(cold, "cold-projection", map[string]any{"sourceWorld": baseline.World, "coldWorld": coldBefore.World, "capacityChanges": capacityChanges, "qualification": "DIV-1675: a Ghost Unit capacity of 0 is written and cold-restored as 300; full cold World hash parity is not claimed"})
	cold.receipt("cold-load")
	coldReceipt, coldPlayer, coldPhase := deliveryScopeProbe(cold)
	cold.owner.SetSettings(audio.EffectsChannel, audio.Settings{Master: 43, Muted: true})
	deliveryScopeAlive(cold, coldReceipt, coldPlayer, coldPhase)
	coldAfter := deliveryScopeCut(cold, "cold-delivery-settings")
	deliveryScopeEqual(w.t, coldBefore, coldAfter)
	if !cold.front.live.view.SaveApplication().PlayerPaused {
		w.t.Fatal("cold ordinary LOAD lost the persisted pause")
	}
	w.check(cold.app.HeadlessKey("0"))
	for tries := 0; uint64(cold.front.live.world.Tick()) == coldBefore.Tick && tries < 8; tries++ {
		w.check(cold.app.HeadlessStep())
	}
	postTick := deliveryScopeCut(cold, "next-production-tick")
	if postTick.Tick <= coldBefore.Tick || postTick.World == coldBefore.World {
		w.t.Fatal("cold ordinary LOAD did not execute the next production tick", coldBefore, postTick)
	}
	cold.front.live.world.SetPurse(sim.SelfSlot, cold.front.live.world.Purse(sim.SelfSlot)+1)
	lossControl := deliveryScopeCut(cold, "independent-current-purse-control")
	if lossControl.World == postTick.World || bytes.Equal(lossControl.raw, postTick.raw) {
		w.t.Fatal("independent current state mutation did not distinguish both World and full SAV")
	}
	cold.receipt("next-production-tick")
	deliveryScopeShutdown(cold)
	coldShutdown := deliveryScopeCut(cold, "shutdown")
	deliveryScopeEqual(w.t, lossControl, coldShutdown)
	deliveryScopeJSON(cold, "frozen-state", []deliveryFrozenCut{coldBefore, coldAfter, postTick, lossControl, coldShutdown})
	w.t.Logf("ordinary SAVE/cancel/draft failure/commit/cold LOAD/next tick: source World=%x cold=%x; full frozen SAV=%s; cold projection explicitly qualified", baseline.World, coldBefore.World, baseline.SAVHash)
}

type deliveryTownCut struct {
	Name     string
	Room     townRoom
	Part     int
	Key      string
	Pending  []string
	SAVHash  string
	SAVBytes int
	raw      []byte
}

func deliveryTownState(w *deliveryWitness, name string) deliveryTownCut {
	w.t.Helper()
	snapshot, _, err := w.front.Snapshot(false)
	w.check(err)
	raw, err := w.front.ExportCurrentSave(snapshot, "Town speech lifecycle")
	w.check(err)
	s := w.front.townUI
	w.check(os.WriteFile(filepath.Join(w.dir, name+"-current.sav"), raw, 0600))
	return deliveryTownCut{name, s.room, s.dialogue.displayPart, s.speech.key,
		append([]string(nil), s.speech.pending...), fmt.Sprintf("%x", sha256.Sum256(raw)), len(raw), raw}
}

func deliveryTownEqual(w *deliveryWitness, before, after deliveryTownCut) {
	w.t.Helper()
	if before.Room != after.Room || before.Part != after.Part || before.Key != after.Key ||
		!reflect.DeepEqual(before.Pending, after.Pending) || !bytes.Equal(before.raw, after.raw) {
		w.t.Fatalf("%s changed room, displayed line, response queue or full SAV: room %d/%d part %d/%d key %q/%q pending %v/%v SAV %s/%s", after.Name,
			before.Room, after.Room, before.Part, after.Part, before.Key, after.Key, before.Pending, after.Pending, before.SAVHash, after.SAVHash)
	}
}

func deliveryTownVoice(w *deliveryWitness) (audio.Voice, uint64, *deliveryWitnessPlayer, int64) {
	w.t.Helper()
	voice := w.front.townUI.speech.voice
	if voice == nil || !voice.Playing() {
		w.t.Fatal("installed controller did not start speech")
	}
	var buffer uint64
	for _, b := range w.owner.BackendState().Buffers {
		if b.Group == audio.SpeechChannel && b.Playing {
			if buffer != 0 {
				w.t.Fatal("instrument: ambiguous playing speech handle")
			}
			buffer = b.ID
		}
	}
	if buffer == 0 || int(buffer) > len(w.players) {
		w.t.Fatal("instrument: lowest speech buffer missing")
	}
	player := w.players[buffer-1]
	player.position = 17 * time.Millisecond
	phase := voice.(interface{ Phase() int64 }).Phase()
	if phase <= 0 {
		w.t.Fatal("instrument: speech phase did not advance")
	}
	return voice, buffer, player, phase
}

func deliveryTownRetained(w *deliveryWitness, voice audio.Voice, player *deliveryWitnessPlayer, phase int64, paused bool) {
	w.t.Helper()
	s := w.front.townUI
	if s.speech.voice != voice || s.speech.paused != paused || voice.Playing() == paused ||
		voice.(interface{ Phase() int64 }).Phase() != phase || player.closes != 0 || player.seeks != 0 || player.position != 17*time.Millisecond {
		w.t.Fatal("temporary inactivity replaced, closed, rewound or lost the speech handle", s.speech, player, phase, paused)
	}
	id := voice.(interface{ ID() audio.VoiceID }).ID()
	for _, sample := range w.owner.Service.Snapshot().Samples {
		if sample.ID == id.Sample {
			d := sample.Duplicates[id.Duplicate]
			if !d.Retained || d.Generation != id.Generation || d.Paused != paused || d.Playing == paused || d.Phase != phase {
				w.t.Fatal("temporary inactivity lost the reserved sample duplicate", id, d)
			}
			retained := 0
			for _, duplicate := range sample.Duplicates {
				if duplicate.Retained {
					retained++
				}
			}
			if retained != 1 {
				w.t.Fatal("temporary inactivity duplicated the current speech buffer", sample)
			}
			return
		}
	}
	w.t.Fatal("temporary inactivity destroyed the current sample", id)
}

func deliveryTownDestroyed(w *deliveryWitness, voice audio.Voice, buffer uint64, player *deliveryWitnessPlayer) {
	w.t.Helper()
	id := voice.(interface{ ID() audio.VoiceID }).ID()
	if player.closes != 1 || player.playing || voice.Playing() || voice.(interface{ Phase() int64 }).Phase() != 0 {
		w.t.Fatal("terminal speech boundary did not close its handle once", id, buffer, player)
	}
	for _, sample := range w.owner.Service.Snapshot().Samples {
		if sample.ID == id.Sample {
			w.t.Fatal("terminal speech boundary retained its sample", sample)
		}
	}
	for _, current := range w.owner.BackendState().Buffers {
		if current.ID == buffer {
			w.t.Fatal("terminal speech boundary retained its lowest buffer", current)
		}
	}
}

func deliveryTownActions(w *deliveryWitness, buffer uint64, sequence uint64, want []string) {
	w.t.Helper()
	var got []string
	for _, r := range w.owner.BackendState().Receipts {
		if r.Buffer == buffer && r.Sequence > sequence {
			got = append(got, r.Action)
		}
	}
	if !reflect.DeepEqual(got, want) {
		w.t.Fatal("lowest speech operations changed ordered pause/resume/destruction", buffer, got, want)
	}
}

func deliveryTownPaused(w *deliveryWitness) {
	w.openTown()
	s := w.front.townUI
	if !s.openTownDialogue(TownTavern, TownOffer{Mission: 100}, 25) {
		w.t.Fatal("instrument: installed tavern topic100/npc25 missing")
	}
	w.check(w.app.HeadlessStep())
	voice, buffer, player, phase := deliveryTownVoice(w)
	before := deliveryTownState(w, "before-focus")
	start := w.owner.BackendState()
	sequence := start.Receipts[len(start.Receipts)-1].Sequence
	w.receipt("before-focus")
	w.check(w.app.HeadlessFocus(false))
	w.check(w.app.HeadlessFocus(false))
	deliveryTownRetained(w, voice, player, phase, true)
	paused := deliveryTownState(w, "after-focus-loss")
	deliveryTownEqual(w, before, paused)
	w.receipt("after-focus-loss")
	w.check(w.app.HeadlessFocus(true))
	w.check(w.app.HeadlessStep())
	deliveryTownRetained(w, voice, player, phase, false)
	resumed := deliveryTownState(w, "after-focus-return")
	deliveryTownEqual(w, before, resumed)
	deliveryTownActions(w, buffer, sequence, []string{"stop", "play"})
	w.receipt("after-focus-return")
	player.playing = false
	w.check(w.app.HeadlessStep())
	if s.speech.voice != nil || player.closes != 1 || s.speech.paused {
		w.t.Fatal("natural line completion did not destroy its retained buffer once")
	}
	deliveryTownDestroyed(w, voice, buffer, player)
	completed := deliveryTownState(w, "after-natural-completion")
	deliveryTownEqual(w, before, completed)
	w.check(w.app.HeadlessFocus(false))
	w.check(w.app.HeadlessFocus(true))
	if s.speech.voice != nil || player.closes != 1 {
		w.t.Fatal("completed line replayed on focus return")
	}
	deliveryTownActions(w, buffer, sequence, []string{"stop", "play", "destroy"})
	w.receipt("after-natural-completion")
	if !s.openTownDialogue(TownTavern, TownOffer{Mission: 100}, 25) {
		w.t.Fatal("instrument: could not reopen installed dialogue")
	}
	w.check(w.app.HeadlessStep())
	oldVoice, oldBuffer, oldPlayer, _ := deliveryTownVoice(w)
	oldPart := s.dialogue.displayPart
	w.check(w.app.HeadlessActivate("dialogue"))
	if oldPlayer.closes != 1 || oldVoice.Playing() || s.room != roomTalk || s.dialogue.displayPart == oldPart || s.speech.voice == nil || s.speech.voice == oldVoice {
		w.t.Fatal("App line replacement did not destroy old voice and start the displayed part")
	}
	deliveryTownDestroyed(w, oldVoice, oldBuffer, oldPlayer)
	w.receipt("after-line-replacement")
	w.t.Logf("installed line replacement closed buffer%d at part%d", oldBuffer, oldPart)
	lastVoice, lastBuffer, lastPlayer, _ := deliveryTownVoice(w)
	for tries := 0; s.room == roomTalk && tries < 64; tries++ {
		w.check(w.app.HeadlessActivate("dialogue"))
	}
	if s.room == roomTalk || lastPlayer.closes != 1 || lastVoice.Playing() || s.speech.voice != nil {
		w.t.Fatal("App dialogue completion retained the outgoing line")
	}
	deliveryTownDestroyed(w, lastVoice, lastBuffer, lastPlayer)
	w.receipt("after-dialogue-completion")
	w.check(w.app.HeadlessKey("escape"))
	w.check(w.app.HeadlessActivate("SCHOOL"))
	for tries := 0; s.room == roomTalk && tries < 64; tries++ {
		w.check(w.app.HeadlessActivate("dialogue"))
	}
	if s.room != roomSchool {
		w.t.Fatal("instrument: ordinary App did not enter school", s.room)
	}
	s.CloseTip()
	s.startTownResponse([]string{"speech/training/npc33s1l1.wav", "speech/training/npc33s2l1.wav"})
	response, responseBuffer, responsePlayer, responsePhase := deliveryTownVoice(w)
	responseBefore := deliveryTownState(w, "before-menu")
	if len(responseBefore.Pending) != 1 {
		w.t.Fatal("instrument: installed two-response observation control has no pending line")
	}
	start = w.owner.BackendState()
	sequence = start.Receipts[len(start.Receipts)-1].Sequence
	w.receipt("before-menu")
	w.check(w.app.HeadlessKey("f3"))
	if w.app.Screen() != ui.ScreenLoad {
		w.t.Fatal("ordinary F3 did not open LOAD over the school", w.app.Screen())
	}
	w.check(w.app.HeadlessKey("escape"))
	if w.app.Screen() != ui.ScreenGameMenu {
		w.t.Fatal("LOAD cancel did not return to temporary menu")
	}
	w.check(w.app.HeadlessStep())
	deliveryTownRetained(w, response, responsePlayer, responsePhase, true)
	menu := deliveryTownState(w, "during-menu")
	deliveryTownEqual(w, responseBefore, menu)
	w.receipt("during-menu")
	w.check(w.app.HeadlessGameMenuAction("return"))
	w.check(w.app.HeadlessStep())
	deliveryTownRetained(w, response, responsePlayer, responsePhase, false)
	returned := deliveryTownState(w, "after-menu-return")
	deliveryTownEqual(w, responseBefore, returned)
	deliveryTownActions(w, responseBuffer, sequence, []string{"stop", "play"})
	w.receipt("after-menu-return")
	responsePlayer.playing = false
	w.check(w.app.HeadlessStep())
	if responsePlayer.closes != 1 || s.speech.voice == nil || s.speech.voice == response || len(s.speech.pending) != 0 {
		w.t.Fatal("resumed response did not complete into exactly its pending line")
	}
	deliveryTownDestroyed(w, response, responseBuffer, responsePlayer)
	w.receipt("after-queued-response")
	roomVoice, roomBuffer, roomPlayer, roomPhase := deliveryTownVoice(w)
	roomID := roomVoice.(interface{ ID() audio.VoiceID }).ID()
	var roomScope audio.ScopeID
	for _, sample := range w.owner.Service.Snapshot().Samples {
		if sample.ID == roomID.Sample {
			roomScope = sample.Duplicates[roomID.Duplicate].Request.Scope
		}
	}
	w.check(w.app.HeadlessFocus(false))
	deliveryTownRetained(w, roomVoice, roomPlayer, roomPhase, true)
	w.check(w.app.HeadlessKey("escape"))
	if s.room != roomSquare || roomPlayer.closes != 1 || roomVoice.Playing() || s.speech.voice != nil || len(s.speech.pending) != 0 {
		w.t.Fatal("App room destruction retained paused speech or its queue")
	}
	deliveryTownDestroyed(w, roomVoice, roomBuffer, roomPlayer)
	if roomScope == 0 || deliveryScopePresent(w.owner.Service.Snapshot(), roomScope) {
		w.t.Fatal("room destruction retained its request scope", roomScope)
	}
	w.receipt("after-room-destruction")
	deliveryScopeJSON(w, "frozen-town-state", []deliveryTownCut{before, paused, resumed, completed, responseBefore, menu, returned})
	w.t.Logf("focus phase%d buffer%d; menu phase%d buffer%d; full SAV retained across focus/menu; authored two-installed-response observation control", phase, buffer, responsePhase, responseBuffer)
	deliveryScopeShutdown(w)
}

func TestReleaseSharedAudioDeliveryScopeLifecycle(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("set AGAINROM_ASSETS to a lawful install")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(os.Getenv("AGAINROM_ROOT")) || !strings.EqualFold(filepath.Clean(root), filepath.Clean(os.Getenv("AGAINROM_ROOT"))) {
		t.Fatal("instrument: AGAINROM_ASSETS and AGAINROM_ROOT must name the same absolute install")
	}
	out := os.Getenv("AGAINROM_SFX_DELIVERY_WITNESS_DIR")
	if !filepath.IsAbs(out) {
		t.Fatal("instrument: AGAINROM_SFX_DELIVERY_WITNESS_DIR must name an absolute existing directory")
	}
	out, err := filepath.EvalSymlinks(out)
	if err != nil {
		t.Fatal(err)
	}
	install, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(install, out)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("instrument: witness output is inside the preserved install")
	}
	for _, name := range []string{"controlled", "physical", "town-pause"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(out, filepath.Base(install), "lifecycle", name)
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			w := newDeliveryWitness(t, dir, name != "physical")
			if name == "town-pause" {
				deliveryTownPaused(w)
				return
			}
			if name == "controlled" {
				deliveryScopeControlled(w)
				return
			}
			w.owner.SetSettings(audio.EffectsChannel, audio.Settings{Master: 100, Muted: true})
			sample, ok := w.front.SoundBank.Sample(100)
			if !ok {
				t.Fatal("instrument: installed physical registry:100 is missing")
			}
			scope := w.owner.NewScope()
			voice := audio.Dispatch(scope.Player(audio.EffectsChannel), sample, audio.FixedRequest("fixed-interface", "registry:100", audio.EffectsChannel, 220, true, audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
			if voice == nil || !voice.Playing() {
				t.Fatal("instrument: real physically muted wrapper did not dispatch")
			}
			for _, buffer := range w.owner.BackendState().Buffers {
				if !buffer.Physical || buffer.DeviceGain != 0 || buffer.Gain != 0 {
					t.Fatal("instrument: real physical wrapper was not muted", buffer)
				}
			}
			w.receipt("physical-dispatch")
			scope.Destroy()
			if voice.Playing() {
				t.Fatal("physical scope destroy left its voice playing")
			}
			deliveryScopeShutdown(w)
		})
	}
}
