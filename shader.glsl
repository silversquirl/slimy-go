#version 430
#extension GL_ARB_gpu_shader_int64 : require

layout(local_size_x = LSIZEX, local_size_y = LSIZEY) in;

layout(location = 1) uniform int64_t world_seed;
layout(location = 2) uniform ivec2 start_pos;
layout(location = 3) uniform uint orad;

layout(std430, binding = 1) readonly restrict buffer mask_buf {
	bool mask[];
};

layout(std430, binding = 2) coherent restrict buffer output_buf {
	uint slime_count[];
};

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

void main() {
	// Widths of buffers
	uint outw = gl_NumWorkGroups.x;
	uint maskw = gl_WorkGroupSize.x;

	// Positions in buffers
	uvec2 outpos = gl_WorkGroupID.xy;
	uvec2 maskpos = gl_LocalInvocationID.xy;

	// Indices into buffers
	uint outi = outpos.y * outw + outpos.x;
	uint maski = maskpos.y * maskw + maskpos.x;

	// Chunk position
	ivec2 pos = ivec2(outpos + maskpos) + start_pos - ivec2(orad);

	// Zero count
	slime_count[outi] = 0;
	barrier();

	// Check mask and slime chunk status
	if (mask[maski] && is_slimy(world_seed, pos)) {
		groupMemoryBarrier();
		atomicAdd(slime_count[outi], 1);
	}
}
