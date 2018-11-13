#!/bin/sh
ver=0.7.3
mkdir -p "elfinfo-$ver"
cp -rv main.go LICENSE README.md machine "elfinfo-$ver"
tar Jcvf "elfinfo-$ver.tar.xz" "elfinfo-$ver"
