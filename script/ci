#!/bin/sh
BRANCH=${1:-$(git rev-parse --abbrev-ref HEAD)}
HASH=${2:-$(git rev-parse HEAD)}

git diff origin/$BRANCH...HEAD | manifest inspect --sha $HASH --formatter github
