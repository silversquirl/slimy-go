package slimy

type Result struct {
	X, Z  int32
	Count uint
}

func (a Result) OrderBefore(b Result) bool {
	// Sort by count
	if a.Count != b.Count {
		return a.Count > b.Count
	}

	// Then by distance from 0,0
	aD2 := a.X*a.X + a.Z*a.Z
	bD2 := b.X*b.X + b.Z*b.Z
	if aD2 != bD2 {
		return aD2 < bD2
	}

	// Then finally break ties by coordinate
	if a.X != b.X {
		return a.X < b.X
	}
	if a.Z != b.Z {
		return a.Z < b.Z
	}
	return false
}
