#!/bin/bash

# Run all example scenarios in sequence
passed=()
failed=()

for d in */ ; do
  if [ -f "$d/run.sh" ] && [ -x "$d/run.sh" ]; then
    echo "Running $d"
    if (cd "$d" && CG_NO_PROMPT=1 ./run.sh); then
      passed+=("$d")
    else
      failed+=("$d")
    fi
  fi
done

echo "Passed tests:"
for test in "${passed[@]}"; do
  echo "  $test"
done

echo "Failed tests:"
for test in "${failed[@]}"; do
  echo "  $test"
done

if [ ${#failed[@]} -gt 0 ]; then
  exit 1
fi
