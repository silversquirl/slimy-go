#include <stdlib.h>
#include <stdio.h>

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

_Bool is_slimy(long seed, int posx, int posy) {
	seed +=
		posx * posx * 4987142L +
		posx * 5947611L +
		posy * posy * 4392871L +
		posy * 389711L;
	seed ^= 987234911L;
	return lcg_next_int(&seed, 10) == 0;
}

int main(int argc, char *argv[]) {
	if (argc != 4) return 1;
	long seed = atol(argv[1]);
	int posx = atoi(argv[2]);
	int posy = atoi(argv[3]);
	printf("%s\n", is_slimy(seed, posx, posy) ? "yes" : "no");
	return 0;
}
