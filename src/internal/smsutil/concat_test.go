package smsutil

import (
	"strings"
	"testing"
	"time"
)

func TestIsLikelyFullSMSSegment(t *testing.T) {
	cases := []struct {
		n    int
		want bool
	}{
		{67, true},
		{60, true},
		{59, false},
		{68, false}, // single-segment / OTP-sized — do not treat as concat fragment
		{70, false},
		{153, true},
		{154, false},
		{160, false},
		{144, false},
		{2, false},
	}
	for _, tc := range cases {
		if got := IsLikelyFullSMSSegment(tc.n); got != tc.want {
			t.Errorf("IsLikelyFullSMSSegment(%d)=%v want %v", tc.n, got, tc.want)
		}
	}
}

func TestUTF16LenEmoji(t *testing.T) {
	// Supplementary-plane emoji counts as two UTF-16 code units.
	if UTF16Len("🏴") != 2 {
		t.Fatalf("UTF16Len(flag)=%d want 2", UTF16Len("🏴"))
	}
}

func TestShouldMergePart(t *testing.T) {
	now := time.Date(2026, 8, 3, 8, 47, 5, 0, time.UTC)
	prev := now.Add(-2 * time.Second)
	long := strings.Repeat("ا", 67)
	next := "ادامه متن"
	otp := strings.Repeat("ب", 68)

	if !ShouldMergePart(true, 67, prev, long, next, now, 0) {
		t.Fatal("expected merge for full UCS-2 segment within window")
	}
	if ShouldMergePart(true, 68, prev, otp, next, now, 0) {
		t.Fatal("should not merge after OTP-sized 68-unit segment")
	}
	if ShouldMergePart(true, 67, prev, long, long, now, 0) {
		t.Fatal("should not merge identical bodies")
	}
	if ShouldMergePart(true, 67, now.Add(-30*time.Second), long, next, now, 0) {
		t.Fatal("should not merge outside window")
	}
	if ShouldMergePart(true, 2, prev, "fd", next, now, 0) {
		t.Fatal("should not merge after short final segment")
	}
	if ShouldMergePart(false, 67, prev, long, next, now, 0) {
		t.Fatal("should not merge different senders")
	}
	if ShouldMergePart(true, 67, now.Add(-10*time.Second), long, "fd", now, 0) {
		t.Fatal("should not merge short part outside short window")
	}
	if !ShouldMergePart(true, 67, now.Add(-2*time.Second), long, "fd", now, 0) {
		t.Fatal("expected short final part within 5s to merge")
	}
}
