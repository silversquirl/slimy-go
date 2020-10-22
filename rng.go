// Go implementation of Java random
// Not safe for concurrent use
package main

type Random struct {
	seed int64
}

const magic = 0x5DEECE66D

func NewRandom(seed int64) Random {
	return Random{mixSeed(seed)}
}

func mixSeed(seed int64) int64 {
	return (seed ^ magic) & ((1 << 48) - 1)
}

func (r *Random) SetSeed(seed int64) {
	r.seed = mixSeed(seed)
}

func (r *Random) Next(bits int) int32 {
	r.seed = (r.seed*magic + 0xB) & ((1 << 48) - 1)
	return int32(r.seed >> (48 - bits))
}

func (r *Random) NextInt(n int32) int32 {
	if n&-n == n {
		return int32((int64(n) * int64(r.Next(31))) >> 31)
	}

	var bits, val int32
	for {
		bits = r.Next(31)
		val = bits % n
		if bits+n > val {
			return val
		}
	}
}
