package cpu

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func readInts(path string) []int64 {
	f, err := os.Open(filepath.Join("testdata", path))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var ret []int64
	for {
		var i int64
		if _, err := fmt.Fscan(f, &i); err == io.EOF {
			return ret
		} else if err != nil {
			panic(err)
		}
		ret = append(ret, i)
	}
}

func TestNextIntPow2(t *testing.T) {
	data := readInts("s1010_nextInt16384.txt")
	r := NewRandom(1010)
	for _, n := range data {
		n2 := r.NextInt(16384)
		if int32(n) != n2 {
			t.Error("Expected", n, "got", n2)
		}
	}
}

func TestNextIntNonPow2(t *testing.T) {
	data := readInts("s1010_nextInt100000.txt")
	r := NewRandom(1010)
	for _, n := range data {
		n2 := r.NextInt(100000)
		if int32(n) != n2 {
			t.Error("Expected", n, "got", n2)
		}
	}
}
