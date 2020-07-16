.PHONY: all clean lin linux win windows

BUILDDIR ?= build

all: lin win
clean:
	rm -rf $(BUILDDIR)/lin/ $(BUILDDIR)/win/
	-rmdir $(BUILDDIR)/

lin linux:
	BUILDDIR=$(BUILDDIR)/lin $(MAKE) -f linux.mk

win windows:
	BUILDDIR=$(BUILDDIR)/win $(MAKE) -f windows.mk
