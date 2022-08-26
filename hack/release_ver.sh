#!/bin/bash

set -eu
set -o pipefail

LATEST_RELEASE_TAG=$(git tag -l "v*" | tail -n 1)
COMMITS_SINCE_RELEASE=$([[ $(git rev-list -n 1 $LATEST_RELEASE_TAG) == $(git rev-parse HEAD) ]]; echo $?)
MODIFICATIONS_PRESENT=$([[ -z $(git status --porcelain --untracked-files=no) ]]; echo $?)
DEV_MANIFEST=$([[ !COMMITS_SINCE_RELEASE && !MODIFICATIONS_PRESENT ]]; echo $?)
RELEASE_VERSION=$LATEST_RELEASE_TAG
if [[ $DEV_MANIFEST ]]; then # Release has been modified.
    RELEASE_VERSION=$(sed -E 's/-rc[0-9]+//' <<< $RELEASE_VERSION) # Remove potencial release candidate tagging.
    RELEASE_VERSION=$(awk -vFS=. -vOFS=. '{$NF++;print}' <<< $RELEASE_VERSION)"-dev" # Increment the versioning and add dev tag.
fi
echo $RELEASE_VERSION

