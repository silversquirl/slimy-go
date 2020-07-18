#define _POSIX_C_SOURCE 200809L
#include <inttypes.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>
#include "cpu.h"

#ifdef ENABLE_GPU
#include "lib/glad.h"
#include "gpu.h"
#endif

struct clusterbuf {
	size_t len, alloc;
	struct cluster *buf;
};

static inline _Bool inorder(struct cluster a, struct cluster b) {
	long ax = a.x, az = a.z;
	long bx = b.x, bz = b.z;
	// Sort by count, then by distance from origin
	if (a.count > b.count) return 1;
	if (a.count < b.count) return 0;
	return ax*ax + az*az <= bx*bx + bz*bz;
}

static void collect_cb(struct cluster clus, int threadid, void *data) {
	struct clusterbuf *buf = data;
	buf += threadid;

	if (buf->len >= buf->alloc) {
		size_t alloc = buf->alloc * 2;
		void *mem = realloc(buf->buf, alloc * sizeof *buf->buf);
		if (!mem) {
			fprintf(stderr, "Error allocating memory for buffer on thread %d\n", threadid);
			return;
		}
		buf->alloc = alloc;
		buf->buf = mem;
	}

	size_t i;
	for (i = buf->len; i > 0; i--) {
		if (inorder(buf->buf[i-1], clus)) {
			break;
		} else {
			buf->buf[i] = buf->buf[i-1];
		}
	}
	buf->buf[i] = clus;
	buf->len++;
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
		"Usage: slimy [-o FILENAME.csv] [-j NUM_THREADS] "
#ifdef ENABLE_GPU
		"[-g] "
#endif
		"SEED RANGE THRESHOLD\n", f);
}

int main(int argc, char *argv[]) {
	FILE *csv = NULL;
	int nthread = 0;
	enum {
		MODE_CPU,
#ifdef ENABLE_GPU
		MODE_GPU,
#endif
	} mode = MODE_CPU;

	const char *optstr =
		"ho:j:"
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

		case 'o':
			if (csv) fclose(csv);
			csv = fopen(optarg, "w");
			if (!csv) {
				fputs("Error opening file for writing: ", stderr);
				perror(optarg);
				return 1;
			}
			break;

		case 'j':
			nthread = atoi(optarg);
			if (!nthread) fprintf(stderr, "Invalid thread count: %s\n", optarg);
			break;

#ifdef ENABLE_GPU
		case 'g':
			mode = MODE_GPU;
			break;
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
		nthread = 1;
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

	struct clusterbuf results[nthread];
	for (int i = 0; i < nthread; i++) {
		results[i].len = 0;
		results[i].alloc = 8;
		results[i].buf = malloc(results[i].alloc * sizeof *results[i].buf);
		if (!results[i].buf) {
			fprintf(stderr, "Error allocating result buffer\n");
			return 1;
		}
	}

	struct searchparams param = {
		.seed = seed,
		.range = range,
		.threshold = thres,

		.outer_rad = 8,
		.inner_rad = 3,

		.cb = collect_cb,
		.data = &results,
	};

	switch (mode) {
	case MODE_CPU:
		if (cpu_search(&param, nthread)) return 1;
		break;

#ifdef ENABLE_GPU
	case MODE_GPU:;
		struct gpuparam gparam;
		if (gpu_init_param(&gparam, &param)) return 1;
		int ret = gpu_search(&gparam);
		gpu_del_param(&gparam);
		if (ret) return ret;
		break;
#endif
	}

	if (csv) {
		fputs("Slime Count,Chunk X,Chunk Z\n", csv);
	}

	// Pad for human readable numbers
	int pad = 1;
	int v = range;
	while (v) pad++, v /= 10;

	int prev = 1<<30; ////////

	size_t idx[nthread];
	for (int i = 0; i < nthread; i++) idx[i] = 0;
	for (;;) {
		int i = 0;
		while (idx[i] >= results[i].len) {
			if (++i >= nthread) goto done;
		}

		int max = i;
		struct cluster maxc = results[max].buf[idx[max]];
		for (; i < nthread; i++) {
			if (idx[i] >= results[i].len) continue;
			struct cluster clus = results[i].buf[idx[i]];
			if (!inorder(maxc, clus)) {
				max = i;
				maxc = clus;
			}
		}

		idx[max]++;
		if (csv) {
			// TODO: keep a list of top 10 or something
			fprintf(csv, "%d,%d,%d\n", maxc.count, maxc.x, maxc.z);
		} else {
			printf("(%*d, %*d) \t%d chunk%s\n", pad, maxc.x, pad, maxc.z, maxc.count, maxc.count == 1 ? "" : "s");
		}

		if (maxc.count > prev) {
		#include <signal.h>
			printf("ERROR\n"); raise(SIGTRAP);
		}
		prev = maxc.count; ////////
	}
done:

	return 0;
}
