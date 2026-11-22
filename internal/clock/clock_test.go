package clock

import (
	"sync"
	"testing"
	"time"
)

func TestRealReturnsUTC(t *testing.T) {
	t.Parallel()
	before := time.Now().Add(-time.Second)
	now := Real{}.Now()
	if now.Location() != time.UTC {
		t.Fatalf("location = %v, want UTC", now.Location())
	}
	if now.Before(before) {
		t.Fatalf("real clock returned %v, earlier than %v", now, before)
	}
}

func TestFakeSetAndAdvance(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 5, 11, 0, 0, 0, time.FixedZone("CET", 3600))
	f := NewFake(start)
	if got := f.Now(); !got.Equal(start) || got.Location() != time.UTC {
		t.Fatalf("Now() = %v, want %v in UTC", got, start)
	}

	if got := f.Advance(90 * time.Minute); !got.Equal(start.Add(90 * time.Minute)) {
		t.Fatalf("Advance returned %v", got)
	}
	if got := f.Now(); !got.Equal(start.Add(90 * time.Minute)) {
		t.Fatalf("Now() after Advance = %v", got)
	}

	target := time.Date(2030, 6, 1, 0, 0, 0, 0, time.FixedZone("X", -7200))
	f.Set(target)
	if got := f.Now(); !got.Equal(target) || got.Location() != time.UTC {
		t.Fatalf("Now() after Set = %v, want %v in UTC", got, target)
	}

	f.Advance(-time.Hour)
	if got := f.Now(); !got.Equal(target.Add(-time.Hour)) {
		t.Fatalf("negative Advance = %v", got)
	}
}

func TestFakeConcurrentAccess(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	f := NewFake(start)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			f.Advance(time.Second)
		}()
		go func() {
			defer wg.Done()
			_ = f.Now()
		}()
	}
	wg.Wait()
	if got, want := f.Now(), start.Add(50*time.Second); !got.Equal(want) {
		t.Fatalf("Now() = %v, want %v", got, want)
	}
}

func TestInterfaceSatisfied(t *testing.T) {
	t.Parallel()
	var _ Clock = Real{}
	var _ Clock = NewFake(time.Time{})
}
