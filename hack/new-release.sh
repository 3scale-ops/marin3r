#!/bin/bash

REPO="3scale/marin3r"

# Skip if alpha release
[[ ${1} == *"-alpha"* ]] || [[ ${1} == *"-dev"* ]] && echo "" && exit 0

# Skip if release already exists
curl -o /dev/null --fail --silent "https://api.github.com/repos/${REPO}/releases/tags/${1}" && echo "" && exit 0

echo ${1}