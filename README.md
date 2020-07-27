# ELFinfo

<a align="center" href="https://github.com/xyproto/elfinfo"><img alt="ELFinfo logo" src="https://raw.githubusercontent.com/xyproto/elfinfo/master/web/elfinfo.png" width="200px"></a>

[![Build Status](https://travis-ci.com/xyproto/elfinfo.svg?branch=master)](https://travis-ci.com/xyproto/elfinfo) [![License](http://img.shields.io/badge/license-MIT-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/elfinfo/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/elfinfo)](https://goreportcard.com/report/github.com/xyproto/elfinfo)

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

This requires Go 1.11 or later and will install the development version:

    go get -u github.com/xyproto/elfinfo

## Example usage

    $ elfinfo sh
    GCC 10.1.0

    $ elfinfo -l /usr/bin/ls
    /usr/bin/ls: stripped=true, compiler=GCC 9.2.1, static=false, byteorder=LE, machine=Advanced Micro Devices x86-64

## Distro Packages

[![Packaging status](https://repology.org/badge/vertical-allrepos/elfinfo.svg)](https://repology.org/project/elfinfo/versions)

## General info

* Version: 1.1.0
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
* License: MIT
