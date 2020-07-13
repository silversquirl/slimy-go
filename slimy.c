#define _POSIX_C_SOURCE 200809L
#include <limits.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <threads.h>
#include <unistd.h>

struct chunkpos {
	int x, z;
};

struct cluster {
	int x, z;
	int count;
};

struct searchparams {
	long seed;
	int range;
	int threshold; // positive for above, negative for below

	int outer_rad, inner_rad;

	void (*cb)(struct cluster clus, void *data);
	void *data;
};

struct threadparams {
	thrd_t thr;
	struct searchparams *common;
	struct chunkpos start, end;
};

long lcg_init(long seed) {
	return (seed ^ 0x5DEECE66DL) & ((1L << 48) - 1);
}

int lcg_next(long *seed, int bits) {
	*seed = (*seed * 0x5DEECE66DL + 0xBL) & ((1L << 48) - 1);
	return *seed >> (48 - bits);
}

int lcg_next_int(long *seed, int max) {
	int bits, val;
	do {
		bits = lcg_next(seed, 31);
		val = bits % max;
	} while (bits - val + (max-1) < 0);
	return val;
}

_Bool is_slimy(long seed, int x, int z) {
	seed +=
		x * x * 4987142L +
		x * 5947611L +
		z * z * 4392871L +
		z * 389711L;
	seed ^= 987234911L;
	seed = lcg_init(seed);
	return !lcg_next_int(&seed, 10);
}

static inline int roundup_pow2(int n) {
	n--;
	for (int i = 0; i < 5; i++) {
		n |= n >> (1<<i);
	}
	n++;
	return n;
}

#define INT_BIT (CHAR_BIT * sizeof (int))
static int isqrt(int n) {
	static int lut[72] = {
		0, 1, 1, 1, 2, 2, 2, 2,
		2, 3, 3, 3, 3, 3, 3, 3,
		4, 4, 4, 4, 4, 4, 4, 4,
		4, 5, 5, 5, 5, 5, 5, 5,
		5, 5, 5, 5, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6,
		6, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7,
		8, 8, 8, 8, 8, 8, 8, 8,
	};
	if (n < sizeof lut / sizeof *lut) return lut[n];

	// LUT miss, fallback to Newton's method
#ifdef __GNUC__
	int x = 1 << ((__builtin_clz(n) ^ INT_BIT-1) / 2);
#else
	int x = n;
#endif

	for (;;) {
		x = (x + n/x) / 2;
		if (x*x <= n && (x+1)*(x+1) > n) return x;
	}
}

static inline _Bool check_threshold(int count, int thres) {
	return thres < 0 ? count <= -thres : count >= thres;
}

int search_strips(void *data) {
	struct threadparams *param = data;
	int orad = param->common->outer_rad;
	int orad2 = orad*orad;
	int irad = param->common->inner_rad;
	int irad2 = irad*irad;
	int side = 2*orad+1; // Side-length of bounding box of outer radius

	int bufn = roundup_pow2(side); // Number of entries in ring buffer
	_Bool buf[bufn * side]; // Ring buffer to store cached sliminess
	unsigned bufp = 0; // Pointer in ring buf

#define buf_wrap(zpos) ((zpos) & bufn-1)
#define buf_at(zpos, xpos) (buf[side*buf_wrap(bufp - orad + zpos) + orad + xpos])
#define buf_at_p(xpos) (buf[side*bufp + orad + xpos])
#define buf_loadrow(x, z) do { \
		bufp = bufp+1 & bufn-1; \
		for (int k = -orad; k <= orad; k++) { \
			buf_at_p(k) = is_slimy(param->common->seed, (x) + k, (z)); \
		} \
	} while (0)

	for (int x = param->start.x; x < param->end.x; x++) {
		// Pre-cache a full circle's box
		for (int z = param->start.z - orad; z < param->start.z + orad; z++) {
			buf_loadrow(x, z);
		}

		for (int z = param->start.z; z < param->end.z; z++) {
			buf_loadrow(x, z + orad);

			// Count slime chunks
			int count = 0;
			for (int cz = -orad; cz <= orad; cz++) {
				int cz2 = cz*cz;
				// TODO: precompute these isqrts
				// Outer circle width at current pos
				int owidth = isqrt(orad2 - cz2);
				// Inner circle width at current pos
				int iwidth = irad2 <= cz2 ? 0 : isqrt(irad2 - cz2);

#ifdef PRETTY_PICTURES
				for (int cx = -orad; cx <= orad; cx++) {
					if (cx > owidth || -cx > owidth) {
						printf("  ");
					} else if (cx < iwidth && -cx < iwidth) {
						printf("  ");
					} else {
						printf("%c ", buf_at(cz, cx) ? 'x' : '.');
					}
				}
				putchar('\n');
#endif

				for (int cx = iwidth; cx <= owidth; cx++) {
					count += buf_at(cz, cx);
					if (cx) count += buf_at(cz, -cx);
				}
			}

			if (check_threshold(count, param->common->threshold)) {
				param->common->cb((struct cluster){x, z, count}, param->common->data);
			}
		}
	}

	return 0;
}

void begin_search(struct searchparams *param, int nthread) {
	int startx = -param->range, endx = param->range;
	int startz = startx, endz = endx;
	int xrange = 2*param->range;

	int posx = startx;
	int xstep = xrange / nthread;
	int xstep0 = xrange % nthread;

	struct threadparams threads[nthread];
	for (int i = 0; i < nthread; i++) {
		threads[i].common = param;
		threads[i].start = (struct chunkpos){posx, startz};

		posx += xstep;
		if (!i) posx += xstep0;

		threads[i].end = (struct chunkpos){posx, endz};

		thrd_create(&threads[i].thr, search_strips, &threads[i]);
	}

	for (int i = 0; i < nthread; i++) {
		thrd_join(threads[i].thr, NULL);
	}
}

static void print_cb(struct cluster clus, void *data) {
	printf("(%d, %d)  %d chunk%s\n", clus.x, clus.z, clus.count, clus.count == 1 ? "" : "s");
}

#ifdef _POSIX_VERSION
static int nproc(void) {
	return sysconf(_SC_NPROCESSORS_ONLN);
}
#elif 0 && defined(_WIN32)
// TODO: check and enable Windows support
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

static int java_string_hash(const char *str) {
	size_t len = strlen(str);
	unsigned hash = 0;
	long coef = 1;
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
	long seed = strtol(argv[optind+0], &end, 10);
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
	printf("  Seed:       %ld\n", seed);
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

	begin_search(&param, nthread);

	return 0;
}
