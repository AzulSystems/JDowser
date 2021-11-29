#!/bin/bash

dev_version() {
  echo "development version from $(date +%D) (${SHA})"
  exit 0
}

ID=g$(git describe --abbrev=12 --always --dirty=+ --long 2>/dev/null)
[ -n "${ID}" ] || { echo "private build from $(date +%D)"; exit 0; }

SHA=$(echo ${ID:(-13)} | tr -d g)
echo "$ID" | grep -Eq "\+$" && dev_version "${ID}"

for T in $(git tag --points-at HEAD 2>/dev/null | sort --version-sort -r); do
  echo "$T" | grep -Eq "^v[0-9][0-9.]*[0-9]$" && { echo "$T (${SHA})"; exit 0; }
done

dev_version "${ID}"
