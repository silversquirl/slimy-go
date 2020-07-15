.PHONY: all clean lin linux win windows

BUILDDIR ?= build

all: lin win
clean:
	rm -f $(BUILDDIR)/lin/slimy $(BUILDDIR)/win/slimy.exe
	rm -f $(BUILDDIR)/lin/*.o $(BUILDDIR)/lin/*.glsl.h
	rm -f $(BUILDDIR)/win/*.o $(BUILDDIR)/win/*.glsl.h
	-rmdir $(BUILDDIR)/win/ $(BUILDDIR)/lin/ $(BUILDDIR)/

lin linux:
	BUILDDIR=$(BUILDDIR)/lin $(MAKE) -f linux.mk

win windows:
	BUILDDIR=$(BUILDDIR)/win $(MAKE) -f windows.mk
