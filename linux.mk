DEFAULT_CC := clang
LDFLAGS += -lpthread
include common.mk

ifndef NOGPU
CFLAGS += $(shell pkg-config --cflags glew glfw3)
LDFLAGS += $(shell pkg-config --libs glew glfw3)
endif

$(BUILDDIR)/slimy: $(OBJ)
	@mkdir -p $(BUILDDIR)
	$(CC) -o $@ $^ $(LDFLAGS)
