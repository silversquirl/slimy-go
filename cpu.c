#include <stdio.h>
#include <stdint.h>
#include "cpu.h"
#include "threadshim.h"

static uint64_t lcg_init(uint64_t seed) {
	return (seed ^ 0x5DEECE66Dll) & ((1ll << 48) - 1);
}

static int32_t lcg_next(uint64_t *seed, int bits) {
	*seed = (*seed * 0x5DEECE66Dll + 0xBll) & ((1ll << 48) - 1);
	return *seed >> (48 - bits);
}

static int32_t lcg_next_int(uint64_t *seed, int32_t max) {
	int32_t bits, val;
	do {
		bits = lcg_next(seed, 31);
		val = bits % max;
	} while (bits - val + (max-1) < 0);
	return val;
}

_Bool is_slimy(int64_t world_seed, int32_t x, int32_t z) {
	uint64_t seed =
		world_seed +
		x * x * 4987142ll +
		x * 5947611ll +
		z * z * 4392871ll +
		z * 389711ll;
	seed ^= 987234911ll;
	seed = lcg_init(seed);
	return !lcg_next_int(&seed, 10);
}

static inline int32_t roundup_pow2(int32_t n) {
	n--;
	for (int32_t i = 0; i < 5; i++) {
		n |= n >> (1<<i);
	}
	n++;
	return n;
}

static int32_t isqrt(int32_t n) {
	static int32_t lut[72] = {
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
	int32_t x = 1 << ((__builtin_clz(n) ^ 31) / 2);
#else
	int32_t x = n;
#endif

	for (;;) {
		x = (x + n/x) / 2;
		if (x*x <= n && (x+1)*(x+1) > n) return x;
	}
}

static inline _Bool check_threshold(int count, int thres) {
	return thres < 0 ? count <= -thres : count >= thres;
}

THREAD_RET search_strips(void *data) {
	struct threadparams *param = data;
	int orad = param->common->outer_rad;
	int orad2 = orad*orad;
	int irad = param->common->inner_rad;
	int irad2 = irad*irad;
	int side = 2*orad+1; // Side-length of bounding box of outer radius

	int bufn = roundup_pow2(side); // Number of entries in ring buffer
	_Bool buf[bufn * side]; // Ring buffer to store cached sliminess
	unsigned bufp = 0; // Pointer in ring buf

#define buf_wrap(zpos) ((zpos) & (bufn-1))
#define buf_at(zpos, xpos) (buf[side*buf_wrap(bufp - orad + zpos) + orad + xpos])
#define buf_at_p(xpos) (buf[side*bufp + orad + xpos])
#define buf_loadrow(x, z) do { \
		bufp = (bufp+1) & (bufn-1); \
		for (int k = -orad; k <= orad; k++) { \
			buf_at_p(k) = is_slimy(param->common->seed, (x) + k, (z)); \
		} \
	} while (0)

	// Circle info
	int owidths[side];
	int iwidths[side];
	for (int cz = -orad; cz <= orad; cz++) {
		int cz2 = cz*cz;
		owidths[cz+orad] = isqrt(orad2 - cz2);
		iwidths[cz+orad] = irad2 < cz2 ? 0 : isqrt(irad2 - cz2);
	}

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
				int owidth = owidths[cz+orad];
				int iwidth = iwidths[cz+orad];

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

int begin_search(struct searchparams *param, int nthread) {
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

		if (thrd_create(&threads[i].thr, search_strips, &threads[i]) != thrd_success) {
			fprintf(stderr, "Error starting thread %d\n", i);
			return 1;
		}
	}

	for (int i = 0; i < nthread; i++) {
		if (thrd_join(threads[i].thr, NULL) != thrd_success) {
			fprintf(stderr, "Error joining thread %d\n", i);
		}
		thrd_close(threads[i].thr);
	}

	return 0;
}
