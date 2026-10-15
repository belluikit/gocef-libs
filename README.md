# gocef-libs

The CEF payload for [gocef](https://github.com/belluikit/gocef) and bellui's
Linux web-view backend (belluikit/bellui#104): a pinned Chromium Embedded
Framework build, verified against upstream, stripped, and published as a release
asset.

## Why the payload is not in the module

[`goffmpeg-libs`](https://github.com/belluikit/goffmpeg-libs) commits FFmpeg's
static libraries straight into a Go module per platform, and that works because
they are ~25 MB each. It does not generalise: a **stripped** `libcef.so` is
**256 MiB**, which is 2.5x GitHub's hard 100 MB per-file limit, and the full
per-architecture payload is ~350 MB. Git LFS is not a way out either, because
the Go module proxy does not resolve LFS pointers.

So the shape is inverted. The binaries travel as **release assets** — no git
involvement, 2 GB per asset allowed — and the Go module carries only the pin:
which CEF build, which upstream checksum it was made from, and what the
published asset must hash to. Small, auditable, and versioned like any other
dependency.

## Why one module rather than one per platform

`goffmpeg-libs`' split into `linux_amd64/`, `darwin_arm64/` and so on exists for
exactly one reason: so a consumer downloads only its own target's binaries
instead of all ~245 MB. With the payload in release assets that reason
disappears — nothing is downloaded by `go get` at all.

And the split would actively hurt here, because the main consumer is
`bellui package`, which cross-packages: packaging a linux/arm64 application from
a macOS workstation needs the arm64 pin, so a per-platform module would have to
be imported for every target anyway. One `manifest` package covering all
platforms is simpler and matches how it is used.

## Layout of a payload

Unpacked, everything sits **flat**, beside `libcef.so`:

```
libcef.so                 stripped, ~256 MiB
cef_helper                the subprocess helper, prebuilt for the target
icudtl.dat                ICU data
*.pak                     resources, incl. chrome_100_percent / chrome_200_percent
snapshot_blob.bin         V8 snapshots — note these come from the distribution's
v8_context_snapshot.bin   Release/, not its Resources/
locales/                  every locale (~48 MB)
libvk_swiftshader.so      software Vulkan, for a machine with no usable GPU
libvulkan.so.1, libEGL.so, libGLESv2.so
chrome-sandbox            see the sandbox note below
LICENSE.cef.txt           CEF's licence
CREDITS.html              Chromium's third-party notices (about:credits)
README.cef.txt            CEF's distribution README, when upstream has one
```

Flat is not a preference. CEF's own distribution nests resources under
`Resources/` and its CMake copies them next to the binary, because that is where
Chromium looks — point `resources_dir_path` at a nested directory and it fails
inside ICU with `Invalid file descriptor to ICU data received`, then core-dumps.

**Locales are shipped whole.** Trimming belongs to whoever packages an
application, because only they know which locales it offers.

**The sandbox is not decided here.** `chrome-sandbox` needs to be owned by root
with the SUID bit, which a Flatpak cannot provide — so the choice between
CEF's sandbox and relying on the surrounding bubblewrap sandbox is
`bellui package`'s to make (belluikit/bellui#107).

## Updating to a new CEF — the security path

This repository exists so that a Chromium security fix is a payload swap rather
than a porting exercise. The procedure is one dispatch:

1. Run the **Mirror** workflow with the new CEF version, `publish: true`. It
   fetches the upstream build, **verifies its SHA-1 against CEF's own build
   index** (not against a value pasted into this repo — the point is to prove we
   mirrored what upstream published), strips, lays out, packs and publishes a
   release for both architectures.
2. Copy the printed SHA-256 values into `manifest/manifest.go` along with the new
   version, and commit.
3. Consumers pick it up with `go get -u`.

Note that `gocef` is pinned to `CEF_API_VERSION` 13300 — the floor of the range
these headers support — precisely so that step 1 does not usually require
re-vendoring headers or rebuilding the binding.

Cadence and ownership of that procedure are belluikit/bellui#98. It is not
optional: mirroring a browser engine means owning its update treadmill.

## Provenance and licence

Built from CEF's official binary distributions at `cef-builds.spotifycdn.com`,
unmodified apart from stripping debug symbols, rearranging the layout and adding
the extracted `CREDITS.html`.

This repository's own code is BSD-3-Clause, see [LICENSE](LICENSE). The payload
is not ours and carries its own licences:

- **CEF** is BSD-3-Clause. Its licence ships in every payload as
  `LICENSE.cef.txt`.
- **Chromium** bundles several hundred third-party projects under many licences,
  some of them LGPL (as FFmpeg is in [`goffmpeg-libs`](https://github.com/belluikit/goffmpeg-libs)).
  Their notices are Chromium's about:credits page, which the payload otherwise
  carries only compressed inside `resources.pak`. The mirror workflow extracts it with
  [`scripts/extract_credits.py`](scripts/extract_credits.py) and ships it as
  `CREDITS.html`. Payloads published before that change do not contain it.

Anyone redistributing a payload, or an application that bundles one, is
responsible for meeting those licences; this repository does not make that
determination for you.
