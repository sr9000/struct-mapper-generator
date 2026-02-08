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

if [ ${#failed[@]} -eq 0 ]; then
  echo "All usecases passed:"
  for test in "${passed[@]}"; do
    echo "  $test"
  done
else
  echo "Passed tests:"
  for test in "${passed[@]}"; do
    echo "  $test"
  done

  echo "Failed tests:"
  for test in "${failed[@]}"; do
    echo "  $test"
  done

  exit 1
fi
