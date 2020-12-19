package cpu

type World int64

func (w World) CalcChunk(x, z int32) bool {
	seed := int64(w) +
		int64(x*x*4987142) +
		int64(x*5947611) +
		int64(z*z)*4392871 + // sic
		int64(z*389711)
	seed ^= 987234911
	r := NewRandom(seed)
	return r.NextInt(10) == 0
}
