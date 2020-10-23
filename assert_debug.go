// +build !release

package slimy

func assert(cond bool, msg interface{}) {
	if !cond {
		panic(msg)
	}
}
