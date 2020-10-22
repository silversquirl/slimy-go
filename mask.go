package main

import "fmt"

type Mask interface {
	Bounds() (w, h int32)
	Query(x, z int32) bool
}

func PrintMask(m Mask) {
	w, h := m.Bounds()
	for z := int32(0); z < h; z++ {
		for x := int32(0); x < w; x++ {
			if x > 0 {
				fmt.Print(" ")
			}
			if m.Query(x, z) {
				fmt.Print("x")
			} else {
				fmt.Print(" ")
			}
		}
		fmt.Print("\n")
	}
}

type CircleMask struct {
	ORad, IRad int32
}

func (m CircleMask) Bounds() (w, h int32) {
	w = 2*m.ORad + 1
	return w, w
}

func (m CircleMask) Query(x, z int32) bool {
	x -= m.ORad
	z -= m.ORad
	d2 := x*x + z*z
	return m.IRad*m.IRad < d2 && d2 <= m.ORad*m.ORad
}
