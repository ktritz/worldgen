package namegen

import (
	"encoding/binary"
	"hash/fnv"
)

// rng is a small deterministic PRNG (splitmix64). Each name-generation call
// creates its own rng from a hash of (namer seed, base, method, key), so
// results never depend on call order or shared state.
type rng struct {
	state uint64
}

func newRNG(seed uint64) *rng {
	return &rng{state: seed}
}

func (r *rng) next() uint64 {
	r.state += 0x9e3779b97f4a7c15
	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// intn returns a deterministic value in [0, n). n must be > 0.
func (r *rng) intn(n int) int {
	return int(r.next() % uint64(n))
}

// hashSeed folds a root seed and a sequence of string parts into a single
// 64-bit sub-seed using FNV-1a. Parts are separated by a 0x1f byte so that
// ("ab","c") and ("a","bc") hash differently.
func hashSeed(seed int64, parts ...string) uint64 {
	h := fnv.New64a()
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(seed))
	h.Write(b[:])
	for _, p := range parts {
		h.Write([]byte{0x1f})
		h.Write([]byte(p))
	}
	return h.Sum64()
}
