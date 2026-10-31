package serve

import "testing"

func TestListenAddrs(t *testing.T) {
	got := listenAddrs(serveOptions{listen: "0.0.0.0", httpPort: 4318, grpcPort: 4317})

	want := []string{"0.0.0.0:4318", "0.0.0.0:4317"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}
