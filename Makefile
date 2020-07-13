ifeq ($(CC),cc)
CC := clang
endif

MINGW_CC := i686-w64-mingw32-gcc
CC += -std=c99 -pedantic
CFLAGS := -Wall -Ofast
LDFLAGS := -lpthread

SRC := $(wildcard *.c)
HDR := $(wildcard *.h)

slimy: $(SRC) $(HDR)
	$(CC) $(CFLAGS) -o $@ $(SRC) $(LDFLAGS)

slimy.exe: $(SRC) $(HDR)
	$(MINGW_CC) $(CFLAGS) -o $@ $(SRC) $(LDFLAGS)

.PHONY: clean exe
clean:
	rm -f slimy slimy.exe
