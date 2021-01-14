package main

import "github.com/vktec/slimy/gpu"

const fsVert = `
#version 420 core
void main() {
	float x = float((gl_VertexID & 1) << 2) - 1;
	float y = float((gl_VertexID & 2) << 1) - 1;
	gl_Position = vec4(x, y, 0, 1);
}
`

const slimeFrag = `
#version 420 core
#extension GL_ARB_gpu_shader_int64 : enable

uniform vec3 view; // xy is pan, z is zoom
uniform ivec2 dim; // dimensions of viewport
out vec4 color;
` + gpu.IsSlime + `
void main() {
` + gpu.Coord + `
	if (!isSlime(coord)) discard;
	color = vec4(0.4, 1, 0.4, 1);
}
`

const maskFrag = `
#version 420 core
uniform vec3 view; // xy is pan, z is zoom
uniform ivec2 dim; // dimensions of viewport
uniform ivec2 origin;
layout(binding = 0) uniform sampler2DRect mask;
out vec4 color;
void main() {
` + gpu.Coord + `
	if (texelFetch(mask, ivec2(coord - origin)).r >= 0.5) discard;
	color = vec4(0, 0, 0, 0.8);
}
`

const gridFrag = `
#version 420 core
uniform vec3 view; // xy is pan, z is zoom
uniform ivec2 dim; // dimensions of viewport
out vec4 color;
void main() {
` + gpu.Fcoord + `
	vec2 grid = mod(fcoord * view.z, view.z);
	if (all(greaterThan(grid, vec2(1)))) discard;
	color = vec4(.3, .3, .3, 1);
}
`
