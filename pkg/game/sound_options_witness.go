package game

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

// WitnessSoundOptions drives the installed panel and concrete players. The
// profile and rendered evidence must be in one explicit directory outside the
// install. A second process can verify startup against the profile it leaves.
func (f *FrontEnd) WitnessSoundOptions(root, output string, report io.Writer) error {
	root, err := editorPhysicalDirectory(root)
	if err != nil {
		return err
	}
	output, err = editorPhysicalDirectory(output)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, output)
	if err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return fmt.Errorf("sound witness: output must be outside install")
	}
	profile, err := filepath.Abs(f.Options.Path)
	if err != nil || profile != filepath.Join(output, "options.txt") {
		return fmt.Errorf("sound witness: -saves must name the saves child of the private output directory")
	}
	for _, name := range []string{"options.txt", "sound-options.png", "music-tracks.png"} {
		info, err := os.Lstat(filepath.Join(output, name))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("sound witness: output is not a regular file: %s", name)
		}
	}
	fmt.Fprintf(report, "sound: startup enabled=%v master=%d channels=%v\n", f.Sound.Enabled, f.Sound.Volume, f.SoundChannels)
	fmt.Fprintf(report, "sound: startup acknowledgments=%v\n", !f.acknowledgmentsOff)
	devices := []struct {
		name    string
		player  any
		channel audio.Channel
	}{
		{"music", f.MusicPlayer, audio.MusicChannel},
		{"effects", f.SoundPlayer, audio.EffectsChannel},
		{"speech", f.SpeechPlayer, audio.SpeechChannel},
		{"ambience", f.AmbientPlayer, audio.EffectsChannel},
		{"movie", f.CutsceneAudioPlayer, audio.ChannelCount},
	}
	check := func(active bool) error {
		master := audio.Settings{Master: f.Sound.Volume, Muted: !f.Sound.Enabled}
		for _, device := range devices {
			want := master
			if device.channel != audio.ChannelCount {
				want = f.SoundChannels.Settings(device.channel, master)
			}
			settings, gains, ok := ui.AudioDeviceState(device.player)
			if !ok || settings != want || active && len(gains) == 0 {
				return fmt.Errorf("sound witness: %s settings=%v want=%v active=%v gains=%v concrete=%v", device.name, settings, want, active, gains, ok)
			}
			gain := float64(want.Master) / 100
			if want.Muted {
				gain = 0
			}
			for _, actual := range gains {
				if math.Abs(actual-gain) > 0.000001 {
					return fmt.Errorf("sound witness: %s player gain=%g want=%g", device.name, actual, gain)
				}
			}
			fmt.Fprintf(report, "sound: device=%s settings=%v playing=%d gains=%v\n", device.name, settings, len(gains), gains)
		}
		return nil
	}
	if err := check(false); err != nil {
		return err
	}
	a := f.App("sound options witness")
	defer a.StopAudio()
	// A synthetic load result enters the production town/menu path without
	// reading or changing any owner save. The controls and players are real.
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "sound-witness", Label: "Sound witness"}} },
		func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
	if err := a.HeadlessKey("load"); err != nil {
		return err
	}
	if err := a.HeadlessActivate("@first"); err != nil {
		return err
	}
	a.Layout(640, 480)
	if err := a.HeadlessKey("escape"); err != nil {
		return err
	}
	if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
		return err
	}
	if err := f.setGameMenuSound(true, 100); err != nil {
		return err
	}
	sample, ok := f.SoundBank.Sample(100)
	if !ok || len(sample.PCM) == 0 {
		return fmt.Errorf("sound witness: installed effect absent")
	}
	// Repeat installed PCM to keep one-shot players alive during all controls.
	// This witness duration is authored; it is not a ROM1 timing observation.
	pcm := make([]int16, audio.DeviceRate*12)
	for i := range pcm {
		pcm[i] = sample.PCM[i%len(sample.PCM)]
	}
	sample = audio.Sample{Rate: audio.DeviceRate, PCM: pcm}
	placement := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}
	var voices []audio.Voice
	for _, group := range []audio.Channel{audio.EffectsChannel, audio.SpeechChannel} {
		player := f.SoundPlayer
		if group == audio.SpeechChannel {
			player = f.SpeechPlayer
		}
		request := audio.FixedRequest("witness-settings", fmt.Sprintf("witness:settings:%d", group), group, 100, false, placement)
		voice := audio.Dispatch(player, sample, request)
		if voice == nil {
			return fmt.Errorf("sound witness: typed settings voice was refused for group %d", group)
		}
		defer voice.Stop()
		voices = append(voices, voice)
	}
	track, ok := f.MusicBank.Track("town.wav")
	if !ok {
		return fmt.Errorf("sound witness: installed music absent")
	}
	f.MusicPlayer.Start(track)
	if voice := ui.RequestAmbient(f.AmbientPlayer, ui.AmbientRiver, sample,
		audio.FixedRequest("witness-settings", "witness:settings:ambient", audio.EffectsChannel, 220, true, placement)); voice == nil || !voice.Playing() {
		return fmt.Errorf("sound witness: typed ambient settings request was refused")
	}
	f.CutsceneAudioPlayer.Start(audio.DeviceRate, 1)
	mono := make([]byte, len(pcm)*2)
	for i, value := range pcm {
		binary.LittleEndian.PutUint16(mono[2*i:], uint16(value))
	}
	f.CutsceneAudioPlayer.Push(mono)
	deadline := time.Now().Add(3 * time.Second)
	for {
		_, gains, _ := ui.AudioDeviceState(f.CutsceneAudioPlayer)
		if len(gains) != 0 {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("sound witness: movie player did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	want := audio.ChannelVolumes{25, 50, 75}
	for channel, value := range want {
		at := ui.SoundSliderPoint(audio.Channel(channel), value)
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, at.X, at.Y); err != nil {
				return err
			}
		}
		if f.SoundChannels[channel] != value {
			return fmt.Errorf("sound witness: slider %d did not commit %d", channel, value)
		}
	}
	if err := check(true); err != nil {
		return err
	}
	for i := 0; i < 2; i++ {
		if err := a.HeadlessGameMenuAction("toggle-sound"); err != nil {
			return err
		}
		if err := check(true); err != nil {
			return err
		}
	}
	for _, voice := range voices {
		if !voice.Playing() {
			return fmt.Errorf("sound witness: active voice was replaced or stopped")
		}
	}
	loaded, err := f.Options.SoundChannelVolumes()
	if err != nil || loaded != want {
		return fmt.Errorf("sound witness: persisted channels=%v: %v", loaded, err)
	}
	picture := a.GameMenuPanel()
	if picture == nil {
		return fmt.Errorf("sound witness: installed menu picture absent")
	}
	if err := writeMediaFrame(output, "sound-options", picture); err != nil {
		return err
	}
	fmt.Fprintln(report, "sound: pointer channels=[25 50 75] active-player-gains=verified master-mute=verified profile=committed")
	for _, voice := range voices {
		voice.Stop()
	}
	a.StopAudio()
	if err := audioWitnessClosed(ui.DeliveryOwner(f.SoundPlayer)); err != nil {
		return err
	}
	probe, err := newAudioWitnessFront(f.Archives.Root, f.Options, f.Sound, f.SoundChannels, f.runtime.deterministicFrames)
	if err != nil {
		return err
	}
	probeApp := probe.App("sound media witness")
	defer probeApp.StopAudio()
	fmt.Fprintln(report, "sound: original shared service closed/empty; later media checks use a fresh installed front end and service")
	if err := probe.witnessMusicPlayback(output, report); err != nil {
		return err
	}
	if err := audioWitnessClosed(ui.DeliveryOwner(probe.SoundPlayer)); err != nil {
		return err
	}
	later, err := newAudioWitnessFront(probe.Archives.Root, probe.Options, probe.Sound, probe.SoundChannels, probe.runtime.deterministicFrames)
	if err != nil {
		return err
	}
	laterApp := later.App("sound dialogue witness")
	defer laterApp.StopAudio()
	if err := later.witnessCommandAcknowledgments(report); err != nil {
		return err
	}
	return later.WitnessTownMedia("", report)
}
