#!/bin/sh
ver=$(git describe --tags)
mkdir -p "elfinfo-$ver"
cp -rv main.go LICENSE README.md vendor go.mod go.sum testfiles "elfinfo-$ver"
tar Jcvf "elfinfo-$ver.tar.xz" "elfinfo-$ver"
