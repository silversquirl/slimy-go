CC := clang -std=c99 -pedantic
CFLAGS := -Wall -Ofast
LDFLAGS := -lpthread
slimy: slimy.c

.PHONY: clean
clean:
	rm -f slimy
