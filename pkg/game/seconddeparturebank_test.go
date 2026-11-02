package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// departureProbe runs mission n's ordinary departure over a start bank whose
// gate slots are set by mask (bit i holds gate slot i nonzero), over a plain
// bank and over a bank every slot of which is filled, and digests the bank,
// the output DWORD, the available list and the room the departure leaves.
func departureProbe(n, mask int, filled bool) string {
	var bank [1024]int32
	if filled {
		for i := range bank {
			bank[i] = int32(i*7%5 + 1)
		}
	}
	gates := []int{772, 777, 778, 779, 780}
	for i, s := range gates {
		bank[s] = 0
		if mask&(1<<i) != 0 {
			bank[s] = 1
		}
	}
	c := newSecondCampaign()
	c.current = partyPlace{kind: placeMission, id: n}
	c.available = []partyPlace{c.current, {kind: placeMission, id: 99}}
	out := c.completeBank(bank)
	h := sha256.New()
	for _, v := range c.bank {
		_ = binary.Write(h, binary.LittleEndian, v)
	}
	_ = binary.Write(h, binary.LittleEndian, int32(out))
	for _, l := range c.available {
		fmt.Fprintf(h, "%d/%d;", l.kind, l.id)
	}
	fmt.Fprintf(h, "%v/%v", c.current, c.room)
	return fmt.Sprintf("%x", h.Sum(nil)[:8])
}

// TestSecondDepartureBankDigests holds the bank every departure case leaves,
// with each gate slot held and not held, equal to the digest recorded from
// the departure switch before the departures became rows.
func TestSecondDepartureBankDigests(t *testing.T) {
	var got []string
	for _, n := range []int{1, 10, 20, 30, 31, 32, 40, 50, 60, 70, 80, 90, 100, 110} {
		for mask := 0; mask < 32; mask++ {
			for _, filled := range []bool{false, true} {
				got = append(got, fmt.Sprintf("%d/%d/%t=%s", n, mask, filled, departureProbe(n, mask, filled)))
			}
		}
	}
	sort.Strings(got)
	sum := sha256.Sum256([]byte(strings.Join(got, "\n")))
	t.Logf("digest %x over %d probes", sum[:], len(got))
	const want = "b006338a209f6f2b8e17a9cab523513ed0d422a535a534d6758135b93694fb8b"
	if have := fmt.Sprintf("%x", sum[:]); have != want {
		t.Fatalf("departure banks changed: %s", have)
	}
}

// TestSecondDepartureExitsAreThePublishedOnes holds each departure's exits
// and movies, as the census lists them, to the published list.
func TestSecondDepartureExitsAreThePublishedOnes(t *testing.T) {
	want := map[int]string{
		10: "mission 20,movie 1", 20: "town 2,mission 21 if 772!=0", 30: "movie 2", 31: "mission 32",
		40: "mission 50,mission 60", 50: "town 3,mission 52 if 780!=0", 60: "mission 80",
		70: "movie 3 if 777=0 778!=0", 80: "movie 3 if 778=0 777!=0", 90: "-", 100: "-",
		110: "movie 5 if 779!=0,movie 4 if 779=0",
	}
	if len(secondDepartures) != len(want) {
		t.Fatalf("%d departure rows, want %d", len(secondDepartures), len(want))
	}
	for n, w := range want {
		exits, ok := secondDepartureExits(n)
		if !ok {
			t.Errorf("mission %d has no departure row", n)
			continue
		}
		if got := secondExitList(exits, false); got != w {
			t.Errorf("mission %d exits %q, want %q", n, got, w)
		}
	}
}
