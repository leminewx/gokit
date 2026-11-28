package stream

import (
	"testing"
)

func TestItem(t *testing.T) {
	item := NewItem(1)
	t.Log(item.GetValue())
	t.Log(item.GetGroup())

	item.SetValue(3)
	t.Log(item.GetValue())
	t.Log(item.GetGroup())

	item.SetGroup("a")
	t.Log(item.GetValue())
	t.Log(item.GetGroup())
}
