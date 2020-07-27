#!/bin/sh
CGO_ENABLED=0 GOOS=linux go build -mod=vendor -v -trimpath -ldflags "-s" -a -o elfinfo
