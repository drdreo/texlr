#!/usr/bin/env sh
# Build the kitchen-sink fixture and rasterize every PDF page to PNG so the
# rendered output can be screenshot-tested. Requires the Texlr toolchain on
# PATH (tectonic, dot, mmdc, gs, go) — run inside `nix develop`:
#
#   nix develop -c ./scripts/screenshot-test.sh
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
doc_dir="$root/testdata/kitchen-sink"
out="$root/artifacts/kitchen-sink"

# The ordinary-image figure needs a PNG; generate it so no binary is committed.
mkdir -p "$doc_dir/images"
if [ ! -f "$doc_dir/images/sample.png" ]; then
  dot -Tpng -Gdpi=144 -o "$doc_dir/images/sample.png" \
    "$doc_dir/diagrams/sample-image.dot"
fi

go run "$root/cmd/texlr" build "$doc_dir/kitchen-sink.tex" \
  --pdf "$out/kitchen-sink.pdf" \
  --source "$out/source" \
  --log "$out/kitchen-sink.log" \
  --force

rm -rf "$out/pages"
mkdir -p "$out/pages"
gs -q -dSAFER -dBATCH -dNOPAUSE -sDEVICE=png16m -r150 \
  -o "$out/pages/page-%02d.png" "$out/kitchen-sink.pdf"

echo "pdf:   $out/kitchen-sink.pdf"
echo "pages: $out/pages"
