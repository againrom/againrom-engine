package game

import (
	"image"
	"time"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The school scene as the ROM1 description states it: the training draw's
// raw range, the idle wait's base, the rotate sound, the column's endpoints
// and the training families' frame counts.
const (
	schoolTrainingRawRange = 32768
	schoolTrainingIdleWait = 3000 * time.Millisecond
	schoolRotateSound      = "town/school/rotate.wav"
)

const townSchoolMoviesPrefix = moviesPrefix + "training/"

// schoolTrainingFamilyShape is each class's movie directory, frame width and
// last transition and idle numbers.
var schoolTrainingFamilyShape = [schoolClassCount]struct {
	dir            string
	width          int
	transitionLast int
	idleLast       int
}{
	{dir: "fighter", width: 160, transitionLast: 18, idleLast: 9},
	{dir: "mage", width: 172, transitionLast: 22, idleLast: 11},
}

// schoolColumnThreshold is the transition index at which a class change
// starts the column.
func schoolColumnThreshold(class int) int {
	if class == schoolMageClass {
		return 5
	}
	return 6
}

func schoolColumnEndpoint(mage bool) int {
	if mage {
		return 15
	}
	return 0
}

// schoolMovieNames are each class's transition and idle art entries.
var schoolMovieNames = [schoolClassCount][2]string{{"fighter-tr", "fighter-m"}, {"mage-tr", "mage-m"}}

// schoolMovieFrames answers the picture each side's movie shows: the idle
// frame over the transition frame, nil for none.
func schoolMovieFrames(s *townScreen) (fighter, mage image.Image) {
	art := s.in.TownSchoolArt.Value()
	if art == nil || !s.schoolPage().Ready() {
		return nil, nil
	}
	pick := func(class int) image.Image {
		side := s.schoolTraining().Sides[class]
		frames, index := art.Scene[schoolMovieNames[class][0]], side.TransitionIndex
		switch {
		case side.IdleActive:
			frames, index = art.Scene[schoolMovieNames[class][1]], side.IdleCached
		case !side.TransitionActive:
			return nil
		}
		if index < 0 || index >= len(frames) {
			return nil
		}
		return frames[index]
	}
	return pick(schoolFighterClass), pick(schoolMageClass)
}

// schoolSceneOnly paints only the school scene's groups onto a blank frame
// the size of the content region.
func schoolSceneOnly(s *townScreen, group string) *image.RGBA {
	dst := image.NewRGBA(ui.TownContentRegion)
	s.schoolPage().Paint(dst, group)
	return dst
}

// schoolTrainingCopy is a deep copy of the training actor's state.
func schoolTrainingCopy(s *townScreen) town.Training {
	c := *s.schoolTraining()
	c.Sides = append([]town.TrainingSide(nil), c.Sides...)
	c.Static = append([]town.TrainingStatic(nil), c.Static...)
	return c
}

// schoolTrainingFresh reports a training actor that is never entered.
func schoolTrainingFresh(s *townScreen) bool {
	tr := s.schoolTraining()
	if s.schoolPage().Ready() || tr.ColumnStarted {
		return false
	}
	for _, side := range tr.Sides {
		if side != (town.TrainingSide{}) {
			return false
		}
	}
	return true
}

// setSchoolIdleStatics initializes both idle timers at now with the given
// drawn extras, so no idle side arms before its wait.
func setSchoolIdleStatics(s *townScreen, now time.Time, fighterExtra, mageExtra time.Duration) {
	st := s.schoolTraining().Static
	for class, extra := range []time.Duration{fighterExtra, mageExtra} {
		st[class].Initialized, st[class].IdleLast, st[class].IdleExtra = true, now, extra
	}
}

// diamondState is the diamond bounce's phase, step and whether it stepped
// since it was armed.
type diamondState struct {
	frame, step int
	ready       bool
}

func schoolDiamondState(s *townScreen) diamondState {
	d := s.schoolDiamond()
	return diamondState{frame: d.Frame, step: d.Step, ready: d.Stepped}
}

func setSchoolDiamondState(s *townScreen, st diamondState) {
	d := s.schoolDiamond()
	d.Frame, d.Step, d.Stepped = st.frame, st.step, st.ready
}

// columnState is the column target's frame, goal, readiness and stamp.
type columnState struct {
	frame, target int
	ready         bool
	last          time.Time
}

func schoolColumnState(s *townScreen) columnState {
	c := s.schoolColumn()
	return columnState{frame: c.Frame, target: c.Goal, ready: c.Ready, last: c.Last}
}
