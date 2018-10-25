#!/bin/sh
for f in /usr/bin/*; do
  ./elfinfo "$f"
done
