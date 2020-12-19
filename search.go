package slimy

type Searcher interface {
	Search(x0, z0, x1, z1 int32, threshold int, worldSeed int64) []Result
	Destroy()
}
