package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type speakerResolver struct {
	src           entrySource
	units         *terrain.UnitSet
	npcFaces      map[int32]data.NPCFace
	portraits     map[string]*image.RGBA
	figurePics    map[figureCacheKey]*image.RGBA
	playerFigure  *image.RGBA
	playerFigures map[int]*image.RGBA

	// cast is the live half of speaker resolution (speakeractors.go): who is
	// standing on this map, and what the world says each of them is wearing.
	cast speakerCast
}

// speakerFigure is the composed-figure cache SpeakerFace shares with the
// info column, keyed the same way (figures.go): the figure AND the worn set
// together, so two speakers of one figure wearing different things are two
// pictures and one speaker who re-equips is a third.
//
// THE WORN SET IS A PARAMETER, AND UNTIL 0160 IT WAS THE ZERO VALUE. This
// function keyed on the figure alone, on the ground that an npc RECORD
// carries no equipment — which is true of the record and was the wrong
// thing to ask. `DLG-FIGURE-020` reads the dialogue site as calling the
// world figure compositor on the SPEAKER'S OWN DRAWABLE, so the equipment is
// the speaker's, not the record's, and `DLG-FIGURE-021` establishes that no
// layer is dropped because the picture is going to a dialogue.
func (r *speakerResolver) speakerFigure(fig figureID, eq data.Equipment) *image.RGBA {
	key := figureCacheKey{fig: fig, eq: eq}
	if pic, tried := r.figurePics[key]; tried {
		return pic
	}
	pic, _ := composeUnitFigure(r.src, key.eq, fig)
	if r.figurePics == nil {
		r.figurePics = make(map[figureCacheKey]*image.RGBA)
	}
	r.figurePics[key] = pic
	return pic
}

// SpeakerFace is the FaceSource this driver is for its own mission: the
// speaker's picture and the window the pane cuts out of it.
//
// IT REACHES THE SAME TWO COMPOSERS THE INFO COLUMN DOES — classPortrait for a
// creature and composeUnitFigure for a person — through the same two caches, so
// a speaker's face and the picture shown when the player clicks that same kind
// of unit are one picture by construction rather than two addresses that agree.
//
// THE TIER DIGIT IS THE RECORD'S OWN `Face`, not the placement's tier and not a
// hardcoded 1. `REG-NPC-088` reads `Picture` into the synthesised actor's typeID
// and `Face` into its face byte, and the formatter appends that byte as the
// digit — so the four goblin records that differ only in `Face` name the four
// goblin pictures. This line passed 1 until the key was decoded, and every
// creature in every dialogue on both roots showed its first tier.
//
// THE WINDOW IS PASSED THROUGH AND NOT APPLIED. `REG-NPC-089` states it as an
// origin in the picture's own pixels and the 72x96 extent is a constant of the
// PANE, so the rectangle is built by the tier that owns the pane
// (ui.NoticeFaceWindow) and this function carries only what the registry said.
// A record stating no origin hands over the zero rectangle, which is that tier's
// own "use the engine's default".
//
// liveFigure is the first arm of SpeakerFace: which live actor this record
// is about, and what that actor is wearing.
//
// THE TWO ARMS ARE `DLG-SPEAKER-022`'s OWN. A live actor matching the section's
// predicate is drawn as himself — his figure directory, his sheet and the worn
// set the world holds for him at this moment, so equipping him changes what his
// dialogue figure wears. Otherwise the speaker is synthesised, and the record's
// own `Flags` and `Face` supply the figure.
//
// A SYNTHESISED SPEAKER IS BARE. `DLG-SPEAKER-022` reads the synthesiser's
// constructor clearing all twelve equipment slots and nothing on that path
// writing one. That is also the only coherent answer for town speakers: they
// are portraits in a conversation, not party heroes whose Humans-row starting
// equipment should be materialised onto the picture.
//
// NO LAYER IS DROPPED FOR BEING IN A DIALOGUE (`DLG-FIGURE-021`): the head slot
// is composed whenever the resolved worn set fills it, on both arms.
func (r *speakerResolver) liveFigure(rec data.NPCFace) (*image.RGBA, bool) {
	if a, ok := r.cast.resolve(rec); ok {
		var eq data.Equipment
		if r.cast.worn != nil {
			eq = r.cast.worn(a.id)
		}
		fig := a.fig
		fig.Hero = a.hero
		fig.Horse = !a.hero && data.FigureHasHorse(a.typeID)
		return r.speakerFigure(fig, eq), true
	}
	return nil, false
}

// NOT FINDING ONE IS NOT AN ERROR, which is what the seam's own doc has always
// said: a speaker number naming no record, a record whose class this bundle does
// not hold, a picture the install does not carry and a mission with no party
// subject to be `Me` all answer false, and the window draws its pane empty.
func (r *speakerResolver) SpeakerFace(speaker int) (*image.RGBA, image.Rectangle, bool) {
	rec, ok := r.npcFaces[int32(speaker)]
	if !ok {
		return nil, image.Rectangle{}, false
	}
	// THE LIVE SEARCH PRECEDES THE PICTURE KIND. The registry's Portrait,
	// Figure and no-picture arms describe what to synthesise only after the
	// client actor walk found nobody. Running the switch first made a carried
	// Brian synthetic and made npc23/Naira reuse the current inventory subject.
	pic, live := r.liveFigure(rec)
	if !live {
		switch rec.Kind {
		case data.NPCPortrait:
			if r.units != nil {
				pic = classPortrait(r.src, r.units.Classes[rec.Class], rec.Face, r.portraits)
			}
		case data.NPCFigure:
			// PARTY-ADDHERO-017
			if own := r.playerFigures[speaker]; own != nil {
				pic = own
				break
			}
			// THE HERO TOKEN IS THE DRAWABLE'S HERO BIT (`TAVERN-TALKSTATS-017`),
			// and the compositor draws the warrior backdrop behind a drawable that
			// has it and no mage bit (`HERO-FIGURE-144`): a Hero record nobody
			// answers for stands on that backdrop and is still bare.
			hero := rec.Tokens.Has(data.NPCTokenHero)
			pic = r.speakerFigure(figureID{Dir: rec.Dir, Face: rec.Face, Hero: hero}, data.Equipment{})
		default:
			if own := r.playerFigures[speaker]; own != nil {
				pic = own
				break
			}
			if rec.Archetype {
				if dir, face, hero, ok := rec.SynthesisedFigure(r.cast.playerDir, r.cast.hasPlayer); ok {
					pic = r.speakerFigure(figureID{Dir: dir, Face: face, Hero: hero}, data.Equipment{})
				}
				break
			}
			// THE PLAYER'S OWN CHARACTER, and the picture is the one already
			// composed for the inventory window — worn layers and all, so the hero
			// speaks wearing what he is wearing.
			pic = r.playerFigures[speaker]
			if pic == nil {
				pic = r.playerFigure
			}
		}
	}
	var win image.Rectangle
	if rec.HasWindow {
		win = ui.NoticeFaceWindow(rec.WindowX, rec.WindowY)
	}
	return pic, win, pic != nil
}

func (mw *mapWorld) SpeakerFace(speaker int) (*image.RGBA, image.Rectangle, bool) {
	r := speakerResolver{
		src: mw.archive(), units: mw.units, npcFaces: mw.npcFaces,
		portraits: mw.portraits, figurePics: mw.figurePics,
		// THE CAST IS BUILT PER CALL AND THE CANDIDATE LIST IS NOT. The list
		// is resolved once when the map opens; the three
		// function values below close over this driver so the worn set and the
		// liveness answer come from the world as it stands NOW, which is
		// `DLG-SPEAKER-022`'s own "those slots change only by a re-send".
		cast: speakerCast{
			actors: mw.speakerActors,
			alive:  mw.entityAlive,
			worn:   mw.equipmentOf,
		},
	}
	if dir, ok := mw.playerFigureDir(); ok {
		r.cast.playerDir, r.cast.hasPlayer = dir, true
	}
	if mw.invSubjectSet {
		r.playerFigure = mw.invSubject.Figure
	}
	return r.SpeakerFace(speaker)
}

// entityAlive is the cast's state gate over this driver's own world: an id the
// world no longer holds is not alive, which is the same answer a corpse gets
// and the one a dialogue needs.
func (mw *mapWorld) entityAlive(id sim.EntityID) bool {
	e, held := mw.entity(id)
	return held && e.Alive()
}

// playerFigureDir is the player's own figure directory, for the `MySex` and
// `MyClass` terms — the party member the start marked as the player's
// character, or no answer at all for a driver with no party.
//
// IT READS THE MEMBER AND NOT THE COMPOSED SUBJECT. The inventory subject is
// whatever the player last clicked; the terms compare against the player's OWN
// drawable (`DLG-SPEAKER-023`'s `[this+0x3f54]`), which is his character
// however the window is pointed.
func (mw *mapWorld) playerFigureDir() (data.FigureDir, bool) {
	if mw.mission == nil {
		return "", false
	}
	for _, m := range mw.mission.party {
		if primaryPlayerHero(m) {
			dir, _ := memberFigure(m)
			return dir, true
		}
	}
	return "", false
}
