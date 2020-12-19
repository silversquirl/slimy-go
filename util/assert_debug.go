// +build !release

package util

func Assert(cond bool, msg interface{}) {
	if !cond {
		panic(msg)
	}
}
