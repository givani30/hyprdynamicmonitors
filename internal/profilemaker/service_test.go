package profilemaker_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fiffeek/hyprdynamicmonitors/internal/config"
	"github.com/fiffeek/hyprdynamicmonitors/internal/hypr"
	"github.com/fiffeek/hyprdynamicmonitors/internal/profilemaker"
	"github.com/fiffeek/hyprdynamicmonitors/internal/testutils"
	"github.com/fiffeek/hyprdynamicmonitors/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var regenerate = flag.Bool("regenerate", false, "regenerate fixtures instead of comparing")

func TestService_EditExisting(t *testing.T) {
	// Sample monitors data
	monitors := []*hypr.MonitorSpec{
		{
			ID:          utils.IntPtr(1),
			Name:        "monA",
			Description: "New Monitor A",
			Width:       2560,
			Height:      1440,
			RefreshRate: 120.0,
			X:           0,
			Y:           0,
			Scale:       1.5,
			Transform:   0,
			Vrr:         false,
		},
		{
			ID:            utils.IntPtr(2),
			Name:          "monB",
			Description:   "New Monitor B",
			Width:         1920,
			Height:        1080,
			RefreshRate:   60.0,
			X:             2560,
			Y:             0,
			Scale:         1.0,
			Transform:     0,
			Mirror:        "eDP-1",
			Vrr:           true,
			CurrentFormat: "XRGB2101010",
			ColorPreset:   "hdr",
			SdrBrightness: 1.1,
			SdrSaturation: 0.98,
		},
		{
			ID:          utils.IntPtr(3),
			Name:        "monC",
			Description: "",
			Width:       1000,
			Height:      1000,
			RefreshRate: 60.0,
			X:           -1000,
			Y:           -1000,
			Scale:       1.0,
			Transform:   0,
		},
		{
			ID:          utils.IntPtr(4),
			Name:        "monD",
			Description: "Dell #Whatever",
			Width:       1337,
			Height:      500,
			RefreshRate: 60.0,
			X:           2137,
			Y:           -5000,
			Scale:       2.0,
			Transform:   2,
		},
	}

	for _, monitor := range monitors {
		require.NoError(t, monitor.Validate(), "monitor spec should be correct")
	}

	testCases := []struct {
		name          string
		inputFile     string
		expectedFile  string
		profileName   string
		expectError   bool
		errorContains string
	}{
		{
			name:         "Replace content between existing markers",
			inputFile:    "testdata/existing_config_with_markers.conf",
			expectedFile: "testdata/expected_replace_markers.conf",
			profileName:  "test-profile",
		},
		{
			name:         "Append content when no markers exist",
			inputFile:    "testdata/existing_config_no_markers.conf",
			expectedFile: "testdata/expected_append_no_markers.conf",
			profileName:  "test-profile",
		},
		{
			name:         "Append content when markers are broken",
			inputFile:    "testdata/config_with_broken_markers.conf",
			expectedFile: "testdata/expected_append_broken_markers.conf",
			profileName:  "test-profile",
		},
		{
			name:         "Handle empty config file",
			inputFile:    "testdata/empty_config.conf",
			expectedFile: "testdata/expected_empty_config.conf",
			profileName:  "test-profile",
		},
		{
			name:         "Handle non-existent file",
			inputFile:    "testdata/non_existent.conf",
			expectedFile: "testdata/expected_non_existent.conf",
			profileName:  "test-profile",
		},
		{
			name:          "Profile not found",
			inputFile:     "testdata/empty_config.conf",
			profileName:   "non-existent-profile",
			expectError:   true,
			errorContains: "profile not found",
		},
		{
			name:         "Repeated calls should not accumulate newlines",
			inputFile:    "testdata/existing_config_with_markers.conf",
			expectedFile: "testdata/expected_replace_markers.conf",
			profileName:  "test-profile",
		},
		{
			name:         "Repeated calls with markers only should not accumulate newlines",
			inputFile:    "testdata/markers_only.conf",
			expectedFile: "testdata/expected_markers_only.conf",
			profileName:  "test-profile",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "profilemaker_test_")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			configFile := filepath.Join(tempDir, "test_config.conf")

			var cfg *config.Config
			if tc.expectError && tc.errorContains == "profile not found" {
				cfg = testutils.NewTestConfig(t).Get()
			} else {
				profile := &config.Profile{
					Name:       tc.profileName,
					ConfigFile: configFile,
					ConfigType: utils.JustPtr(config.Template),
					Conditions: &config.ProfileCondition{
						RequiredMonitors: []*config.RequiredMonitor{
							{Name: utils.StringPtr("eDP-1")},
						},
					},
				}

				cfg = testutils.NewTestConfig(t).
					WithProfiles(map[string]*config.Profile{
						tc.profileName: profile,
					}).
					Get()
			}

			if tc.inputFile != "testdata/non_existent.conf" {
				inputData, err := os.ReadFile(tc.inputFile)
				require.NoError(t, err)
				err = os.WriteFile(configFile, inputData, 0o600)
				require.NoError(t, err)
			}

			service := profilemaker.NewService(cfg, nil)

			// For repeated calls test, call EditExisting multiple times
			if strings.Contains(tc.name, "Repeated calls") {
				for i := 0; i < 3; i++ {
					err = service.EditExisting(tc.profileName, monitors)
					if err != nil {
						break
					}
				}
			} else {
				err = service.EditExisting(tc.profileName, monitors)
			}

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			assert.NoError(t, err)

			testutils.AssertFixture(t, configFile, tc.expectedFile, *regenerate)
		})
	}
}

func TestService_ToLuaBlocks(t *testing.T) {
	monitors := []*hypr.MonitorSpec{
		{
			ID:          utils.IntPtr(1),
			Name:        "monA",
			Description: "New Monitor A",
			Width:       2560,
			Height:      1440,
			RefreshRate: 120.0,
			X:           0,
			Y:           0,
			Scale:       1.5,
			Transform:   0,
			Vrr:         false,
		},
		{
			ID:            utils.IntPtr(2),
			Name:          "monB",
			Description:   "New Monitor B",
			Width:         1920,
			Height:        1080,
			RefreshRate:   60.0,
			X:             2560,
			Y:             0,
			Scale:         1.0,
			Transform:     0,
			Mirror:        "eDP-1",
			Vrr:           true,
			CurrentFormat: "XRGB2101010",
			ColorPreset:   "hdr",
			SdrBrightness: 1.1,
			SdrSaturation: 0.98,
		},
		{
			ID:          utils.IntPtr(3),
			Name:        "monC",
			Description: "",
			Disabled:    true,
		},
	}

	for _, monitor := range monitors {
		require.NoError(t, monitor.Validate(), "monitor spec should be correct")
	}

	service := profilemaker.NewService(testutils.NewTestConfig(t).Get(), nil)
	blocks := service.ToLuaBlocks(monitors)

	require.Len(t, blocks, 3)
	assert.Equal(t, `hl.monitor({
    output = "desc:New Monitor A",
    mode = "2560x1440@120.00000",
    position = "0x0",
    scale = 1.50000000,
    transform = 0,
    vrr = 0,
})`, blocks[0])
	assert.Contains(t, blocks[1], `    bitdepth = 10,`)
	assert.Contains(t, blocks[1], `    cm = "hdr",`)
	assert.Contains(t, blocks[1], `    mirror = "eDP-1",`)
	assert.Equal(t, `hl.monitor({
    output = "monC",
    disabled = true,
})`, blocks[2])
}

func TestService_EditExistingLuaConfigFormat(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.lua")
	require.NoError(t, os.WriteFile(configFile, []byte(`# <<<<< TUI AUTO START
old()
# <<<<< TUI AUTO END
`), 0o600))

	profileName := "test-profile"
	profile := &config.Profile{
		Name:       profileName,
		ConfigFile: configFile,
		ConfigType: utils.JustPtr(config.Template),
		Conditions: &config.ProfileCondition{
			RequiredMonitors: []*config.RequiredMonitor{
				{Name: utils.StringPtr("eDP-1")},
			},
		},
	}

	cfg := testutils.NewTestConfig(t).
		WithProfiles(map[string]*config.Profile{profileName: profile}).
		WithConfigFormat(config.LuaConfigFormat).
		Get()

	monitors := []*hypr.MonitorSpec{
		{
			ID:          utils.IntPtr(1),
			Name:        "monA",
			Description: "New Monitor A",
			Width:       2560,
			Height:      1440,
			RefreshRate: 120.0,
			Scale:       1.5,
		},
	}
	require.NoError(t, monitors[0].Validate(), "monitor spec should be correct")

	service := profilemaker.NewService(cfg, nil)
	require.NoError(t, service.EditExisting(profileName, monitors))

	contents, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.Contains(t, string(contents), `-- <<<<< TUI AUTO START`)
	assert.Contains(t, string(contents), `hl.monitor({`)
	assert.Contains(t, string(contents), `    output = "desc:New Monitor A",`)
	assert.NotContains(t, string(contents), `monitor=`)
	assert.NotContains(t, string(contents), `# <<<<< TUI AUTO START`)
}
