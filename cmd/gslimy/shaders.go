package main

const fsVert = `
#version 430 core
void main() {
	float x = float((gl_VertexID & 1) << 2) - 1;
	float y = float((gl_VertexID & 2) << 1) - 1;
	gl_Position = vec4(x, y, 0, 1);
}
`

const gridFrag = `
#version 430 core
#extension GL_ARB_gpu_shader_int64 : require
layout(location = 0) uniform vec3 view; // xy is pan, z is zoom
layout(location = 1) uniform ivec2 dim; // dimensions of viewport
layout(location = 2) uniform int64_t worldSeed;
out vec4 color;
void main() {
	vec2 grid = mod(vec2(gl_FragCoord.xy + view.xy - vec2(dim/2)), view.z);
	if (all(greaterThanEqual(grid, vec2(1)))) discard;
	color = vec4(.3, .3, .3, 1);
}
`

const slimeFrag = `
#version 430 core
#extension GL_ARB_gpu_shader_int64 : require

layout(location = 0) uniform vec3 view; // xy is pan, z is zoom
layout(location = 1) uniform ivec2 dim; // dimensions of viewport
layout(location = 2) uniform int64_t worldSeed;
out vec4 color;

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

void main() {
	ivec2 coord = ivec2((gl_FragCoord.xy + view.xy - 0.5*vec2(dim)) / view.z);
	if (!isSlime(coord, worldSeed)) discard;
	color = vec4(0.4, 1, 0.4, 1);
}
`
