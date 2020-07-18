HOST ?= x86_64-w64-mingw32
DEFAULT_CC := $(HOST)-gcc
include common.mk

CFLAGS += -Wno-pedantic

ifndef NOGPU
LDFLAGS += -Wl,-Bstatic -lglfw3 -Wl,-Bdynamic -lopengl32 -lgdi32
endif

$(BUILDDIR)/slimy.exe: $(OBJ)
	@mkdir -p $(dir $@)
	$(CC) -o $@ $^ $(LDFLAGS)
