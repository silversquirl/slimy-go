BUILDDIR ?= build

ifeq ($(CC),cc)
CC := $(DEFAULT_CC)
endif

CC += -std=c99 -pedantic
CFLAGS += -Wall -Ofast -I$(BUILDDIR)

SRC := $(wildcard *.c)
HDR := $(wildcard *.h)

ifdef NOGPU
$(info nogpu)
SRC := $(filter-out gpu.c,$(SRC))
else
CFLAGS += -DENABLE_GPU
HDR += $(patsubst %,$(BUILDDIR)/%.glsl.h,mask slime)
endif

OBJ := $(SRC:%.c=$(BUILDDIR)/%.o)

$(BUILDDIR)/%.o: %.c $(HDR)
	@mkdir -p $(BUILDDIR)
	$(CC) $(CFLAGS) -c -o $@ $<

$(BUILDDIR)/%.glsl.h: %.glsl
	@mkdir -p $(BUILDDIR)
	python -c"print('char $(patsubst %.glsl,%_glsl,$<)[]={'+','.join(map(hex,map(ord,open('$<').read())))+',0};')" >$@
