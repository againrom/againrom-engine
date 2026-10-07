package game

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

func TestReleaseTownSpeechResponses(t *testing.T) {
	var delivered [][]byte
	var players []*deliveryWitnessPlayer
	restore := ui.SetDeliveryPlayerFactory(func(reader io.ReadSeeker) (ui.DeliveryDevicePlayer, error) {
		size, err := reader.Seek(0, io.SeekEnd)
		if err != nil || size <= 0 || size > 64<<20 {
			t.Fatalf("invalid installed buffer size %d: %v", size, err)
		}
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		raw := make([]byte, size)
		if _, err := io.ReadFull(reader, raw); err != nil {
			t.Fatal(err)
		}
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		delivered = append(delivered, raw)
		player := &deliveryWitnessPlayer{reader: reader, position: 17 * time.Millisecond}
		players = append(players, player)
		return player, nil
	})
	t.Cleanup(restore)
	f := releaseFront(t)
	owner := ui.DeliveryOwner(f.SoundPlayer)
	if owner == nil || owner != ui.DeliveryOwner(f.SpeechPlayer) {
		t.Fatal("installed response has no shared effects/speech owner")
	}
	teach, ok := f.SoundBank.namedSample(schoolTeachSound)
	if !ok || len(teach.PCM) == 0 {
		t.Fatalf("the installed teaching sound %s did not decode", schoolTeachSound)
	}
	var report bytes.Buffer
	if err := f.witnessTownResponses(&report); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(report.String(), "response: action="); got != 34 {
		t.Fatalf("covered %d response choices, want 30 teacher and 4 merchant", got)
	}
	pcm := make([]byte, len(teach.PCM)*4)
	for i, value := range teach.PCM {
		binary.LittleEndian.PutUint16(pcm[i*4:], uint16(value))
		binary.LittleEndian.PutUint16(pcm[i*4+2:], uint16(value))
	}
	taught := 0
	for _, raw := range delivered {
		if bytes.Equal(raw, pcm) {
			taught++
		}
	}
	if taught != 30 {
		t.Fatalf("%d exact installed teaching PCM buffers for 30 paid steps", taught)
	}
	requests, admitted := 0, 0
	for _, r := range owner.Service.Snapshot().Receipts {
		if r.Source != "school-training" || r.Selector != schoolTeachSound {
			continue
		}
		if r.Group != audio.EffectsChannel || r.Repeat || r.Placement != (audio.Placement{Left: 10000, Right: 10000}) {
			t.Fatalf("wrong typed teaching request %+v", r)
		}
		switch r.Reason {
		case audio.DeliveryAdmitted:
			requests++
			admitted++
		case audio.DeliverySampleBusy, audio.DeliveryChannelsBusy, audio.DeliverySampleUnavailable,
			audio.DeliveryDeviceUnavailable, audio.DeliveryBufferUnavailable, audio.DeliveryInvalidGroup:
			requests++
		}
	}
	if requests != 30 || admitted != 30 {
		t.Fatalf("typed teaching requests=%d admitted=%d, want 30/30", requests, admitted)
	}
	a := f.App("response shutdown")
	a.StopAudio()
	if err := audioWitnessClosed(owner); err != nil {
		t.Fatal(err)
	}
	for _, player := range players {
		if player.closes != 1 || player.playing {
			t.Fatal("response backend player survived shutdown or closed twice", player)
		}
	}
	t.Log("30 typed teaching requests admitted with exact installed PCM; controlled backend phase 17ms; shared focus/SAV/disposal controls passed")
}
