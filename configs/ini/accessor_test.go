package ini

import (
	"errors"
	"testing"
)

func TestTypedAccessors(t *testing.T) {
	cfg, err := LoadFromString(`
[server]
port = 8080
ratio = 0.75
debug = true
peers = a, b , c
empty =
`)
	if err != nil {
		t.Fatal(err)
	}

	if port, err := cfg.GetInt("server", "port"); err != nil || port != 8080 {
		t.Fatalf("port = %v %v", port, err)
	}
	if ratio, err := cfg.GetFloat64("server", "ratio"); err != nil || ratio != 0.75 {
		t.Fatalf("ratio = %v %v", ratio, err)
	}
	if debug, err := cfg.GetBool("server", "debug"); err != nil || !debug {
		t.Fatalf("debug = %v %v", debug, err)
	}

	peers, err := cfg.GetStrings("server", "peers")
	if err != nil || len(peers) != 3 || peers[1] != "b" {
		t.Fatalf("peers = %v %v, want [a b c]", peers, err)
	}

	empty, err := cfg.GetStrings("server", "empty")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty = %v %v, want empty slice", empty, err)
	}

	if _, err := cfg.GetInt("server", "missing"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
	if _, err := cfg.GetInt("server", "debug"); !errors.Is(err, ErrConvertFailed) {
		t.Fatalf("expected ErrConvertFailed, got %v", err)
	}
	if cfg.HasSection("nope") || cfg.HasKey("server", "nope") {
		t.Fatal("unexpected section or key")
	}
	if shown := cfg.GetSectionNames(); len(shown) != 1 || shown[0] != "server" {
		t.Fatalf("GetSectionNames = %v", shown)
	}
	if shown := cfg.GetKeys("server"); len(shown) != 5 || shown[0] != "port" {
		t.Fatalf("GetKeys = %v", shown)
	}
}
