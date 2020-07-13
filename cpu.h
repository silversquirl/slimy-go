#ifndef SLIMY_CPU_H
#define SLIMY_CPU_H

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

struct threadparams {
	thrd_t thr;
	struct searchparams *common;
	struct chunkpos start, end;
};

_Bool is_slimy(int64_t seed, int32_t x, int32_t z);
THREAD_RET search_strips(void *data);
int begin_search(struct searchparams *param, int nthread);

#endif
