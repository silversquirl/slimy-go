#include "common.h"

int32_t isqrt(int32_t n) {
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
