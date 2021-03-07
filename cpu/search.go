package cpu

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/vktec/slimy"
	"github.com/vktec/slimy/util"
)

const SectionSize = 128

type Searcher struct {
	workerCount int
	mask        Mask
}

func NewSearcher(workerCount int, mask Mask) (*Searcher, error) {
	return &Searcher{workerCount, mask}, nil
}
func (s *Searcher) Destroy() {}

func (s *Searcher) Search(x0, z0, x1, z1 int32, threshold int, worldSeed int64) []slimy.Result {
	w := World(worldSeed)

	mw, mh := s.mask.Bounds()
	if mw >= SectionSize || mh >= SectionSize {
		panic("Mask bounds exceed section size")
	}

	if s.workerCount <= 0 {
		s.workerCount = runtime.GOMAXPROCS(0)
	}

	sectionCh := make(chan *Section, 8)
	resultCh := make(chan []slimy.Result, 8)
	wgroup := new(sync.WaitGroup)
	ctx := searchContext{w, threshold, s.mask, wgroup, sectionCh, resultCh}
	go ctx.sendSections(x0, z0, x1, z1)

	wgroup.Add(s.workerCount)
	for i := 0; i < s.workerCount; i++ {
		go ctx.search()
	}

	var results []slimy.Result
	for sectionResults := range resultCh {
		start := len(results)
		results = append(results, sectionResults...)
		for i := start; i < len(results); i++ {
			for j := i; j > 0; j-- {
				if results[j].OrderBefore(results[j-1], threshold) {
					results[j-1], results[j] = results[j], results[j-1]
				} else {
					break
				}
			}
		}
	}
	return results
}

type searchContext struct {
	world     World
	threshold int
	mask      Mask
	wgroup    *sync.WaitGroup
	sectionCh chan *Section
	resultCh  chan []slimy.Result
}

func (ctx searchContext) sendSections(x0, z0, x1, z1 int32) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if z0 > z1 {
		z0, z1 = z1, z0
	}

	mx, mz := ctx.mask.Bounds()
	shiftX := SectionSize - mx + 1
	shiftZ := SectionSize - mz + 1

	for x := x0; x < x1; x += shiftX {
		for z := z0; z < z1; z += shiftZ {
			ctx.sectionCh <- &Section{X: x, Z: z}
		}
	}
	close(ctx.sectionCh)

	ctx.wgroup.Wait()
	close(ctx.resultCh)
}

func (ctx searchContext) search() {
	for sec := range ctx.sectionCh {
		sec.Compute(ctx.world)
		results := sec.Search(ctx.mask, ctx.threshold)
		if len(results) > 0 {
			ctx.resultCh <- results
		}
	}
	ctx.wgroup.Done()
}

type Section struct {
	X, Z  int32
	Slime [SectionSize * SectionSize]bool
}

func (sec *Section) Compute(world World) {
	for z := int32(0); z < SectionSize; z++ {
		for x := int32(0); x < SectionSize; x++ {
			sec.Set(x, z, world.CalcChunk(sec.X+x, sec.Z+z))
		}
	}
}

func (sec *Section) Search(mask Mask, threshold int) (results []slimy.Result) {
	w, h := mask.Bounds()
	offX, offZ := sec.X+w/2, sec.Z+h/2
	x1, z1 := SectionSize-w, SectionSize-h

	for z := int32(0); z < z1; z++ {
		for x := int32(0); x < x1; x++ {
			// TODO: avoid checking the full mask area every time
			//       This can be done by adding the new and subtracting the old chunks
			count := sec.CheckMask(x, z, mask)
			if checkThreshold(threshold, int(count)) {
				results = append(results, slimy.Result{x + offX, z + offZ, count})
			}
		}
	}
	return results
}
func checkThreshold(threshold, count int) bool {
	if threshold < 0 {
		return int(count) <= -threshold
	} else {
		return int(count) >= threshold
	}
}

func (sec *Section) CheckMask(x0, z0 int32, mask Mask) (count uint) {
	w, h := mask.Bounds()
	for z := int32(0); z < h; z++ {
		for x := int32(0); x < w; x++ {
			if sec.Get(x+x0, z+z0) && mask.Query(x, z) {
				count++
			}
		}
	}
	return count
}

func secIdx(x, z int32) int {
	util.Assert(x < SectionSize, "x out of range")
	util.Assert(z < SectionSize, "z out of range")
	return int(SectionSize*z + x)
}

func (sec *Section) Set(x, z int32, v bool) {
	sec.Slime[secIdx(x, z)] = v
}

func (sec *Section) Get(x, z int32) bool {
	return sec.Slime[secIdx(x, z)]
}

func (sec *Section) Print() {
	for z := int32(0); z < SectionSize; z++ {
		for x := int32(0); x < SectionSize; x++ {
			if x > 0 {
				fmt.Print(" ")
			}
			if sec.Get(x, z) {
				fmt.Print("x")
			} else {
				fmt.Print(" ")
			}
		}
		fmt.Print("\n")
	}
}
