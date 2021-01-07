# slimy

Slimy is a tool to find slime chunk clusters in Minecraft seeds.
It can search on either the CPU or the GPU, and makes use of multithreading to speed up the CPU search.

## Installation

Slimy is still in development, so there are no releases currently.
However, it is very easy to install Slimy from source code: simply install [Go], then run `go install github.com/vktec/slimy/cmd/slimy`

[Go]: https://golang.org/

## System requirements

Requirements for CPU search are minimal, though performance will suffer on less powerful CPUs.
GPU search requires support for OpenGL 4.3 or greater, with the `GL_ARB_gpu_shader_int64` and `GL_ARB_compute_variable_group_size` extensions.
