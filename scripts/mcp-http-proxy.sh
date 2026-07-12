#!/bin/bash
url="$1"
shift
headers=()
while [ $# -gt 0 ]; do
  case "$1" in
    --header) shift; headers+=(-H "$1");;
  esac
  shift
done

while IFS= read -r line; do
  [ -z "$line" ] && continue
  echo "$(curl -s -X POST "$url" -H "Content-Type: application/json" -H "Accept: application/json" "${headers[@]}" -d "$line")"
done
