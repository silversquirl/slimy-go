package gpu

import "image"

func NewSearcher(mask image.Image) (*Searcher, error) {
	return NewGLFWSearcher(mask)
}
