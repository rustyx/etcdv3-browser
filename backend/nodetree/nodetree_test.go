package nodetree

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitPath(t *testing.T) {
	for i, data := range []struct {
		In  string
		Out []string
	}{
		{"", []string{}},
		{"a", []string{"a"}},
		{"/", []string{"/"}},
		{"a/b", []string{"a/", "b"}},
		{"a/b/", []string{"a/", "b/"}},
		{"a//b/c", []string{"a/", "/", "b/", "c"}},
		{"/a", []string{"/a"}},
		{"/a/b", []string{"/a/", "b"}},
		{"//a//b//", []string{"//a/", "/", "b/", "/"}},
	} {
		got := splitPath(&data.In)
		require.Equal(t, data.Out, got, fmt.Sprintf("case %v \"%v\"", i, data.In))
	}
}

func TestAddDelete(t *testing.T) {
	n := NewNode("")
	n.AddNode("a")
	n.AddNode("a/b")
	n.AddNode("a/c")
	n.AddNode("a/d/e/")
	n.AddNode("a/d/f")
	n.AddNode("/a/e")
	require.Equal(t, 3, len(n.next), "wrong root size")
	require.Equal(t, 3, len(n.next["a/"].next), "wrong a/ size")
	require.Nil(t, n.next["a/"].next["b"].next, "wrong a/b/ size")
	require.Equal(t, 1, len(n.next["/a/"].next), "wrong /a/ size")
	require.Equal(t, 2, len(n.next["a/"].next["d/"].next), "wrong a/d/ size")
	require.False(t, n.next["a/"].next["d/"].HasValue, "wrong a/d hasValue")
	require.True(t, n.next["a/"].next["d/"].next["e/"].HasValue, "wrong a/d/e/ hasValue")
	require.Equal(t, "b", n.GetNode("a/b").Key, "a/b expected to exist")
	require.Equal(t, "c", n.GetNode("a/c").Key, "a/c expected to exist")
	require.Equal(t, "d/", n.GetNode("a/d/").Key, "a/d/ expected to exist")
	require.Equal(t, "e/", n.GetNode("a/d/e/").Key, "a/d/e/ expected to exist")
	require.Equal(t, "f", n.GetNode("a/d/f").Key, "a/d/f expected to exist")
	require.Equal(t, "e", n.GetNode("/a/e").Key, "/a/e expected to exist")
	require.Nil(t, n.GetNode("a/e"), "a/e expected to not exist")
	n.DeleteNode("a/d/e/")
	require.Nil(t, n.GetNode("a/d/e/"), "a/d/e/ expected to be gone")
	require.Equal(t, "f", n.GetNode("a/d/f").Key, "a/d/f expected to exist")
	n.DeleteNode("a/d/f")
	require.Nil(t, n.GetNode("a/d/f"), "a/d/f expected to be gone")
	require.Nil(t, n.GetNode("a/d/"), "a/d/ expected to be gone")
	require.Equal(t, "b", n.GetNode("a/b").Key, "a/b expected to exist")
}
