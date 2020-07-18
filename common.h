#ifndef SLIMY_COMMON_H
#define SLIMY_COMMON_H

#include <stdint.h>
#include "threadshim.h"

struct chunkpos {
	int x, z;
};

struct cluster {
	int x, z;
	int count;
};

struct searchparams {
	int64_t seed;
	int range;
	int threshold; // positive for above, negative for below

	int outer_rad, inner_rad;

	void (*cb)(struct cluster clus, int threadid, void *data);
	void *data;
};

int32_t isqrt(int32_t n);

static inline int32_t roundup_pow2(int32_t n) {
	n--;
	for (int32_t i = 0; i < 5; i++) {
		n |= n >> (1<<i);
	}
	n++;
	return n;
}

static inline int32_t rounddn_pow2(int32_t n) {
	return roundup_pow2(n+1) / 2;
}

#ifdef DEBUG
#include <time.h>
#define BENCH_INIT() struct timespec _bench_time; const char *_bench_name;
#define BENCH_BEGIN(name) (_bench_name = (name), clock_gettime(CLOCK_MONOTONIC, &_bench_time))
#define BENCH_END() \
	do { \
		struct timespec _bench_end_time; \
		clock_gettime(CLOCK_MONOTONIC, &_bench_end_time); \
		double _bench_start_timef = _bench_time.tv_sec + _bench_time.tv_nsec/1e9; \
		double _bench_end_timef = _bench_end_time.tv_sec + _bench_end_time.tv_nsec/1e9; \
		fprintf(stderr, "BENCH: %s: %f\n", _bench_name, _bench_end_timef - _bench_start_timef); \
	} while (0)

#else

#define BENCH_INIT()
#define BENCH_BEGIN(name)
#define BENCH_END()
#endif

#endif
