package a

import (
	"os"
	"testing"
)

func fixture() int { return Mid() }

func TestLeafDirect(t *testing.T) { _ = Leaf() }

func TestTopIsTransitive(t *testing.T) { _ = Top() }

func TestUnrelated(t *testing.T) { _ = Unrelated() }

func TestMethod(t *testing.T) { _ = T{}.Grow() }

func TestField(t *testing.T) { _ = T{Name: "x"} }

func TestTable(t *testing.T) { _ = Table }

func TestHelper(t *testing.T) { _ = fixture() }

func TestReadsFile(t *testing.T) { _, _ = os.ReadFile("testdata/in.txt") }

func TestNothing(t *testing.T) {}

func TestSquare(t *testing.T) { _ = Sq{}.Area() }

func TestCircle(t *testing.T) { _ = Ci{}.Area() }

func TestShape(t *testing.T) {
	var s Shape = Ci{}
	_ = s.Area()
}
