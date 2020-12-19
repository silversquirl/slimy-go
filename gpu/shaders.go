package gpu

const searchComp = `
#version 430 core
#extension GL_ARB_gpu_shader_int64 : require
#extension GL_ARB_compute_variable_group_size : require // TODO: don't require
layout(local_size_variable) in;

layout(location = 0) uniform ivec2 offset;
layout(location = 1) uniform int64_t worldSeed;
layout(location = 2) uniform uint threshold;
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
	bool slime = isSlime(coord, worldSeed);
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
bool isSlime(ivec2 c, int64_t worldSeed) {
	// Calculate slime seed
	uint64_t seed = worldSeed +
		uint64_t(c.x*c.x*4987142) +
		uint64_t(c.x*5947611) +
		uint64_t(c.y*c.y)*4392871 +
		uint64_t(c.y*389711);
	seed ^= 987234911ul;
	// Calculate LCG seed
	seed = (seed ^ 0x5DEECE66Dul) & uint64_t((1l << 48) - 1);
	// Calculate random value
	int bits, val;
	do {
		seed = (seed*0x5DEECE66Dul + 0xB) & uint64_t((1l << 48) - 1);
		bits = int(seed >> (48 - 31));
		val = bits % 10;
	} while (bits-val+9 < 0);
	// Check slime chunk
	return val == 0;
}
`
