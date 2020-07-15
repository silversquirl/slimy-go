#define _POSIX_C_SOURCE 200809L
#include <inttypes.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>
#include "cpu.h"

#ifdef ENABLE_GPU
#include <GL/glew.h>
#include "gpu.h"
#endif

static void print_cb(struct cluster clus, void *data) {
	printf("(%d, %d)  %d chunk%s\n", clus.x, clus.z, clus.count, clus.count == 1 ? "" : "s");
}

#ifdef _POSIX_VERSION
static int nproc(void) {
	return sysconf(_SC_NPROCESSORS_ONLN);
}
#elif defined(_WIN32)
#include <windows.h>
static int nproc(void) {
	SYSTEM_INFO sysinfo;
	GetSystemInfo(&sysinfo);
	return sysinfo.dwNumberOfProcessors;
}
#else
static int nproc(void) {
	fprintf(stderr, "Warning: Could not detect number of CPU cores, falling back to 1 thread.\n");
	return 1;
}
#endif

static int32_t java_string_hash(const char *str) {
	size_t len = strlen(str);
	uint32_t hash = 0;
	uint64_t coef = 1;
	while (len--) {
		hash += str[len] * coef;
		coef *= 31;
	}
	return hash;
}

static void usage(FILE *f) {
	fputs(
		"Usage: slimy [-j NUM_THREADS] "
#ifdef ENABLE_GPU
		"[-g] "
#endif
		"SEED RANGE THRESHOLD\n", f);
}

int main(int argc, char *argv[]) {
	int nthread = 0;
	enum {
		MODE_CPU,
#ifdef ENABLE_GPU
		MODE_GPU,
#endif
	} mode = MODE_CPU;

	const char *optstr =
		"hj:"
#ifdef ENABLE_GPU
		"g"
#endif
		;

	int opt;
	while ((opt = getopt(argc, argv, optstr)) >= 0) {
		switch (opt) {
		case '?':
			usage(stderr);
			return 1;
		case 'h':
			usage(stdout);
			return 0;
		case 'j':
			nthread = atoi(optarg);
			if (!nthread) fprintf(stderr, "Invalid thread count: %s\n", optarg);
			break;

#ifdef ENABLE_GPU
		case 'g':
			mode = MODE_GPU;
#endif
		}
	}

	if (argc - optind < 3) {
		usage(stderr);
		return 1;
	}

	char *end;
	int64_t seed = strtoll(argv[optind+0], &end, 10);
	if (*end) {
		seed = java_string_hash(argv[optind+0]);
	}

	int range = atoi(argv[optind+1]);
	if (!range) {
		fprintf(stderr, "Invalid range: %s\n", argv[optind+1]);
		return 1;
	}

	int thres = atoi(argv[optind+2]);
	if (!thres) {
		fprintf(stderr, "Invalid threshold: %s\n", argv[optind+2]);
		return 1;
	}

#ifdef ENABLE_GPU
	const GLubyte *gl_vers;
#endif

	switch (mode) {
	case MODE_CPU:
		if (!nthread) nthread = nproc();

		if (argc - optind != 3) {
			usage(stderr);
			return 1;
		}
		break;

#ifdef ENABLE_GPU
	case MODE_GPU:
		if (gpu_init()) return 1;
		gl_vers = glGetString(GL_VERSION);
		if (!gl_vers) {
			fprintf(stderr, "Could not get OpenGL version: %d\n", glGetError());
			return 1;
		}
		break;
#endif
	}

	putchar('\n');
	printf("  Seed:   %"PRIi64"\n", seed);
	printf("  Range:  %d\n", range);
	printf("  Thres: %c%d\n", thres < 0 ? '<' : '>', thres < 0 ? -thres : thres);
	switch (mode) {
	case MODE_CPU:
		printf("  Mode:   CPU (%d thread%s)\n", nthread, nthread == 1 ? "" : "s");
		break;

#ifdef ENABLE_GPU
	case MODE_GPU:
		printf("  Mode:   GPU (OpenGL %s)\n", gl_vers);
		break;
#endif
	}
	putchar('\n');

	struct searchparams param = {
		.seed = seed,
		.range = range,
		.threshold = thres,

		.outer_rad = 8,
		.inner_rad = 3,

		.cb = print_cb,
		.data = NULL,
	};

	switch (mode) {
	case MODE_CPU:
		return cpu_search(&param, nthread);

#ifdef ENABLE_GPU
	case MODE_GPU:
		return gpu_search(&param);
#endif
	}
}
