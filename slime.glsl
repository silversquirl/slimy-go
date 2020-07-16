#version 430
#extension GL_ARB_gpu_shader_int64 : require

layout(local_size_x = 1, local_size_y = 1) in;

layout(location = 1) uniform int64_t world_seed;
layout(location = 2) uniform ivec2 start_pos;

layout(binding = 0) coherent restrict buffer output_buf {
	bool slime_chunks[];
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
	ivec2 pos = ivec2(gl_WorkGroupID.xy) + start_pos;
	uint idx = gl_WorkGroupID.y * gl_NumWorkGroups.x + gl_WorkGroupID.x;
	slime_chunks[idx] = is_slimy(world_seed, pos);
}
