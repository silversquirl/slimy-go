# slimy

Slimy is a tool to find slime chunk clusters in Minecraft seeds.
It can search on either the CPU or the GPU, and makes use of multithreading to speed up the CPU search.

## Installation

Slimy is still in development, so there are no releases currently.
However, it is very easy to install Slimy from source code: simply install [Go], then run `go install github.com/vktec/slimy/cmd/slimy`

[Go]: https://golang.org/

## System requirements

Requirements for CPU search are minimal, though performance will suffer on less powerful CPUs.
GPU search requires support for OpenGL 4.2 or greater, with the `GL_ARB_compute_shader` and `GL_ARB_shader_storage_buffer_object` extensions.
For reference, most integrated GPUs since 2013 (or 2012 on Linux) will support these features.
