package prepare_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fiffeek/hyprdynamicmonitors/internal/config"
	"github.com/fiffeek/hyprdynamicmonitors/internal/prepare"
	"github.com/fiffeek/hyprdynamicmonitors/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_TruncateDestinationLua(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "monitors.lua")
	require.NoError(t, os.WriteFile(destination, []byte(`hl.monitor({
    output = "desc:Keep",
    mode = "preferred",
})
hl.monitor({
    output = "desc:Drop",
    disabled = true,
})
return true
`), 0o600))

	cfg := testutils.NewTestConfig(t).
		WithDestination(destination).
		WithConfigFormat(config.LuaConfigFormat).
		Get()

	service := prepare.NewService(cfg)
	require.NoError(t, service.TruncateDestination())

	contents, err := os.ReadFile(destination)
	require.NoError(t, err)
	assert.Contains(t, string(contents), `output = "desc:Keep"`)
	assert.NotContains(t, string(contents), `output = "desc:Drop"`)
	assert.Contains(t, string(contents), `return true`)
}
