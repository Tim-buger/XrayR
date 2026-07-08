package limiter

import (
	"testing"

	"github.com/XrayR-project/XrayR/api"
)

func TestRecordOnlineIP(t *testing.T) {
	l := New()
	tag := "Vless_0.0.0.0_443"
	userTag := tag + "|user@example.com|1001"
	users := []api.UserInfo{{
		UID:   1001,
		Email: "user@example.com",
	}}

	if err := l.AddInboundLimiter(tag, 0, &users, nil); err != nil {
		t.Fatal(err)
	}
	if err := l.RecordOnlineIP(tag, userTag, "203.0.113.10"); err != nil {
		t.Fatal(err)
	}

	onlineUsers, err := l.GetOnlineDevice(tag)
	if err != nil {
		t.Fatal(err)
	}
	if len(*onlineUsers) != 1 {
		t.Fatalf("unexpected online user count: got %d, want 1", len(*onlineUsers))
	}
	if (*onlineUsers)[0].UID != 1001 {
		t.Fatalf("unexpected uid: got %d, want 1001", (*onlineUsers)[0].UID)
	}
	if (*onlineUsers)[0].IP != "203.0.113.10" {
		t.Fatalf("unexpected ip: got %s, want 203.0.113.10", (*onlineUsers)[0].IP)
	}

	onlineUsers, err = l.GetOnlineDevice(tag)
	if err != nil {
		t.Fatal(err)
	}
	if len(*onlineUsers) != 0 {
		t.Fatalf("online users should be reset after read, got %d", len(*onlineUsers))
	}
}
