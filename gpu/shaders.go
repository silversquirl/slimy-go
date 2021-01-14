package gpu

const searchComp = `
#version 430 core
#extension GL_ARB_gpu_shader_int64 : enable
#extension GL_ARB_compute_variable_group_size : require // TODO: don't require
layout(local_size_variable) in;

uniform ivec2 offset;
uniform uint threshold;
layout(binding = 0) uniform sampler2DRect mask;
layout(binding = 0) uniform atomic_uint resultCount;
layout(std430, binding = 1) buffer resultData {
	uvec3 results[];
};

` + IsSlime + `
shared uint count;
void main() {
	if (gl_LocalInvocationIndex == 0) {
		count = 0;
	}
	memoryBarrierShared();
	barrier();

	ivec2 coord = ivec2(gl_WorkGroupID.xy + gl_LocalInvocationID.xy) + offset;
	bool slime = isSlime(coord);
	bool mask = texelFetch(mask, ivec2(gl_LocalInvocationID.xy)).r >= 0.5;

	atomicAdd(count, uint(slime) * uint(mask));
	memoryBarrierShared();
	barrier();

	if (gl_LocalInvocationIndex == 0) {
		if (count >= threshold) {
			uint idx = atomicCounterIncrement(resultCount);
			memoryBarrierAtomicCounter();
			results[idx] = uvec3(gl_WorkGroupID.xy, count);
		}
	}
}
`

// MIRROR CHANGES IN cmd/gslimy/main.go:coord
const Fcoord = `
	vec2 fcoord = vec2(1, -1) * ((gl_FragCoord.xy - dim/2)/view.z + view.xy);
`
const Coord = Fcoord + `
	ivec2 coord = ivec2(floor(fcoord));
`

const IsSlime = `
#line 54
#ifdef GL_ARB_gpu_shader_int64
// int64 extension
uniform int64_t worldSeed;
uint64_t slime_magic = 0x5DEECE66Dul;
uint64_t slime_mask = (1l << 48) - 1;

bool isSlime(ivec2 c) {
	// Calculate slime seed
	uint64_t seed = worldSeed +
		uint64_t(c.x*c.x*4987142) +
		uint64_t(c.x*5947611) +
		uint64_t(c.y*c.y)*4392871 +
		uint64_t(c.y*389711);
	seed ^= 987234911ul;
	// Calculate LCG seed
	seed = (seed ^ slime_magic) & slime_mask;
	// Calculate random value
	int bits, val;
	do {
		seed = (seed*slime_magic + 0xB) & slime_mask;
		bits = int(seed >> (48 - 31));
		val = bits % 10;
	} while (bits-val+9 < 0);
	// Check slime chunk
	return val == 0;
}

#else
// int64 emulation
uvec2 i64(int i) {
	return uvec2(int(i < 0) * -1, i);
}
uvec2 add64(uvec2 a, uvec2 b) {
	uvec2 v;
	v.y = uaddCarry(a.y, b.y, v.x);
	v.x += a.x + b.x;
	return v;
}
uvec2 mul64(uvec2 a, uvec2 b) {
	uvec2 v;
	umulExtended(a.y, b.y, v.x, v.y);
	v.x += a.x*b.y + a.y*b.x;
	return v;
}
// WARNING: DOES NOT WORK WITH SHIFT > 32
uvec2 rsh64(uvec2 v, int shift) {
	return uvec2(v.x >> shift, v.x<<(32-shift) | v.y>>shift);
}

uniform uvec2 worldSeedV;
uvec2 slimev_magic = uvec2(0x5, 0xDEECE66D);
uvec2 slimev_mask = uvec2(0xffff, -1);

bool isSlime(ivec2 c) {
	// Calculate slime seed
	uvec2 seed = worldSeedV;
	seed = add64(seed, i64(c.x*c.x*4987142));
	seed = add64(seed, i64(c.x*5947611));
	seed = add64(seed, mul64(i64(c.y*c.y), uvec2(0, 4392871)));
	seed = add64(seed, i64(c.y*389711));
	seed.y ^= 987234911u;
	// Calculate LCG seed
	seed = (seed ^ slimev_magic) & slimev_mask;
	// Calculate random value
	int bits, val;
	do {
		seed = add64(mul64(seed, slimev_magic), uvec2(0, 0xB)) & slimev_mask;
		bits = int(rsh64(seed, 48 - 31).y);
		val = bits % 10;
	} while (bits-val+9 < 0);
	// Check slime chunk
	return val == 0;
}
#endif
`
