#include <stdio.h>
#include <string.h>
#include <GL/glew.h>
#include <GLFW/glfw3.h>
#include "common.h"
#include "gpu.h"

#include "shader.glsl.h"

static void glfw_error(int code, const char *desc) {
	fprintf(stderr, "GLFW error: %s\n", desc);
}

int gpu_init(void) {
	if (!glfwInit()) {
		fprintf(stderr, "Error initializing GLFW\n");
		return 1;
	}

	glfwSetErrorCallback(glfw_error);

	glfwWindowHint(GLFW_CONTEXT_VERSION_MAJOR, 4);
	glfwWindowHint(GLFW_CONTEXT_VERSION_MINOR, 3);
	glfwWindowHint(GLFW_VISIBLE, GLFW_FALSE);

	GLFWwindow *win = glfwCreateWindow(1, 1, "", NULL, NULL);
	if (!win) {
		fprintf(stderr, "Error creating GLFW window\n");
		glfwTerminate();
		return 1;
	}

	glfwMakeContextCurrent(win);
	if (glewInit() != GLEW_OK) {
		fprintf(stderr, "Error initializing GLEW\n");
		glfwTerminate();
		return 1;
	}

	if (!GLEW_ARB_gpu_shader_int64) {
		fprintf(stderr, "64-bit integer extension unavailable\n");
		glfwTerminate();
		return 1;
	}

	if (!GLEW_ARB_compute_variable_group_size) {
		fprintf(stderr, "Variable workgroup size extension unavailable\n");
		glfwTerminate();
		return 1;
	}

	return 0;
}

static GLuint build_program(const char *name, const char *shadersrc) {
	GLuint shad = glCreateShader(GL_COMPUTE_SHADER);
	glShaderSource(shad, 1, &shadersrc, NULL);
	glCompileShader(shad);

	GLint result, log_len;
	glGetShaderiv(shad, GL_COMPILE_STATUS, &result);
	glGetShaderiv(shad, GL_INFO_LOG_LENGTH, &log_len);

	if (log_len > 0) {
		char buf[log_len];
		glGetShaderInfoLog(shad, log_len, NULL, buf);
		fprintf(stderr, "Error compiling shader '%s':\n%.*s\n", name, log_len, buf);
	}

	if (!result) {
		glDeleteShader(shad);
		return 0;
	}

	GLuint prog = glCreateProgram();
	glAttachShader(prog, shad);
	glLinkProgram(prog);

	glGetProgramiv(prog, GL_LINK_STATUS, &result);
	glGetProgramiv(prog, GL_INFO_LOG_LENGTH, &log_len);

	if (log_len > 0) {
		char buf[log_len];
		glGetProgramInfoLog(prog, log_len, NULL, buf);
		fprintf(stderr, "Error linking shader '%s':\n%.*s\n", name, log_len, buf);
	}

	glDetachShader(prog, shad);
	glDeleteShader(shad);

	return result ? prog : 0;
}


#define vgl_perror() _vgl_perror(vgl_strerror(), __LINE__, __func__, __FILE__)
static inline int _vgl_perror(const char *err, int line, const char *func, const char *file) {
	if (!err) return 0;
	fprintf(stderr, "%s:%d (%s): %s\n", file, line, func, err);
	return 1;
}

static const char *vgl_strerror(void) {
	switch (glGetError()) {
	case GL_NO_ERROR:
		break;

#define _vgl_match_error(err) case GL_##err: return #err;
	_vgl_match_error(INVALID_ENUM);
	_vgl_match_error(INVALID_VALUE);
	_vgl_match_error(INVALID_OPERATION);
	_vgl_match_error(INVALID_FRAMEBUFFER_OPERATION);
	_vgl_match_error(OUT_OF_MEMORY);
	_vgl_match_error(STACK_UNDERFLOW);
	_vgl_match_error(STACK_OVERFLOW);
	}
	return NULL;
}

// GLSL's bool is the same size as uint, and GLboolean isn't
typedef GLuint GLSLbool;
// GLSL's vec3 type has vec4 alignment
typedef struct {GLint x, y, z, _;} GLSLivec3;

int gpu_search(struct searchparams *param) {
	int orad = param->outer_rad;
	int orad2 = orad*orad;
	int irad = param->inner_rad;
	int irad2 = irad*irad;
	int side = 2*orad+1; // Side-length of bounding box of outer radius

	if (side > 1024) {
		fprintf(stderr, "Outer radius too big, must be less than 512\n");
		return 1;
	}

	// Mask info
	GLSLbool mask[side*side];
	size_t mask_size = side * side * sizeof *mask;
	for (int cz = -orad; cz <= orad; cz++) {
		int cz2 = cz*cz;
		int owidth = isqrt(orad2 - cz2);
		int iwidth = irad2 < cz2 ? 0 : isqrt(irad2 - cz2);

		int i = (orad + cz) * side + orad;
		for (int cx = 0; cx <= orad; cx++) {
			mask[i + cx] = mask[i - cx] = (cx >= iwidth && cx <= owidth);
		}
	}

	int searchw = 2*param->range;
	GLuint groupw = searchw, groupr = 0, collw = 1;
	enum {GROUP_LIMIT = 0x800};
	if (groupw > GROUP_LIMIT) {
		groupw = GROUP_LIMIT;
		collw = searchw / groupw;
		groupr = searchw % groupw;
		if (groupr) collw++;
	}

	GLuint prog = build_program("shader.glsl", shader_glsl);
	if (!prog) return 1;

	// Load GPU parameters
	glUseProgram(prog);
	glUniform1i64ARB(1, param->seed);
	glUniform1ui(3, orad);
	glUniform1i(4, param->threshold);

	GLuint bufs[3];
	glGenBuffers(3, bufs);
	GLuint count_buf = bufs[0], mask_buf = bufs[1], result_buf = bufs[2];

	// Allocate counter
	GLuint count_data = 0;
	glBindBuffer(GL_ATOMIC_COUNTER_BUFFER, count_buf);
	glBufferData(GL_ATOMIC_COUNTER_BUFFER, sizeof count_data, &count_data, GL_DYNAMIC_READ);
	glBindBufferBase(GL_ATOMIC_COUNTER_BUFFER, 0, count_buf);
	if (vgl_perror()) return 1;
	glBindBuffer(GL_ATOMIC_COUNTER_BUFFER, 0);

	// Load mask
	glBindBuffer(GL_SHADER_STORAGE_BUFFER, mask_buf);
	glBufferData(GL_SHADER_STORAGE_BUFFER, mask_size, mask, GL_STATIC_DRAW);
	if (vgl_perror()) return 1;
	glBindBufferBase(GL_SHADER_STORAGE_BUFFER, 1, mask_buf);

	// Allocate result buffer
	size_t result_len = groupw * groupw;
	size_t result_size = result_len * sizeof (GLSLivec3);
	glBindBuffer(GL_SHADER_STORAGE_BUFFER, result_buf);
	glBufferData(GL_SHADER_STORAGE_BUFFER, result_size, NULL, GL_STREAM_READ);
	if (vgl_perror()) return 1;
	glBindBufferBase(GL_SHADER_STORAGE_BUFFER, 2, result_buf);

	glBindBuffer(GL_SHADER_STORAGE_BUFFER, 0);

	glUseProgram(prog);

	if (vgl_perror()) return 1;

	struct chunkpos pos = {-param->range, -param->range};
	for (GLuint collx = 0; collx < collw; collx++) {
		GLuint gwidth = groupw;
		if (!collx && groupr) gwidth = groupr;

		pos.z = -param->range;
		for (GLuint colly = 0; colly < collw; colly++) {
			GLuint gheight = groupw;
			if (!colly && groupr) gheight = groupr;

			// Compute values
			glUniform2i(2, pos.x, pos.z);
			glDispatchComputeGroupSizeARB(gwidth, gheight, 1, side, side, 1);

			// Map buffeers
			glBindBuffer(GL_ATOMIC_COUNTER_BUFFER, count_buf);
			glBindBuffer(GL_SHADER_STORAGE_BUFFER, result_buf);
			GLuint *count = glMapBuffer(GL_ATOMIC_COUNTER_BUFFER, GL_READ_WRITE);
			if (vgl_perror()) return 1;
			if (*count) {
				GLSLivec3 *result = glMapBufferRange(GL_SHADER_STORAGE_BUFFER, 0, *count * sizeof *result, GL_MAP_READ_BIT);
				if (vgl_perror()) return 1;

				// Read computed values
				for (GLuint i = 0; i < *count; i++) {
					struct cluster clus = {result[i].x, result[i].y, result[i].z};
					param->cb(clus, param->data);
				}
				*count = 0;

				// Unmap buffers
				glUnmapBuffer(GL_SHADER_STORAGE_BUFFER);
				glBindBuffer(GL_SHADER_STORAGE_BUFFER, 0);
			}
			glUnmapBuffer(GL_ATOMIC_COUNTER_BUFFER);
			glBindBuffer(GL_ATOMIC_COUNTER_BUFFER, 0);

			if (vgl_perror()) return 1;
 
			pos.z += gheight;
		}

		pos.x += gwidth;
	}

	return 0;
}
