package mesh

import (
	"testing"
)

func TestNewNode_Basic(t *testing.T) {
	node, err := NewNode("node-1", ":0", 10, nil)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	defer node.Stop()

	if node == nil {
		t.Fatal("expected non-nil Node")
	}
	if node.id != "node-1" {
		t.Errorf("node id = %s, want node-1", node.id)
	}
}

func TestNewNode_WithSeedAddresses(t *testing.T) {
	seeds := []string{":9001", ":9002"}
	node, err := NewNode("node-2", ":0", 10, seeds)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	defer node.Stop()

	if node.config.seedAddresses == nil {
		t.Error("seedAddresses should not be nil")
	}
	if len(node.config.seedAddresses) != 2 {
		t.Errorf("seedAddresses len = %d, want 2", len(node.config.seedAddresses))
	}
}

func TestNewNode_WithOptions(t *testing.T) {
	node, err := NewNode("node-3", ":0", 10, nil,
		WithDebug(),
		WithMessageTTL(5),
		WithGossipFanoutSize(4),
	)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	defer node.Stop()

	if !node.config.Debug {
		t.Error("Debug should be true")
	}
	if node.config.MessageTTL != 5 {
		t.Errorf("MessageTTL = %d, want 5", node.config.MessageTTL)
	}
	if node.config.GossipFanoutSize != 4 {
		t.Errorf("GossipFanoutSize = %d, want 4", node.config.GossipFanoutSize)
	}
}

func TestNode_GetNodeStates(t *testing.T) {
	node, err := NewNode("node-4", ":0", 10, nil)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	defer node.Stop()

	states := node.GetNodeStates()
	// 节点注册了自身，应至少有一个
	if len(states) < 1 {
		t.Error("GetNodeStates should return at least the node itself")
	}

	found := false
	for _, s := range states {
		if s.ID == "node-4" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetNodeStates should include the node itself")
	}
}

func TestNode_Subscribe(t *testing.T) {
	node, err := NewNode("node-sub", ":0", 10, nil)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	defer node.Stop()

	err = node.Subscribe("/test/topic", func(msg Message, pathValues map[string]string) {
		// callback
	})
	if err != nil {
		t.Errorf("Subscribe failed: %v", err)
	}
}

func TestNode_Publish(t *testing.T) {
	node, err := NewNode("node-pub", ":0", 10, nil)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	defer node.Stop()

	err = node.Publish("/test/topic", "hello")
	if err != nil {
		t.Errorf("Publish failed: %v", err)
	}
}

func TestNode_Put(t *testing.T) {
	node, err := NewNode("node-put", ":0", 10, nil)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	defer node.Stop()

	err = node.Put("key1", "value1")
	if err != nil {
		t.Errorf("Put failed: %v", err)
	}
}

func TestNode_RegisterQueryHandler(t *testing.T) {
	node, err := NewNode("node-qh", ":0", 10, nil)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	defer node.Stop()

	err = node.RegisterQueryHandler("/rpc/echo", func(msg Message) any {
		return msg.Payload
	})
	if err != nil {
		t.Errorf("RegisterQueryHandler failed: %v", err)
	}
}
