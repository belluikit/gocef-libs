// Package manifest is the pin: which CEF build this repository mirrors, and how
// to verify what it publishes.
//
// It deliberately contains no binaries. A stripped libcef.so is 256 MiB — 2.5x
// GitHub's hard 100 MB per-file limit — so the payload cannot live in a
// git-backed Go module the way goffmpeg-libs commits FFmpeg's ~25 MB static
// libraries. It travels as a release asset instead, and this package is the
// small, auditable, versioned part that says which asset and what it should
// hash to.
//
// Consumers are `bellui package`, which fetches the asset for the target it is
// packaging, and anything else that needs to know which CEF an application is
// carrying (belluikit/bellui#98).
package manifest

import "fmt"

// The CEF build this repository mirrors. Bumping these is the whole content of
// a Chromium security update: change them, run the mirror workflow, publish.
const (
	CEFVersion      = "154.0.34+g14c5a08+chromium-154.0.8037.98"
	ChromiumVersion = "154.0.8037.98"

	// UpstreamHost is where the mirror workflow fetches from. It is recorded
	// rather than hard-coded in the workflow so provenance and process live in
	// the same place.
	UpstreamHost = "cef-builds.spotifycdn.com"
)

// Platform names a target. These match the directory names CEF's own build
// index uses, so a reader can find the upstream artifact from the name alone.
type Platform string

const (
	Linux64    Platform = "linux64"
	LinuxARM64 Platform = "linuxarm64"
)

// Asset describes one published payload.
type Asset struct {
	// Platform this payload is for.
	Platform Platform

	// Name is the release asset's filename.
	Name string

	// SHA256 is the hash of the published asset, written by the mirror workflow.
	// Empty until the first release is published for this pin.
	SHA256 string

	// UpstreamSHA1 is the checksum CEF's build index gives for the archive this
	// was built from — the link back to upstream, checked before anything is
	// unpacked.
	UpstreamSHA1 string
}

// Assets is every payload this pin publishes.
//
// The SHA256 values are filled in by the mirror workflow when it publishes, and
// committed back. A consumer that finds one empty should refuse to install the
// asset rather than trust it: an unverified Chromium is the one thing this
// repository exists to prevent.
//
// They are also independently checkable. The tarball is built reproducibly, so
// re-running the mirror on this CEF version derives the same hashes — verified
// for both architectures across separate CI runs, not merely intended.
var Assets = map[Platform]Asset{
	Linux64: {
		Platform:     Linux64,
		Name:         "cef-" + CEFVersion + "-linux64.tar.zst",
		SHA256:       "cf12f8b59045c473702b1b21d6127ea02470a990f937ab92a393818aeadd539d",
		UpstreamSHA1: "15c91ad767097217b2e82ba8b933fe30b134be69",
	},
	LinuxARM64: {
		Platform:     LinuxARM64,
		Name:         "cef-" + CEFVersion + "-linuxarm64.tar.zst",
		SHA256:       "2d8dfb08448b92009871f6a14ca2a33820565dd04f7b18a1c174d015d3907185",
		UpstreamSHA1: "fa85793519eef199e6b6fa8cc8d6f0eb45c6dac2",
	},
}

// For returns the asset for a platform.
func For(p Platform) (Asset, error) {
	a, ok := Assets[p]
	if !ok {
		return Asset{}, fmt.Errorf("gocef-libs: no payload for platform %q", p)
	}
	return a, nil
}

// PlatformFor maps a Go GOOS/GOARCH pair to a payload platform.
func PlatformFor(goos, goarch string) (Platform, error) {
	if goos != "linux" {
		return "", fmt.Errorf("gocef-libs: no CEF payload for %s (only linux)", goos)
	}
	switch goarch {
	case "amd64":
		return Linux64, nil
	case "arm64":
		return LinuxARM64, nil
	}
	return "", fmt.Errorf("gocef-libs: no CEF payload for linux/%s", goarch)
}

// ReleaseTag is the git tag the mirror workflow publishes each pin under. One
// release per CEF build, so an application can be traced to an exact Chromium.
func ReleaseTag() string {
	return "cef-" + CEFVersion
}

// DownloadURL is where an asset can be fetched from.
func (a Asset) DownloadURL() string {
	return "https://github.com/belluikit/gocef-libs/releases/download/" + ReleaseTag() + "/" + a.Name
}

// UpstreamURL is the CEF build this asset was made from. The '+' characters in
// a CEF version have to be percent-encoded to survive a URL.
func (a Asset) UpstreamURL() string {
	encoded := ""
	for _, r := range CEFVersion {
		if r == '+' {
			encoded += "%2B"
			continue
		}
		encoded += string(r)
	}
	return "https://" + UpstreamHost + "/cef_binary_" + encoded + "_" + string(a.Platform) + "_minimal.tar.bz2"
}
