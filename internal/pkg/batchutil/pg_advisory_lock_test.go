package batchutil

import "testing"

func TestLockID_Stable(t *testing.T) {
	id := LockID("fetcher")
	if id != LockID("fetcher") {
		t.Fatalf("LockID must be stable for identical input")
	}
	if id == LockID("updater") {
		t.Fatalf("LockID should differ for different input")
	}
}
