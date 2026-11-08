package town

// The host's modes. Each is the text a town description's "host" section
// carries and names one way the town host presents or runs a town; the
// validator admits exactly the constants of each type.

// RowsMode is where the square's rows come from.
type RowsMode string

const (
	RowsChapterHints RowsMode = "chapter-hints"
	RowsHotspots     RowsMode = "hotspots"
)

// ShopMode is which shop the town's shop room holds.
type ShopMode string

const (
	ShopChapter ShopMode = "chapter"
	ShopBank    ShopMode = "bank"
)

// InnMode is which inn profile the town's tavern or inn page lists, talks
// and hires by.
type InnMode string

const (
	InnChapter InnMode = "chapter"
	InnBank    InnMode = "bank"
)

// TrackSourceMode is who names the town's music track.
type TrackSourceMode string

const (
	TrackDescription TrackSourceMode = "description"
	TrackCampaign    TrackSourceMode = "campaign"
)

// HeaderMode is how the town's header line is written.
type HeaderMode string

const (
	HeaderChapter  HeaderMode = "chapter"
	HeaderLocation HeaderMode = "location"
)

// FooterMode is what the town's footer lists.
type FooterMode string

const (
	FooterParty    FooterMode = "party"
	FooterServices FooterMode = "services"
)

// TalkMode is how a town conversation is opened and read.
type TalkMode string

const (
	TalkBuilding TalkMode = "building"
	TalkSection  TalkMode = "section"
)

// PagesMode is whether a room page is kept between visits or built at entry.
type PagesMode string

const (
	PagesRetained PagesMode = "retained"
	PagesEntry    PagesMode = "entry"
)

// ArtMode is where a room's art is loaded from.
type ArtMode string

const (
	ArtPages    ArtMode = "pages"
	ArtManifest ArtMode = "manifest"
)

// AudioRoomMode is which room the room sounds follow.
type AudioRoomMode string

const (
	AudioRoomBuilding  AudioRoomMode = "building"
	AudioRoomRequested AudioRoomMode = "requested"
)

// LeaveMode is where leaving a room goes.
type LeaveMode string

const (
	LeaveRoomMap      LeaveMode = "room-map"
	LeaveDestinations LeaveMode = "destinations"
)

// modeTexts is the JSON text of each constant of one mode type.
func modeTexts[T ~string](values ...T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}
