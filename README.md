<a align="right" href="https://github.com/xyproto/elfinfo"><img align="right" src="https://raw.githubusercontent.com/xyproto/elfinfo/master/web/elfinfo.png" style="margin-left: 2em" width="200px"></a>

# ELFinfo

Tiny program for emitting the most basic information about an ELF file.

Can detect the compiler used for compiling a given binary, even if it is stripped. The following languages/compilers are supported:

* GCC
* Clang
* FPC
* OCaml
* Go
* TCC (compiler name only, TCC does not store the version number in the executables)
* Rust (for stripped executables, only the compiler name and GCC version used for linking are available)
* GHC

## Installation

This requires Go 1.12 or later and will install the development version:

    go get -u github.com/xyproto/elfinfo

## Example usage

    $ elfinfo -c sh
    GCC 8.1.1

    $ elfinfo /usr/bin/ls
    /usr/bin/ls: stripped=true, compiler=GCC 8.2.0, byteorder=LittleEndian, machine=Advanced Micro Devices x86-64

## Distro Packages

[![Packaging status](https://repology.org/badge/vertical-allrepos/elfinfo.svg)](https://repology.org/project/elfinfo/versions)

## General info

* Version: 0.7.6
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
* License: MIT
