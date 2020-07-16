DEFAULT_CC := clang
LDFLAGS += -lpthread
include common.mk

ifndef NOGPU
CFLAGS += $(shell pkg-config --cflags glfw3)
LDFLAGS += $(shell pkg-config --libs glfw3) -lGL -ldl
endif

$(BUILDDIR)/slimy: $(OBJ)
	@mkdir -p $(dir $@)
	$(CC) -o $@ $^ $(LDFLAGS)