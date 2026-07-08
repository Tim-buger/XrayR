package controller

import "testing"

type testCounter int64

func (c *testCounter) Value() int64 {
	return int64(*c)
}

func (c *testCounter) Set(v int64) int64 {
	old := int64(*c)
	*c = testCounter(v)
	return old
}

func (c *testCounter) Add(v int64) int64 {
	*c = testCounter(int64(*c) + v)
	return int64(*c)
}

func TestResetTrafficKeepsTrafficAddedAfterSample(t *testing.T) {
	var up testCounter = 125
	var down testCounter = 260
	upCounters := []trafficCounterSample{{counter: &up, value: 100}}
	downCounters := []trafficCounterSample{{counter: &down, value: 200}}

	(*Controller)(nil).resetTraffic(&upCounters, &downCounters)

	if up.Value() != 25 {
		t.Fatalf("unexpected upload counter value: got %d, want 25", up.Value())
	}
	if down.Value() != 60 {
		t.Fatalf("unexpected download counter value: got %d, want 60", down.Value())
	}
}

func TestResetTrafficDoesNotLeaveNegativeCounter(t *testing.T) {
	var up testCounter = 25
	var down testCounter = 40
	upCounters := []trafficCounterSample{{counter: &up, value: 100}}
	downCounters := []trafficCounterSample{{counter: &down, value: 200}}

	(*Controller)(nil).resetTraffic(&upCounters, &downCounters)

	if up.Value() != 0 {
		t.Fatalf("unexpected upload counter value: got %d, want 0", up.Value())
	}
	if down.Value() != 0 {
		t.Fatalf("unexpected download counter value: got %d, want 0", down.Value())
	}
}
