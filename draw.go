package slimy

import (
	"image/color"
	"image/draw"
	"runtime"
	"sync"
)

// Draws slime chunks on an image. The search area comes from the image's Bounds
func (w World) DrawArea(workerCount int, dst draw.Image) {
	if workerCount <= 0 {
		workerCount = runtime.GOMAXPROCS(0)
	}

	bounds := dst.Bounds()
	x0, x1 := int32(bounds.Min.X), int32(bounds.Max.X)
	z0, z1 := int32(bounds.Min.Y), int32(bounds.Max.Y)

	sectionCh := make(chan *Section, 8)
	resultCh := make(chan []SearchResult, 8)
	wgroup := new(sync.WaitGroup)
	ctx := searchContext{w, 0, Mask{}, wgroup, sectionCh, resultCh}
	go ctx.sendSections(x0, z0, x1, z1)

	wgroup.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go ctx.draw(dst)
	}

	for range resultCh {
	}
}

var (
	backgroundColor = color.RGBA{0, 0, 0, 255}
	slimeChunkColor = color.RGBA{100, 255, 100, 255}
)

func (ctx searchContext) draw(dst draw.Image) {
	for sec := range ctx.sectionCh {
		sec.Compute(ctx.world)
		for z := int32(0); z < SectionSize; z++ {
			for x := int32(0); x < SectionSize; x++ {
				color := backgroundColor
				if sec.Get(x, z) {
					color = slimeChunkColor
				}
				dst.Set(int(x+sec.X), int(z+sec.Z), color)
			}
		}
	}
	ctx.wgroup.Done()
}
