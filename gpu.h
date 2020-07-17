#ifndef SLIMY_GPU_H
#define SLIMY_GPU_H

#include <stdint.h>
#include "common.h"

struct gpuparam {
	struct searchparams *param;
	GLuint groupw, groupr, collw;
	GLuint slime_prog, mask_prog;
	GLuint slime_buf, mask_buf, result_buf, count_buf;
};

int gpu_init(void);
int gpu_init_param(struct gpuparam *gparam, struct searchparams *param);
void gpu_del_param(struct gpuparam *gparam);
int gpu_search(struct gpuparam *gparam);

#endif
