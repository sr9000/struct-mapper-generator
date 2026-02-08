#!/bin/bash

# Run all example scenarios in sequence
for d in */ ; do
  if [ -f "$d/run.sh" ] && [ -x "$d/run.sh" ]; then
    echo "Running $d"
    (cd "$d" && CG_NO_PROMPT=1 ./run.sh)
  fi
done
