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

	void (*cb)(struct cluster clus, void *data);
	void *data;
};

int32_t isqrt(int32_t n);

#endif
