package gpu

import (
	"image"

	"github.com/vktec/glhl"
)

func NewSearcher(mask image.Image) (*Searcher, error) {
	flags := glhl.Core
	if Debug {
		flags |= glhl.Debug
	}
	ctx, err := glhl.NewContext(4, 2, flags)
	if err != nil {
		return NewGLFWSearcher(mask)
	}
	s := &Searcher{ctx: ctx, getProcAddr: glhl.GetProcAddr}
	return s, s.init(mask)
}
