ifeq ($(CC),cc)
CC := clang
endif

MINGW_CC := i686-w64-mingw32-gcc
CC += -std=c99 -pedantic
CFLAGS := -Wall -Ofast
LDFLAGS := -lpthread

SRC := $(wildcard *.c)
HDR := $(wildcard *.h)

ifdef NOGPU
SRC := $(filter-out gpu.c,$(SRC))
else
CFLAGS += -DENABLE_GPU
LDFLAGS += -lGL -lGLEW -lglfw
HDR += shader.glsl.h
endif

slimy: $(SRC) $(HDR)
	$(CC) $(CFLAGS) -o $@ $(SRC) $(LDFLAGS)

slimy.exe: $(SRC) $(HDR)
	$(MINGW_CC) $(CFLAGS) -o $@ $(SRC) $(LDFLAGS)

%.glsl.h: %.glsl
	python -c"print('char $(patsubst %.glsl,%_glsl,$<)[]={'+','.join(map(hex,map(ord,open('$<').read())))+',0};')" >$@

.PHONY: clean exe
clean:
	rm -f slimy slimy.exe
	rm -f *.glsl.h
