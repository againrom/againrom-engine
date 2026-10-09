package game

import (
	"path/filepath"

	"againrom/pkg/base"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// The archives the front-end requires under the asset root.
const (
	MainArchive     = "main.res"     // menu art
	GraphicsArchive = "graphics.res" // terrain tiles
	ScenarioArchive = "scenario.res" // campaign maps
	WorldArchive    = "world.res"    // the placeable-definition table
	MoviesArchive   = "movies.res"   // the shop merchant's own static picture
)

// RequiredArchives is the fixed order OpenArchives opens the five in, and the
// same set DiscoverAssetRoot looks for when it decides whether a directory is an
// install.
//
// It is spelt ONCE, here, beside the constants it lists, for the reason
// graphicsPrefix is spelt beside its own. A second spelling in the discovery path
// is how a renamed container leaves this tree opening one set and recognising
// another: the game would refuse to start on an install the discovery had just
// accepted, and the two lists would agree again only by accident.
func RequiredArchives() []string {
	return []string{MainArchive, GraphicsArchive, ScenarioArchive, WorldArchive, MoviesArchive}
}

// graphicsPrefix is the address prefix of everything this package reads out
// of graphics.res: that container's identity segment — GraphicsArchive's
// stem, by the one derivation rule — and the separator.
//
// It is spelt ONCE, here, beside the archive name it belongs to, and every
// address this package builds is composed from it: the two registry
// constants and each sprite sheet the loaders resolve. A second spelling
// elsewhere is how a renamed container leaves half the addresses naming an
// identity nothing in the set answers.
const graphicsPrefix = "graphics/"

// mainPrefix is the address prefix of everything this package reads out of
// MainArchive — the language selector and every mission's event text — spelt
// beside that archive's name for the reason graphicsPrefix is spelt beside its
// own: one spelling, so a renamed container moves every address it owns together
// instead of leaving half of them naming an identity nothing answers.
const mainPrefix = "main/"

// worldPrefix is the address prefix of everything this package reads out of
// WorldArchive, spelt beside that archive's name for the same reason
// graphicsPrefix is spelt beside its own.
//
// The developer tool that walks the same table from a command line spells its
// own address, and neither reaches into the other: an address prefix belongs to
// the tier that opens the archive, and a constant shared between the two would
// have to live below both, in a tier that opens nothing.
const worldPrefix = "world/"

// moviesPrefix is the address prefix of everything this package reads out of
// MoviesArchive, spelt beside that archive's name for the same reason
// graphicsPrefix is spelt beside its own.
const moviesPrefix = "movies/"

// Archives is the set of archives the front-end opens at startup.
type Archives struct {
	Root string

	// Containers is the read-only filesystem over the five required archives,
	// in the fixed order below. An entry is named by an address carrying its
	// container's identity — `main/graphics/mainmenu/menu_.bmp` — so ONE
	// string identifies an entry across the whole set and a consumer states
	// which container its assets come from in its own constants, instead of the
	// front-end knowing it out of band on the consumer's behalf.
	//
	// It lists NO directory beneath the archives, deliberately. Listing the
	// asset root there would send every archive-miss to the host tree, where a
	// stray unpacked file would silently fill a slot that is absent today —
	// the accidental masking 0027 puts out of scope.
	Containers *vfs.FS

	// Loose is the read-only filesystem over the asset root alone, with no
	// archive beneath it: the files an install ships loose beside its
	// containers, reached by an address that names no container identity at
	// all.
	Loose *vfs.FS

	// Base is the profile the root was detected as, once, when it was opened.
	// A root no profile matches leaves it zero, which states no limit and is
	// the first game.
	Base base.Match
}

// Game is the game of the profile the root was detected as.
func (a *Archives) Game() base.Game { return a.Base.Profile.GameOf() }

// OpenArchives opens all five required archives under root, in a fixed order,
// and returns the first failure with the path in the message.
//
// All five are opened before anything else happens, INCLUDING graphics.res,
// which the menu itself never reads. That is deliberate: an install missing a
// piece the application will need should say so before the user picks a map, not
// after. Opening it lazily would also make the headless check pass on an install
// the windowed run then fails on, which is the opposite of what a check is for.
//
// The order is fixed so a broken install reports the same failure every time and
// the windowed and headless modes cannot name different ones. WorldArchive and
// MoviesArchive join it at the END rather than in their file-order place: the
// order decides which failure a doubly-broken install reports, every such pair
// that was pinned before either joined is a pair of the other three, and
// appending changes none of them.
//
// WorldArchive is REQUIRED and not optional, because what lies behind it is the
// movement domain and the per-class health of every placed unit. A front-end
// that ran without it would give every mover the ground domain and one health
// constant — which is a defect this tree has already had once, and would then
// have again on any install that lost the file, silently and with every test
// still green.
//
// MoviesArchive is REQUIRED for the same reason: it carries the shop merchant's
// own static picture (SHOP-MERCHANT-046), and an install missing it should say
// so before the user opens a shop, not after (1009).
//
// The failure the caller sees is the container filesystem's own, and it is
// the `open <path>: <err>` wording this function has always printed — the
// same string, and with the transitional handles gone there is no longer a
// second place in this function that could produce it.
func OpenArchives(root string) (*Archives, error) {
	names := RequiredArchives()
	hosts := make([]string, 0, len(names))
	for _, name := range names {
		hosts = append(hosts, filepath.Join(root, name))
	}
	containers, err := vfs.Open(hosts, nil)
	if err != nil {
		return nil, err
	}
	// No archive is listed here, and a listed directory is not validated at open
	// time, so this call has nothing to fail on today; its error is returned
	// rather than dropped because that is the contract, not because a root is
	// checked here.
	loose, err := vfs.Open(nil, []string{root})
	if err != nil {
		return nil, err
	}

	a := &Archives{Root: root, Containers: containers, Loose: loose}
	if match, err := DetectBase(root); err == nil {
		a.Base = match
	}
	return a, nil
}

// OpenContainers opens the container filesystem over the single archive at path,
// listing no directory beneath it.
//
// It is what a DEVELOPER FRONT-END takes: those tools are pointed at one
// archive by a flag rather than at an install root, so they cannot reach the
// three-archive filesystem OpenArchives builds, and the loaders they call
// now resolve addresses rather than bare entry paths.
//
// It exists HERE, in the library, and not at each call site, because the
// dependency DAG grants `pkg/vfs` to this tier and not to the cmd tier that reads
// through it: the front-ends receive the filesystem, and the allow-map that says
// which tier may name the container layer is unchanged by this story.
//
// The failure is the filesystem's own `open <path>: <err>` — the frozen
// wording both tools have always printed for an archive that will not open,
// from one place rather than a wrapper stacked on top of it.
func OpenContainers(path string) (*vfs.FS, error) {
	return vfs.Open([]string{path}, nil)
}

// OpenGraphics opens the container filesystem over the graphics archive at path
// and returns it together with the terrain tileset built over it.
//
// ONE OPEN SERVES BOTH, and that is what the return is for. The tileset is
// built off the filesystem — terrain's tile constants carry graphics.res's
// identity segment, so the render package names an address and nothing here
// has to know which container answers it on the package's behalf — and the
// same filesystem goes back, so a developer front-end that also wants the
// object bundle reads it through the filesystem already open rather than
// opening a second one over the same host file.
//
// OpenTileset came out with it. Its own reason for existing was that a caller
// needing terrain alone kept its signature, and the teardown leaves no such
// caller: the one call site wants the filesystem too, so the wrapper was
// indirection with nothing behind it.
//
// The error wrapping is `open <path>: <err>` and it is CONTRACT: it is what the
// standalone viewer has always printed when its archive will not open, and the
// text is frozen. It is the filesystem's own — the identical string OpenContainers
// and OpenArchives return, from the one place all three reach.
func OpenGraphics(path string) (*vfs.FS, *terrain.Tileset, error) {
	containers, err := OpenContainers(path)
	if err != nil {
		return nil, nil, err
	}
	return containers, terrain.LoadTileset(containers), nil
}
