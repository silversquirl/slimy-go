# slimy

[![Discord](https://img.shields.io/badge/chat%20on-discord-7289DA?logo=discord)](https://discord.gg/zEnfMVJqe6)

Slimy is a tool to find slime chunk clusters in Minecraft seeds.
It can search on either the CPU or the GPU, and makes use of multithreading to speed up the CPU search.

## Installation

Slimy is still in development and there are no full releases yet, however you can download the builds of the latest commit for [Linux] or [Windows].
Alternatively, it is quite easy to install Slimy from source code: install [Go], GLFW and a C compiler, then run `go get github.com/vktec/slimy/cmd/slimy`.

[Linux]: https://ci.vktec.org.uk/slimy/main/files/slimy-linux-amd64
[Windows]: https://ci.vktec.org.uk/slimy/main/files/slimy-windows-amd64.exe
[Go]: https://golang.org/

## System requirements

Requirements for CPU search are minimal, though performance will suffer on less powerful CPUs.
GPU search requires support for OpenGL 4.2 or greater, with the `GL_ARB_compute_shader` and `GL_ARB_shader_storage_buffer_object` extensions.
For reference, most integrated GPUs since 2013 (or 2012 on Linux) will support these features.
