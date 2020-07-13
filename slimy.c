#define _POSIX_C_SOURCE 200809L
#include <inttypes.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>
#include "cpu.h"

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
	fputs("Usage: slimy [-j NUM_THREADS] SEED RANGE THRESHOLD\n", f);
}

int main(int argc, char *argv[]) {
	int nthread = 0;

	int opt;
	while ((opt = getopt(argc, argv, "hj:")) >= 0) {
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
		}
	}

	if (!nthread) nthread = nproc() / 2;

	if (argc - optind != 3) {
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

	putchar('\n');
	printf("  Seed:       %"PRIi64"\n", seed);
	printf("  Range:      %d\n", range);
	printf("  Threshold: %c%d\n", thres < 0 ? '<' : '>', thres < 0 ? -thres : thres);
	printf("  Threads:    %d\n", nthread);
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

	return begin_search(&param, nthread);
}
