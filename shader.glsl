#version 430
#extension GL_ARB_gpu_shader_int64 : require
#extension GL_ARB_compute_variable_group_size : require

layout(local_size_variable) in;

layout(location = 1) uniform int64_t world_seed;
layout(location = 2) uniform ivec2 start_pos;
layout(location = 3) uniform uint orad;
layout(location = 4) uniform int thres;

layout(binding = 0) uniform atomic_uint resulti;

layout(std430, binding = 1) readonly restrict buffer mask_buf {
	bool mask[];
};

layout(std430, binding = 2) restrict buffer output_buf {
	ivec3 result[];
};

shared uint count;

int lcg_next(inout uint64_t seed, int bits) {
	seed = (seed * 0x5DEECE66DUL + 0xBUL) & ((1UL << 48) - 1);
	return int(seed >> (48 - bits));
}

int lcg_next_int(inout uint64_t seed, int max) {
	int bits, val;
	do {
		bits = lcg_next(seed, 31);
		val = bits % max;
	} while (bits - val + (max-1) < 0);
	return val;
}

bool is_slimy(int64_t world_seed, ivec2 pos) {
	uint64_t seed =
		world_seed +
		pos.x * pos.x * 4987142UL +
		pos.x * 5947611UL +
		pos.y * pos.y * 4392871UL +
		pos.y * 389711UL;
	seed ^= 987234911UL;
	seed = (seed ^ 0x5DEECE66DUL) & ((1UL << 48) - 1);
	return lcg_next_int(seed, 10) == 0;
}

bool check_threshold(uint count) {
	return thres < 0 ? count <= uint(-thres) : count >= uint(thres);
}

void main() {
	// Zero count
	bool first = gl_LocalInvocationIndex == 0;
	if (first) {
		count = 0;
	}
	barrier();

	// Mask index
	uint maskw = gl_LocalGroupSizeARB.x;
	uvec2 maskpos = gl_LocalInvocationID.xy;
	uint maski = maskpos.y * maskw + maskpos.x;

	// Circle centre position
	ivec2 centre = ivec2(gl_WorkGroupID.xy) + start_pos;

	// Chunk position
	ivec2 pos = centre + ivec2(maskpos) - ivec2(orad);

	// Check mask and slime chunk status
	if (mask[maski] && is_slimy(world_seed, pos)) {
		atomicAdd(count, 1);
	}

	groupMemoryBarrier();
	barrier();

	// Write to result buffer
	if (first && check_threshold(count)) {
		uint outi = atomicCounterIncrement(resulti);
		result[outi] = ivec3(centre, count);
	}
}
