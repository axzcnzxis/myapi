package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The panel playground serves the /v1 relay endpoints under the /pg prefix
// (POST /pg/chat/completions). Advanced custom channels declare their routes in
// /v1 terms, so route matching has to resolve the alias; otherwise such a channel
// is rejected for every playground request even though it declares the
// equivalent /v1 route.
func TestAdvancedCustomMatchPathForModelResolvesPlaygroundAlias(t *testing.T) {
	config := &AdvancedCustomConfig{
		Routes: []AdvancedCustomRoute{{
			IncomingPath: "/v1/chat/completions",
			UpstreamPath: "/v1/chat/completions",
			Converter:    advancedCustomConverterNone,
		}},
	}
	require.NoError(t, config.Validate())

	route, ok := config.MatchPathForModel("/pg/chat/completions", "hy3")
	require.True(t, ok, "the playground alias must resolve to the channel's /v1 route")
	assert.Equal(t, "/v1/chat/completions", route.IncomingPath)

	assert.True(t, config.SupportsPathForModel("/pg/chat/completions", "hy3"))
}

// A channel that declares the alias explicitly keeps matching it, the alias never
// leaks into unrelated /v1 paths, and plain /v1 requests are unaffected.
func TestAdvancedCustomMatchPathForModelKeepsDeclaredAliasRoute(t *testing.T) {
	config := &AdvancedCustomConfig{
		Routes: []AdvancedCustomRoute{{
			IncomingPath: "/pg/chat/completions",
			UpstreamPath: "/v1/chat/completions",
			Converter:    advancedCustomConverterNone,
		}},
	}
	require.NoError(t, config.Validate())

	_, ok := config.MatchPathForModel("/pg/chat/completions", "hy3")
	assert.True(t, ok, "an explicitly declared alias route keeps matching")

	_, ok = config.MatchPathForModel("/v1/chat/completions", "hy3")
	assert.False(t, ok, "a declared alias route must not answer plain /v1 requests")

	_, ok = config.MatchPathForModel("/v1/messages", "hy3")
	assert.False(t, ok)
}
