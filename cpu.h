#ifndef SLIMY_CPU_H
#define SLIMY_CPU_H

#include <stdint.h>
#include "common.h"

struct threadparams {
	thrd_t thr;
	struct searchparams *common;
	struct chunkpos start, end;
};

_Bool is_slimy(int64_t seed, int32_t x, int32_t z);
THREAD_RET cpu_search_strips(void *data);
int cpu_search(struct searchparams *param, int nthread);

#endif
