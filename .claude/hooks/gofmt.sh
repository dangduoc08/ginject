#!/usr/bin/env sh
file="$(jq -r '.tool_input.file_path // empty')"

case "$file" in
  *.go) ;;
  *) exit 0 ;;
esac

[ -f "$file" ] || exit 0

if ! out="$(gofmt -l -w "$file" 2>&1)"; then
  echo "gofmt failed on $file: $out" >&2
  exit 2
fi

exit 0
