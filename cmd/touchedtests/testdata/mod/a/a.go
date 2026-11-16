package a

func Leaf() int { return 1 }

func Mid() int { return Leaf() }

func Top() int { return Mid() }

func Unrelated() int { return 2 }

type T struct {
	Size int
	Name string
}

func (t T) Grow() int { return t.Size }

const Limit = 3

var Table = []int{Limit}

type Shape interface{ Area() int }

type Sq struct{}

func (Sq) Area() int { return 4 }

type Ci struct{}

func (Ci) Area() int { return 3 }
