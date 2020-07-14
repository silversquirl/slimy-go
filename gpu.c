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

// GLSL's bool is the same size as uint, and GLboolean isn't
typedef GLuint GLSLbool;

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
	if (groupw > 0x4000) {
		groupw = 0x4000;
		collw = groupw / searchw;
		groupr = groupw % searchw;
		if (groupr) collw++;
	}

	// Preprocess
	char *lsx = strstr(shader_glsl, "LSIZEX");
	char *lsy = strstr(shader_glsl, "LSIZEY");
	lsx[snprintf(lsx, 6, "%5d", side)] = ' ';
	lsy[snprintf(lsy, 6, "%5d", side)] = ' ';

	GLuint prog = build_program("shader.glsl", shader_glsl);

	// Revert preprocessing
	memcpy(lsx, "LSIZEX", 6);
	memcpy(lsy, "LSIZEY", 6);

	if (!prog) return 1;

	// Load GPU parameters
	glUseProgram(prog);
	glUniform1i64ARB(1, param->seed);
	glUniform1ui(3, orad);

	GLuint bufs[2];
	glGenBuffers(2, bufs);
	GLuint mask_buf = bufs[0], count_buf = bufs[1];

	// Load mask
	glBindBuffer(GL_SHADER_STORAGE_BUFFER, mask_buf);
	glBufferData(GL_SHADER_STORAGE_BUFFER, mask_size, mask, GL_STATIC_DRAW);
	glBindBufferBase(GL_SHADER_STORAGE_BUFFER, 1, mask_buf);

	// Allocate count buffer
	size_t count_len = groupw * groupw;
	size_t count_size = count_len * sizeof (GLuint);
	glBindBuffer(GL_SHADER_STORAGE_BUFFER, count_buf);
	glBufferData(GL_SHADER_STORAGE_BUFFER, count_size, NULL, GL_STREAM_READ);
	glBindBufferBase(GL_SHADER_STORAGE_BUFFER, 2, count_buf);

	glBindBuffer(GL_SHADER_STORAGE_BUFFER, 0);

	glUseProgram(prog);

	struct chunkpos pos = {-param->range, -param->range};
	for (GLuint collx = 0; collx < collw; collx++) {
		GLuint gwidth = groupw;
		if (!collx && groupr) gwidth = groupr;

		for (GLuint colly = 0; colly < collw; colly++) {
			GLuint gheight = groupw;
			if (!colly && groupr) gheight = groupr;

			glUniform2i(2, pos.x - orad, pos.z - orad);
			glDispatchCompute(gwidth, gheight, 1);

			// Read output
			// TODO: consider doing this in a shader. Requires count buffer to be GL_STREAM_COPY
			glBindBuffer(GL_SHADER_STORAGE_BUFFER, count_buf);
			GLuint *count_data = glMapBuffer(GL_SHADER_STORAGE_BUFFER, GL_READ_ONLY);
			for (GLuint ox = 0; ox < gwidth; ox++) {
				for (GLuint oz = 0; oz < gheight; oz++) {
					GLuint count = count_data[oz*groupw + ox];
					if (check_threshold(count, param->threshold)) {
						int x = pos.x + ox;
						int z = pos.z + oz;
						param->cb((struct cluster){x, z, count}, param->data);
					}
				}
			}
			glUnmapBuffer(GL_SHADER_STORAGE_BUFFER);

			pos.z += gheight;
		}

		pos.x += gwidth;
	}

	return 0;
}
