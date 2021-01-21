package gpu

import (
	"image"

	"github.com/vktec/glhl"
)

func NewSearcher(mask image.Image) (*Searcher, error) {
	ctx, err := glhl.NewContext(4, 3, glhl.Core|glhl.Debug)
	if err != nil {
		return nil, err
	}
	s := &Searcher{ctx: ctx, getProcAddr: glhl.GetProcAddr}
	return s, s.init(mask)
}
