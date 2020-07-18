BUILDDIR ?= build

ifeq ($(CC),cc)
CC := $(DEFAULT_CC)
endif

CC += -std=c99 -pedantic
CFLAGS += -Wall -I$(BUILDDIR)

ifdef DEBUG
CFLAGS += -g
else
CFLAGS += -Ofast
endif

SRC := $(wildcard *.c) lib/glad.c
HDR := $(wildcard *.h)

ifdef NOGPU
SRC := $(filter-out gpu.c,$(SRC))
else
CFLAGS += -DENABLE_GPU
HDR += $(patsubst %,$(BUILDDIR)/%.glsl.h,mask slime)
endif

OBJ := $(SRC:%.c=$(BUILDDIR)/%.o)

$(BUILDDIR)/%.o: %.c $(HDR)
	@mkdir -p $(dir $@)
	$(CC) $(CFLAGS) -c -o $@ $<

$(BUILDDIR)/%.glsl.h: %.glsl
	@mkdir -p $(dir $@)
	python3 -c"print('char $(patsubst %.glsl,%_glsl,$<)[]={'+','.join(map(hex,map(ord,open('$<').read())))+',0};')" >$@
