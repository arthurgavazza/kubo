package cli

import (
	"testing"

	"github.com/ipfs/kubo/test/cli/harness"
	"github.com/ipfs/kubo/test/cli/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoutingDHT(t *testing.T) {
	t.Parallel()
	nodes := harness.NewT(t).NewNodes(5).Init()
	nodes.ForEachPar(func(node *harness.Node) {
		node.IPFS("config", "Routing.Type", "dht")
	})
	nodes.StartDaemons().Connect()

	t.Run("ipfs routing findpeer", func(t *testing.T) {
		t.Parallel()
		res := nodes[1].RunIPFS("routing", "findpeer", nodes[0].PeerID().String())
		assert.Equal(t, 0, res.ExitCode())

		swarmAddr := nodes[0].SwarmAddrsWithoutPeerIDs()[0]
		require.Equal(t, swarmAddr.String(), res.Stdout.Trimmed())
	})

	t.Run("ipfs routing get <key>", func(t *testing.T) {
		t.Parallel()
		hash := nodes[2].IPFSAddStr("hello world")
		nodes[2].IPFS("name", "publish", "/ipfs/"+hash)

		res := nodes[1].IPFS("routing", "get", "/ipns/"+nodes[2].PeerID().String())
		assert.Contains(t, res.Stdout.String(), "/ipfs/"+hash)

		t.Run("put round trips (#3124)", func(t *testing.T) {
			t.Parallel()
			nodes[0].WriteBytes("get_result", res.Stdout.Bytes())
			res := nodes[0].IPFS("routing", "put", "/ipns/"+nodes[2].PeerID().String(), "get_result")
			assert.Greater(t, len(res.Stdout.Lines()), 0, "should put to at least one node")
		})

		t.Run("put with bad keys fails (issue #5113, #4611)", func(t *testing.T) {
			t.Parallel()
			keys := []string{"foo", "/pk/foo", "/ipns/foo"}
			for _, key := range keys {
				key := key
				t.Run(key, func(t *testing.T) {
					t.Parallel()
					res := nodes[0].RunIPFS("routing", "put", key)
					assert.Equal(t, 1, res.ExitCode())
					assert.Contains(t, res.Stderr.String(), "invalid")
					assert.Empty(t, res.Stdout.String())
				})
			}
		})

		t.Run("get with bad keys (issue #4611)", func(t *testing.T) {
			for _, key := range []string{"foo", "/pk/foo"} {
				key := key
				t.Run(key, func(t *testing.T) {
					t.Parallel()
					res := nodes[0].RunIPFS("routing", "get", key)
					assert.Equal(t, 1, res.ExitCode())
					assert.Contains(t, res.Stderr.String(), "invalid")
					assert.Empty(t, res.Stdout.String())
				})
			}
		})
	})

	t.Run("ipfs routing findprovs", func(t *testing.T) {
		t.Parallel()
		hash := nodes[3].IPFSAddStr("some stuff")
		res := nodes[4].IPFS("routing", "findprovs", hash)
		assert.Equal(t, nodes[3].PeerID().String(), res.Stdout.Trimmed())
	})

	t.Run("routing commands fail when offline", func(t *testing.T) {
		t.Parallel()
		node := harness.NewT(t).NewNode().Init()

		// these cannot be run in parallel due to repo locking (seems like a bug)

		t.Run("routing findprovs", func(t *testing.T) {
			res := node.RunIPFS("routing", "findprovs", testutils.CIDEmptyDir)
			assert.Equal(t, 1, res.ExitCode())
			assert.Contains(t, res.Stderr.String(), "this command must be run in online mode")
		})

		t.Run("routing findpeer", func(t *testing.T) {
			res := node.RunIPFS("routing", "findpeer", testutils.CIDEmptyDir)
			assert.Equal(t, 1, res.ExitCode())
			assert.Contains(t, res.Stderr.String(), "this command must be run in online mode")
		})

		t.Run("routing put", func(t *testing.T) {
			node.WriteBytes("foo", []byte("foo"))
			res := node.RunIPFS("routing", "put", "/ipns/"+node.PeerID().String(), "foo")
			assert.Equal(t, 1, res.ExitCode())
			assert.Contains(t, res.Stderr.String(), "this action must be run in online mode")
		})
	})
}
