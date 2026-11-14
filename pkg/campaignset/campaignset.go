// Package campaignset is the one answer to whether a number is a mission of
// an install's campaign and the install holds its map. The game, the map
// editor and the starter ask it; it imports no game code.
package campaignset

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/formats/reg"
	"againrom/pkg/installpath"
	"againrom/pkg/vfs"
)

// Archive is the container that holds the campaign's registry and maps.
const Archive = "scenario.res"

// The registry keys a town building's offer list is written under, and the
// stride of the main missions' numbers.
const (
	InnKey     = "InnMission"
	ShopKey    = "ShopMission"
	TCKey      = "TCMission"
	Stride     = 10
	sectionTag = "Mission"
)

// Prefix is the address prefix of everything read out of Archive.
var Prefix = mustPrefix()

// Registry is the address of the campaign's registry.
var Registry = Prefix + "scenario.reg"

func mustPrefix() string {
	identity, err := vfs.Identity(Archive)
	if err != nil {
		panic("campaignset: " + err.Error())
	}
	return identity + "/"
}

// MapAddress is the address of mission n's map; false for a number that is
// not positive.
func MapAddress(n int) (string, bool) {
	if n <= 0 {
		return "", false
	}
	return Prefix + strconv.Itoa(n) + ".alm", true
}

// SectionNumber is the mission number of a registry section name.
func SectionNumber(name string) (int, bool) {
	if len(name) <= len(sectionTag) || !strings.EqualFold(name[:len(sectionTag)], sectionTag) {
		return 0, false
	}
	n, err := strconv.Atoi(name[len(sectionTag):])
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// Numbers are the missions of a campaign: the main ones, the other sections
// and the ones a building offers.
type Numbers struct {
	Main, Side, Offered []int
}

// Read takes the numbers out of a parsed registry; a nil one holds none.
func Read(r *reg.Reg) Numbers {
	var c Numbers
	if r == nil || r.Root == nil {
		return c
	}
	seen := map[int]bool{}
	for _, section := range r.Root.Children {
		if !section.Dir {
			continue
		}
		n, ok := SectionNumber(section.Name)
		if !ok {
			continue
		}
		if n%Stride == 0 {
			c.Main = append(c.Main, n)
		} else {
			c.Side = append(c.Side, n)
		}
		for _, key := range [...]string{InnKey, ShopKey, TCKey} {
			for _, v := range Ints(section, key) {
				if v <= 0 || seen[int(v)] {
					continue
				}
				seen[int(v)] = true
				c.Offered = append(c.Offered, int(v))
			}
		}
	}
	sort.Ints(c.Main)
	sort.Ints(c.Side)
	sort.Ints(c.Offered)
	return c
}

// Ints is every integer a section's key holds, written as one or as an array.
func Ints(section *reg.Node, key string) []int32 {
	for _, ch := range section.Children {
		if ch.Dir || !strings.EqualFold(ch.Name, key) {
			continue
		}
		switch ch.Type {
		case reg.TypeInt:
			return []int32{ch.Int}
		case reg.TypeIntArray:
			return ch.Ints
		}
		return nil
	}
	return nil
}

// Check is nil when n is a mission of the campaign and fsys holds the map at
// address, and the reason otherwise.
func (c Numbers) Check(fsys *vfs.FS, n int, address string) error {
	if !slices.Contains(c.Main, n) && !slices.Contains(c.Side, n) && !slices.Contains(c.Offered, n) {
		return fmt.Errorf("number %d is no mission of this install's campaign", n)
	}
	if _, err := fsys.ReadFile(address); err != nil {
		return fmt.Errorf("the install holds no map for mission %d (%s)", n, address)
	}
	return nil
}

// Held answers, for the install under root, whether number n is a mission of
// its campaign with its map in place.
type Held struct {
	numbers Numbers
	fsys    *vfs.FS
}

// Open reads the campaign of the install under root.
func Open(root string) (*Held, error) {
	host, err := installpath.New(root).Path(Archive)
	if err != nil {
		return nil, err
	}
	fsys, err := vfs.Open([]string{host}, nil)
	if err != nil {
		return nil, err
	}
	b, err := fsys.ReadFile(Registry)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", Registry, err)
	}
	r, err := reg.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", Registry, err)
	}
	return &Held{numbers: Read(r), fsys: fsys}, nil
}

// Answer is the test the mission lists take: nil for a mission the install
// plays.
func (h *Held) Answer(n int) error {
	address, ok := MapAddress(n)
	if !ok {
		return fmt.Errorf("number %d is no mission number", n)
	}
	return h.numbers.Check(h.fsys, n, address)
}
