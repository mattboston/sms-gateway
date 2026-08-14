package smsutil

import (
	"time"
	"unicode/utf16"
)

// DefaultConcatWindow is how close consecutive full segments must be to merge.
const DefaultConcatWindow = 20 * time.Second

// ShortPartConcatWindow is the max gap allowed when appending a short final part.
const ShortPartConcatWindow = 5 * time.Second

// UTF16Len returns the UTF-16 code-unit length of s (SMS UCS-2 semantics).
func UTF16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

// IsLikelyFullSMSSegment reports whether n UTF-16 code units looks like a full
// concatenated SMS segment. UCS-2 concat parts are typically 67 units; GSM-7
// concat parts are typically 153. Single-segment maxima (70 / 160) are excluded
// so complete one-off messages (e.g. 68-char OTPs) are not treated as fragments.
func IsLikelyFullSMSSegment(n int) bool {
	return (n >= 60 && n <= 67) || (n >= 145 && n <= 153)
}

// ShouldMergePart reports whether newBody should be appended to an existing
// inbound message whose last received segment had lastPartUnits UTF-16 units
// and was last touched at lastTouch.
func ShouldMergePart(phoneMatch bool, lastPartUnits int, lastTouch time.Time, existingBody, newBody string, now time.Time, window time.Duration) bool {
	if !phoneMatch || newBody == "" {
		return false
	}
	if existingBody == newBody {
		return false
	}
	if !IsLikelyFullSMSSegment(lastPartUnits) {
		return false
	}

	maxWindow := window
	if maxWindow <= 0 {
		maxWindow = DefaultConcatWindow
	}
	// Short trailing parts must arrive quickly; otherwise they are likely a
	// separate message (e.g. a later "fd" reply) rather than a concat fragment.
	if !IsLikelyFullSMSSegment(UTF16Len(newBody)) && maxWindow > ShortPartConcatWindow {
		maxWindow = ShortPartConcatWindow
	}

	if now.Sub(lastTouch) > maxWindow {
		return false
	}
	return true
}
