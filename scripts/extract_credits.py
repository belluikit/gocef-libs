#!/usr/bin/env python3
"""Extract Chromium's third-party credits page from a CEF resources.pak.

Chromium ships the licences of everything it bundles as the about:credits
page (chrome://credits). CEF's binary distribution has no separate credits
file, but the page is inside resources.pak, compressed. This finds it and
writes it out as a standalone HTML file, so a payload can carry its
third-party notices beside LICENSE.cef.txt.

    scripts/extract_credits.py <resources.pak> <CREDITS.html>

Pak format v5 (ui/base/resource/data_pack.cc):
    u32 version (5), u32 encoding, u16 resource_count, u16 alias_count,
    (resource_count + 1) x {u16 id, u32 offset}   -- last entry is a sentinel
    alias_count x {u16 id, u16 entry_index}
Each resource is stored raw, gzip-compressed (starts 1f 8b), or
brotli-compressed with Chromium's own header (ui/base/resource/
resource_bundle.cc): the bytes 1e 9b, then the decompressed size as a
6-byte little-endian integer, then the brotli stream.

Python's standard library has gzip but no brotli, so brotli entries are
decoded by the `brotli` command-line tool (Debian/Ubuntu package `brotli`,
Homebrew `brotli`). Nothing else is needed.

The page is identified by content, not by resource ID, because IDs are
assigned at Chromium build time and change between versions. It is the
entry whose <title> is "Credits" and which carries the about:credits
markup. Exactly one must match; anything else is an error, so a Chromium
change shows up as a failed mirror rather than a missing or wrong file.
"""

import gzip
import re
import shutil
import struct
import subprocess
import sys

BROTLI_MAGIC = b"\x1e\x9b"
BROTLI_HEADER = 8
GZIP_MAGIC = b"\x1f\x8b"

TITLE_RE = re.compile(rb"<title>\s*Credits\s*</title>", re.IGNORECASE)
# Each credited project is a <div class="product"> in about_credits.tmpl.
PRODUCT_RE = re.compile(rb'class="product"')


def entries(data):
    version, _encoding, count, _aliases = struct.unpack_from("<IIHH", data, 0)
    if version != 5:
        sys.exit("unsupported pak version %d (this reads v5)" % version)
    table = [struct.unpack_from("<HI", data, 12 + 6 * i) for i in range(count + 1)]
    for i in range(count):
        rid, start = table[i]
        end = table[i + 1][1]
        yield rid, data[start:end]


def brotli_decode(blob):
    tool = shutil.which("brotli")
    if tool is None:
        sys.exit("a brotli-compressed resource needs the `brotli` command (apt install brotli)")
    expected = int.from_bytes(blob[2:BROTLI_HEADER], "little")
    out = subprocess.run(
        [tool, "--decompress", "--stdout"],
        input=blob[BROTLI_HEADER:],
        capture_output=True,
        check=True,
    ).stdout
    if len(out) != expected:
        sys.exit("brotli: header says %d bytes, decoded %d" % (expected, len(out)))
    return out


def decode(blob):
    if blob[:2] == GZIP_MAGIC:
        return gzip.decompress(blob)
    if blob[:2] == BROTLI_MAGIC:
        return brotli_decode(blob)
    return blob


def might_be_credits(blob):
    # Cheap filter so only plausible entries are decompressed: the credits
    # page is a large HTML document, hundreds of KB even compressed.
    if blob[:2] in (GZIP_MAGIC, BROTLI_MAGIC):
        return len(blob) > 50_000
    return TITLE_RE.search(blob) is not None


def main():
    if len(sys.argv) != 3:
        sys.exit("usage: extract_credits.py <resources.pak> <CREDITS.html>")
    with open(sys.argv[1], "rb") as f:
        data = f.read()

    found = []
    for rid, blob in entries(data):
        if not might_be_credits(blob):
            continue
        text = decode(blob)
        if TITLE_RE.search(text) and PRODUCT_RE.search(text):
            found.append((rid, text))

    if len(found) != 1:
        sys.exit("expected exactly one credits page in %s, found %d (ids %s)"
                 % (sys.argv[1], len(found), [rid for rid, _ in found]))

    rid, text = found[0]
    with open(sys.argv[2], "wb") as f:
        f.write(text)
    print("resource %d: %d bytes, %d credited projects -> %s"
          % (rid, len(text), len(PRODUCT_RE.findall(text)), sys.argv[2]))


if __name__ == "__main__":
    main()
