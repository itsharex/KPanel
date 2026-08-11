package browsercore

import "testing"

func TestLimiterBoundsAndReleasesPerSessionWork(t *testing.T) {
	limiter := NewLimiter(3, 2)
	releaseOne, ok := limiter.Acquire("one")
	if !ok {
		t.Fatal("first acquire rejected")
	}
	releaseTwo, ok := limiter.Acquire("one")
	if !ok {
		t.Fatal("second acquire rejected")
	}
	if _, ok := limiter.Acquire("one"); ok {
		t.Fatal("per-session limit was not enforced")
	}
	releaseOther, ok := limiter.Acquire("two")
	if !ok {
		t.Fatal("global spare slot rejected")
	}
	if _, ok := limiter.Acquire("three"); ok {
		t.Fatal("global limit was not enforced")
	}
	releaseOne()
	releaseOne()
	if _, ok := limiter.Acquire("one"); !ok {
		t.Fatal("released slot was not reusable")
	}
	releaseTwo()
	releaseOther()
}
