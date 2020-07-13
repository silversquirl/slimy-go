ifeq ($(CC),cc)
CC := clang
endif

CC += -std=c99 -pedantic
CFLAGS := -Wall -Ofast
LDFLAGS := -lpthread

slimy$(EXE): slimy.c
	$(CC) $(CFLAGS) -o $@ $^ $(LDFLAGS)

.PHONY: clean exe
clean:
	rm -f slimy slimy.exe

exe:
	$(MAKE) -f Makefile.mingw
