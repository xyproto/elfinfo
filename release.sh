#!/bin/sh
ver=$(git describe --tags | cut -dv -f2)
echo "Version: $ver"
mkdir -p "elfinfo-$ver"
cp -rv main.go LICENSE README.md vendor go.mod go.sum "elfinfo-$ver"
tar Jcvf "elfinfo-$ver.tar.xz" "elfinfo-$ver"
