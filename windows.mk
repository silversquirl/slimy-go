HOST ?= x86_64-w64-mingw32
DEFAULT_CC := $(HOST)-gcc
include common.mk

ifndef NOGPU
CFLAGS += -DGLEW_STATIC
LDFLAGS += -Wl,-Bstatic -lglew32 -lglfw3 -Wl,-Bdynamic -lopengl32 -lglu32 -lgdi32
endif

$(BUILDDIR)/slimy.exe: $(OBJ)
	@mkdir -p $(BUILDDIR)
	$(CC) -o $@ $^ $(LDFLAGS)
