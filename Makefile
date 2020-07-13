CC := clang -std=c99 -pedantic
CFLAGS := -Wall
LDFLAGS :=
slimy: slimy.c

.PHONY: clean
clean:
	rm -f slimy
