package batchutil

import "testing"

func TestLockID_Stable(t *testing.T) {
	if LockID("fetcher") != LockID("fetcher") {
		t.Fatalf("LockID must be stable for identical input")
	}
	if LockID("fetcher") == LockID("updater") {
		t.Fatalf("LockID should differ for different input")
	}
}

