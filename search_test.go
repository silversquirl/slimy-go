package slimy

import "testing"

func checkResults(t *testing.T, got, expected []SearchResult) {
	if len(got) != len(expected) {
		t.Fatalf("Wrong number of results: expected %d, got %d", len(expected), len(got))
	}

	for i := range got {
		if got[i] != expected[i] {
			t.Fatalf("Incorrect result at index %d: expected %v, got %v", i, expected[i], got[i])
		}
	}
}

func TestSearchMaskTooBig(t *testing.T) {
	mask := Mask{64, 1}
	world := World(1)

	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Error("Expected mask bounds error")
			} else if err != "Mask bounds exceed section size" {
				panic(err)
			}
		}()
		world.Search(0, 0, 0, 1, 1, 0, mask)
	}()
}

func BenchmarkSearch100(b *testing.B) {
	mask := Mask{8, 1}
	world := World(1)

	for i := 0; i < b.N; i++ {
		world.Search(0, -100, -100, 0, 0, 1_000_000, mask)
	}
}

func BenchmarkSearch1k(b *testing.B) {
	mask := Mask{8, 1}
	world := World(1)

	for i := 0; i < b.N; i++ {
		world.Search(0, -500, -500, 500, 500, 1_000_000, mask)
	}
}

func BenchmarkSearch5k(b *testing.B) {
	mask := Mask{8, 1}
	world := World(1)

	for i := 0; i < b.N; i++ {
		world.Search(0, 0, 0, 5000, 5000, 1_000_000, mask)
	}
}
