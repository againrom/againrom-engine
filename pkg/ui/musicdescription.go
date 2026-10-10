package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// The music controller is one builder over a description. A description is
// data: it names the archive, the track manifest, each scene's list, which
// scenes keep a list already playing, how a list is ordered, what a silent
// request does and the mission's music-area rule. The controller holds no
// track name and no game rule of its own.

// MusicDescription is one game's music as data. Every object in the encoded
// form carries a cite list naming the claim, divergence row or owner ruling
// behind its values; the controller ignores it.
type MusicDescription struct {
	Music    string                      `json:"music"`
	Cite     []string                    `json:"cite"`
	Archive  MusicArchiveSpec            `json:"archive"`
	Tracks   []string                    `json:"tracks"`
	List     string                      `json:"list"`
	Silence  string                      `json:"silence"`
	Unlisted string                      `json:"unlisted"`
	Scenes   map[string]MusicSceneSpec   `json:"scenes"`
	Towns    map[string]MusicTrackChoice `json:"towns"`
	Areas    *MusicAreaSpec              `json:"areas"`
}

// MusicArchiveSpec is the archive's path under the install root, '/'
// separated; a name that differs only in case resolves to it.
type MusicArchiveSpec struct {
	Path string   `json:"path"`
	Cite []string `json:"cite"`
}

// MusicSceneSpec is one scene's list. Keep: a fresh request whose list equals
// the list the player holds keeps it, resuming it when a silent request
// paused it; otherwise a fresh request sets the list and starts it.
// MageFirst is the school's list when the selected member is a mage.
type MusicSceneSpec struct {
	Tracks    []string `json:"tracks"`
	MageFirst []string `json:"mage-first"`
	Keep      bool     `json:"keep"`
	Cite      []string `json:"cite"`
}

// MusicTrackChoice is one track named by a value the screen answers.
type MusicTrackChoice struct {
	Track string   `json:"track"`
	Cite  []string `json:"cite"`
}

// MusicAreaSpec is the mission's music-area rule: every Period mission ticks
// the area holding the hero picks a theme, and a track end opens it.
type MusicAreaSpec struct {
	Period int      `json:"period"`
	Cite   []string `json:"cite"`
}

// The list rules. Shuffled: an ordinary list is permuted when Random Order is
// on and starts at its first entry; on the original's shared stream it draws
// a start entry and swaps n times only in Random Order. Ordered: a list keeps
// its order and starts at one draw modulo its length; Random Order does not
// apply.
const (
	MusicListShuffled = "shuffled"
	MusicListOrdered  = "ordered"
)

// The silence rules. Clear: a silent request stops the player and drops its
// list. Pause: it stops the player and keeps the list and the position.
const (
	MusicSilenceClear = "clear"
	MusicSilencePause = "pause"
)

// The unlisted rules: what a screen no scene owns does. Silence: it makes a
// silent request. Keep: it makes no request.
const (
	MusicUnlistedSilence = "silence"
	MusicUnlistedKeep    = "keep"
)

// musicSceneNames are the scene names a description may key.
var musicSceneNames = map[string]MusicScene{
	"menu": MusicMenu, "credits": MusicCredits, "chargen": MusicChargen, "campaign": MusicCampaign,
	"mission": MusicMission, "town": MusicTown, "shop": MusicShop, "tavern": MusicTavern, "school": MusicSchool,
}

// DecodeMusic decodes and checks one music description.
func DecodeMusic(data []byte) (*MusicDescription, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var d MusicDescription
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("music description: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("music description %s: trailing data", d.Music)
	}
	if err := d.validate(); err != nil {
		return nil, fmt.Errorf("music description %s: %w", d.Music, err)
	}
	return &d, nil
}

func (d *MusicDescription) validate() error {
	switch {
	case d.Music == "":
		return fmt.Errorf("no name")
	case d.Archive.Path == "":
		return fmt.Errorf("no archive path")
	case d.List != MusicListShuffled && d.List != MusicListOrdered:
		return fmt.Errorf("list rule %q", d.List)
	case d.Silence != MusicSilenceClear && d.Silence != MusicSilencePause:
		return fmt.Errorf("silence rule %q", d.Silence)
	case d.Unlisted != MusicUnlistedSilence && d.Unlisted != MusicUnlistedKeep:
		return fmt.Errorf("unlisted rule %q", d.Unlisted)
	case d.Areas != nil && d.Areas.Period <= 0:
		return fmt.Errorf("area period %d", d.Areas.Period)
	}
	for name, scene := range d.Scenes {
		if _, ok := musicSceneNames[name]; !ok {
			return fmt.Errorf("unknown scene %q", name)
		}
		for _, list := range [][]string{scene.Tracks, scene.MageFirst} {
			for _, track := range list {
				if !d.Known(track) {
					return fmt.Errorf("scene %s names %q outside the manifest", name, track)
				}
			}
		}
	}
	for id, town := range d.Towns {
		if !d.Known(town.Track) {
			return fmt.Errorf("town %s names %q outside the manifest", id, town.Track)
		}
	}
	return nil
}

// Known reports whether name is in the manifest, ignoring case.
func (d *MusicDescription) Known(name string) bool {
	if d == nil {
		return false
	}
	for _, known := range d.Tracks {
		if strings.EqualFold(name, known) {
			return true
		}
	}
	return false
}

// sceneSpec is the description's entry for scene; a scene it does not key
// is silent and keeps nothing.
func (d *MusicDescription) sceneSpec(scene MusicScene) MusicSceneSpec {
	if d == nil {
		return MusicSceneSpec{}
	}
	for name, s := range musicSceneNames {
		if s == scene {
			return d.Scenes[name]
		}
	}
	return MusicSceneSpec{}
}

// sceneTracks is scene's list, the mage-first list when asked and stated.
func (d *MusicDescription) sceneTracks(scene MusicScene, mageFirst bool) []string {
	spec := d.sceneSpec(scene)
	if mageFirst && len(spec.MageFirst) != 0 {
		return append([]string(nil), spec.MageFirst...)
	}
	return append([]string(nil), spec.Tracks...)
}

// TownTrack is the track the description names for a town value.
func (d *MusicDescription) TownTrack(id int) (string, bool) {
	if d == nil {
		return "", false
	}
	town, ok := d.Towns[fmt.Sprint(id)]
	return town.Track, ok && town.Track != ""
}
