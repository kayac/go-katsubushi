#!/bin/sh

set -ue
dir="$1"
file="$2"
cd "$dir"
zip "../${file}-${dir}.zip" "$file"
