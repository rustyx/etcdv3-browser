package nodetree

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddDelete(t *testing.T) {
	n := NewNode("")
	n.AddNode("a")
	n.AddNode("a/b")
	n.AddNode("a/c")
	n.AddNode("a/d/e/")
	n.AddNode("a/d/f")
	require.Equal(t, len(n.next), 1, "wrong root size")
	require.Equal(t, len(n.next["a"].next), 3, "wrong a/ size")
	require.Nil(t, n.next["a"].next["b"].next, "wrong a/b/ size")
	require.Equal(t, len(n.next["a"].next["d"].next), 2, "wrong a/d/ size")
	require.False(t, n.next["a"].next["d"].HasValue, "wrong a/d hasValue")
	require.True(t, n.next["a"].next["d"].next["e"].HasValue, "wrong a/d/e hasValue")
	require.Equal(t, n.GetNode("a/b").Key, "b", "a/b expected to exist")
	require.Equal(t, n.GetNode("a/c/").Key, "c", "a/c/ expected to exist")
	require.Equal(t, n.GetNode("a/d").Key, "d", "a/d expected to exist")
	require.Equal(t, n.GetNode("a/d/e").Key, "e", "a/d/e expected to exist")
	require.Equal(t, n.GetNode("a/d/f").Key, "f", "a/d/f expected to exist")
	require.Nil(t, n.GetNode("a/e"), "a/e expected to not exist")
	n.DeleteNode("a/d/e")
	require.Nil(t, n.GetNode("a/d/e"), "a/d/e expected to be gone")
	require.Equal(t, n.GetNode("a/d/f").Key, "f", "a/d/f expected to exist")
	n.DeleteNode("a/d/f")
	require.Nil(t, n.GetNode("a/d/f"), "a/d/f expected to be gone")
	require.Nil(t, n.GetNode("a/d"), "a/d expected to be gone")
	require.Equal(t, n.GetNode("a/b").Key, "b", "a/b expected to exist")
}
