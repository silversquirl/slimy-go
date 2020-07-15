# slimy

Slimy is a tool to find slime chunk clusters quickly. It searches Minecraft worlds
to find perimeter locations containing large or small amounts of slime chunks.

## Usage

`slimy [-j NUM_THREADS] [-g] SEED RANGE THRESHOLD`

- `-j NUM_THREADS` - Sets the number of threads to use in CPU mode [Default: number of CPU cores]
- `-g` - Switches to GPU mode

`SEED` specifies the world seed to search.

`RANGE` specifies the range to search. The searched area will be a square with a
side length of 2 times this value.

`THRESHOLD` specifies the conditions by which a cluster is considered successful.
If positive, the cluster must contain at least `THRESHOLD` chunks. If negative, the
cluster must contain no more than `-THRESHOLD` chunks.

## Building

The recommended way to build slimy is using [cinsh]. This will ensure reproducible
results, as all builds are run inside containers. To build with cinsh, run one of
the following commands:

```
cinsh build     # Build both Linux and Windows binaries
cinsh build-lin # Build only the Linux binary
cinsh build-win # Build only the Windows binary
```

If you cannot use cinsh for some reason, eg. compiling on a non-Linux machine, you
can use make instead. To build with make, run one of the following commands:

```
make     # Build both Linux and Windows binaries
make lin # Build only the Linux binary
make win # Build only the Windows binary
```

To build CPU-only binaries, which does not require GLEW or GLFW3, add `NOGPU=1` to
the end of one of the above `make` commands.

[cinsh]: https://github.com/vktec/cinsh
