package video

import (
	"fmt"

	"againrom/pkg/formats/reg"
)

// maxSidecarRecords bounds a declared fade or pan count before anything is
// allocated from it.
const maxSidecarRecords = 4096

// Fade is one palette fade record: from frame Start to End the palette is
// scaled by a factor that moves from From to To.
type Fade struct {
	Start, End int32
	From, To   float32
}

// Pan is one source-origin pan record: from frame Start the origin moves by
// (StepX, StepY) per frame until frame End.
type Pan struct {
	Start, End   int32
	StepX, StepY int32
}

// Sidecar is a movie's same-basename registry (VIDEO-031, VIDEO-032,
// REG-CUT-053). The zero value is a movie with no effects and origin (0, 0).
type Sidecar struct {
	StartX, StartY int32
	Fades          []Fade
	Pans           []Pan
}

// SidecarName is the registry member that sits beside a movie: the movie's
// last extension replaced with .reg (VIDEO-031).
func SidecarName(movie string) string {
	for i := len(movie) - 1; i >= 0; i-- {
		switch movie[i] {
		case '.':
			return movie[:i] + ".reg"
		case '/', '\\':
			return movie + ".reg"
		}
	}
	return movie + ".reg"
}

// ParseSidecar reads a cutscene registry. Every key defaults to 0 when absent,
// as the installed reader does; a negative count reads as none, and a count
// above maxSidecarRecords is refused before allocation.
func ParseSidecar(data []byte) (*Sidecar, error) {
	r, err := reg.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("video: sidecar: %w", err)
	}
	geti := func(section, key string) int32 {
		v, _ := r.GetInt(section, key)
		return v
	}
	s := &Sidecar{StartX: geti("Common", "startx"), StartY: geti("Common", "starty")}
	nFade, nPan := geti("Common", "nFadings"), geti("Common", "nPanaramings")
	if nFade > maxSidecarRecords || nPan > maxSidecarRecords {
		return nil, fmt.Errorf("video: sidecar: record count %d/%d exceeds bound", nFade, nPan)
	}
	for i := int32(1); i <= nFade; i++ {
		sec := fmt.Sprintf("Fading%d", i)
		f := Fade{Start: geti(sec, "startframe"), End: geti(sec, "endframe")}
		if v, ok := r.GetFloat(sec, "startfade"); ok {
			f.From = float32(v)
		}
		if v, ok := r.GetFloat(sec, "endfade"); ok {
			f.To = float32(v)
		}
		s.Fades = append(s.Fades, f)
	}
	for i := int32(1); i <= nPan; i++ {
		sec := fmt.Sprintf("Panaraming%d", i)
		s.Pans = append(s.Pans, Pan{Start: geti(sec, "startframe"), End: geti(sec, "endframe"), StepX: geti(sec, "stepx"), StepY: geti(sec, "stepy")})
	}
	return s, nil
}
