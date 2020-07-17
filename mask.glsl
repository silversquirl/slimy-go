#version 430
#extension GL_ARB_compute_variable_group_size : require

layout(local_size_variable) in;

layout(location = 2) uniform ivec2 start_pos;
layout(location = 3) uniform uint orad;
layout(location = 4) uniform int thres;

layout(std430, binding = 0) readonly restrict buffer slime_buf {
	bool slime_chunks[];
};

layout(std430, binding = 1) readonly restrict buffer mask_buf {
	bool mask[];
};

layout(std430, binding = 2) coherent restrict buffer output_buf {
	ivec3 result[];
};
layout(binding = 3) uniform atomic_uint resulti;

shared uint count;

bool check_threshold(uint count) {
	return thres < 0 ? count <= uint(-thres) : count >= uint(thres);
}

void main() {
	// Zero count
	count = 0;
	barrier();

	// Mask index
	uint maskw = gl_LocalGroupSizeARB.x;
	uvec2 maskpos = gl_LocalInvocationID.xy;
	uint maski = maskpos.y * maskw + maskpos.x;

	// Chunk position/index
	uint slimew = gl_NumWorkGroups.x + 2*orad;
	uvec2 slimepos = gl_WorkGroupID.xy + maskpos;
	uint slimei = slimepos.y * slimew + slimepos.x;

	// Check mask and slime chunk status
	if (mask[maski] && slime_chunks[slimei]) {
		atomicAdd(count, 1);
	}

	groupMemoryBarrier();
	barrier();

	// Write to result buffer
	bool first = gl_LocalInvocationIndex == 0;
	if (first && check_threshold(count)) {
		ivec2 centre = ivec2(gl_WorkGroupID.xy) + start_pos;
		uint outi = atomicCounterIncrement(resulti);
		result[outi] = ivec3(centre, count);
	}
}
