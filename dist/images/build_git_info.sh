#!/bin/bash

BRANCH=$(git rev-parse  --symbolic-full-name HEAD)
COMMIT=$(git rev-parse  HEAD)

echo "ref: ${BRANCH}  commit: ${COMMIT}" > git_info
