#ifndef SLIMY_GPU_H
#define SLIMY_GPU_H

#include <stdint.h>
#include "common.h"

int gpu_init(void);
int gpu_search(struct searchparams *param);

#endif
