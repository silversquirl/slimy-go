package slimy

import "fmt"

type Mask struct {
	ORad, IRad int32
}

func (m Mask) Bounds() (w, h int32) {
	w = 2*m.ORad + 1
	return w, w
}

func (m Mask) Query(x, z int32) bool {
	x -= m.ORad
	z -= m.ORad
	d2 := x*x + z*z
	return m.IRad*m.IRad < d2 && d2 <= m.ORad*m.ORad
}

func (m Mask) Print() {
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
