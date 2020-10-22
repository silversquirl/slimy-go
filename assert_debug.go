// +build !release

package main

func assert(cond bool, msg interface{}) {
	if !cond {
		panic(msg)
	}
}
