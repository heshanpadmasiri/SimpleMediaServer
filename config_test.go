package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test loadConfigFromFile function

func TestLoadConfigFromFile_ValidSingleSource_LoadsCorrectly(t *testing.T) {
	// Create temporary directory and file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.toml")

	// Create a test media directory
	mediaDir := filepath.Join(tempDir, "media")
	err := os.Mkdir(mediaDir, 0755)
	assert.NoError(t, err)

	configContent := `MediaSource = "` + mediaDir + `"
Port = 8080`

	err = os.WriteFile(tempFile, []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfigFromFile(tempFile)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, 8080, config.Port)
	assert.Len(t, config.MediaSources, 1)
	assert.Equal(t, mediaDir, config.MediaSources[0])
}

func TestLoadConfigFromFile_ValidMultipleSources_LoadsCorrectly(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.toml")

	// Create multiple test media directories
	mediaDir1 := filepath.Join(tempDir, "media1")
	mediaDir2 := filepath.Join(tempDir, "media2")
	err := os.Mkdir(mediaDir1, 0755)
	assert.NoError(t, err)
	err = os.Mkdir(mediaDir2, 0755)
	assert.NoError(t, err)

	configContent := `MediaSources = ["` + mediaDir1 + `", "` + mediaDir2 + `"]
Port = 9000`

	err = os.WriteFile(tempFile, []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfigFromFile(tempFile)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, 9000, config.Port)
	assert.Len(t, config.MediaSources, 2)
	assert.Equal(t, mediaDir1, config.MediaSources[0])
	assert.Equal(t, mediaDir2, config.MediaSources[1])
}

func TestLoadConfigFromFile_DefaultPort_UsesDefault8080(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.toml")

	mediaDir := filepath.Join(tempDir, "media")
	err := os.Mkdir(mediaDir, 0755)
	assert.NoError(t, err)

	configContent := `MediaSource = "` + mediaDir + `"`

	err = os.WriteFile(tempFile, []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfigFromFile(tempFile)

	assert.NoError(t, err)
	assert.Equal(t, 8080, config.Port) // Default port
}

func TestLoadConfigFromFile_NonexistentFile_ReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	nonexistentFile := filepath.Join(tempDir, "nonexistent.toml")

	config, err := loadConfigFromFile(nonexistentFile)

	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestLoadConfigFromFile_InvalidTOML_ReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.toml")

	invalidContent := `MediaSource = ` // Invalid TOML

	err := os.WriteFile(tempFile, []byte(invalidContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfigFromFile(tempFile)

	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestLoadConfigFromFile_NoMediaSourceOrSources_ReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.toml")

	configContent := `Port = 8080` // No media source

	err := os.WriteFile(tempFile, []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfigFromFile(tempFile)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "either MediaSource or MediaSources is required")
}

func TestLoadConfigFromFile_NonexistentMediaPath_ReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.toml")

	nonexistentPath := filepath.Join(tempDir, "nonexistent")
	configContent := `MediaSource = "` + nonexistentPath + `"
Port = 8080`

	err := os.WriteFile(tempFile, []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfigFromFile(tempFile)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "media source path does not exist")
}

func TestLoadConfigFromFile_BothMediaSourceAndSources_PrefersSources(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.toml")

	mediaDir1 := filepath.Join(tempDir, "media1")
	mediaDir2 := filepath.Join(tempDir, "media2")
	mediaDir3 := filepath.Join(tempDir, "media3")
	err := os.Mkdir(mediaDir1, 0755)
	assert.NoError(t, err)
	err = os.Mkdir(mediaDir2, 0755)
	assert.NoError(t, err)
	err = os.Mkdir(mediaDir3, 0755)
	assert.NoError(t, err)

	// Both MediaSource and MediaSources provided
	configContent := `MediaSource = "` + mediaDir1 + `"
MediaSources = ["` + mediaDir2 + `", "` + mediaDir3 + `"]
Port = 8080`

	err = os.WriteFile(tempFile, []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfigFromFile(tempFile)

	assert.NoError(t, err)
	// Should prefer MediaSources over MediaSource
	assert.Len(t, config.MediaSources, 2)
	assert.Equal(t, mediaDir2, config.MediaSources[0])
	assert.Equal(t, mediaDir3, config.MediaSources[1])
}

func TestLoadConfigFromFile_EmptyMediaSources_FallsBackToMediaSource(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.toml")

	mediaDir := filepath.Join(tempDir, "media")
	err := os.Mkdir(mediaDir, 0755)
	assert.NoError(t, err)

	// Empty MediaSources array, should fall back to MediaSource
	configContent := `MediaSource = "` + mediaDir + `"
MediaSources = []
Port = 8080`

	err = os.WriteFile(tempFile, []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfigFromFile(tempFile)

	assert.NoError(t, err)
	assert.Len(t, config.MediaSources, 1)
	assert.Equal(t, mediaDir, config.MediaSources[0])
}

// Test loadConfig function (searches multiple locations)

func TestLoadConfig_CurrentDirectory_LoadsCorrectly(t *testing.T) {
	// Save original directory
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(originalDir)

	// Create temp directory and change to it
	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	// Create config.toml in current directory
	mediaDir := filepath.Join(tempDir, "media")
	err = os.Mkdir(mediaDir, 0755)
	assert.NoError(t, err)

	configContent := `MediaSource = "` + mediaDir + `"
Port = 7777`

	err = os.WriteFile("config.toml", []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfig()

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, 7777, config.Port)
	assert.Len(t, config.MediaSources, 1)
}

func TestLoadConfig_HomeConfigDirectory_LoadsCorrectly(t *testing.T) {
	// Save original directory
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(originalDir)

	// Create temp directory without config.toml
	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Create temporary home directory
	tempHome := filepath.Join(tempDir, "testhome")
	err = os.Mkdir(tempHome, 0755)
	assert.NoError(t, err)
	os.Setenv("HOME", tempHome)

	// Create .config/simpleMediaServer directory
	configDir := filepath.Join(tempHome, ".config", "simpleMediaServer")
	err = os.MkdirAll(configDir, 0755)
	assert.NoError(t, err)

	// Create media directory
	mediaDir := filepath.Join(tempHome, "media")
	err = os.Mkdir(mediaDir, 0755)
	assert.NoError(t, err)

	// Create config.toml in home config directory
	configPath := filepath.Join(configDir, "config.toml")
	configContent := `MediaSource = "` + mediaDir + `"
Port = 6666`

	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfig()

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, 6666, config.Port)
}

func TestLoadConfig_NoConfigFound_ReturnsError(t *testing.T) {
	// Save original directory
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(originalDir)

	// Create temp directory without config.toml
	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set HOME to temp directory without config
	tempHome := filepath.Join(tempDir, "testhome")
	err = os.Mkdir(tempHome, 0755)
	assert.NoError(t, err)
	os.Setenv("HOME", tempHome)

	config, err := loadConfig()

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "no configuration file found")
}

func TestLoadConfig_CurrentDirectoryTakesPrecedence_OverHomeConfig(t *testing.T) {
	// Save original directory
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(originalDir)

	// Create temp directory
	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Create temporary home directory
	tempHome := filepath.Join(tempDir, "testhome")
	err = os.Mkdir(tempHome, 0755)
	assert.NoError(t, err)
	os.Setenv("HOME", tempHome)

	// Create media directories
	currentMediaDir := filepath.Join(tempDir, "current_media")
	err = os.Mkdir(currentMediaDir, 0755)
	assert.NoError(t, err)

	homeMediaDir := filepath.Join(tempHome, "home_media")
	err = os.Mkdir(homeMediaDir, 0755)
	assert.NoError(t, err)

	// Create config in current directory
	currentConfigContent := `MediaSource = "` + currentMediaDir + `"
Port = 5555`
	err = os.WriteFile("config.toml", []byte(currentConfigContent), 0644)
	assert.NoError(t, err)

	// Create config in home directory
	configDir := filepath.Join(tempHome, ".config", "simpleMediaServer")
	err = os.MkdirAll(configDir, 0755)
	assert.NoError(t, err)
	homeConfigPath := filepath.Join(configDir, "config.toml")
	homeConfigContent := `MediaSource = "` + homeMediaDir + `"
Port = 4444`
	err = os.WriteFile(homeConfigPath, []byte(homeConfigContent), 0644)
	assert.NoError(t, err)

	config, err := loadConfig()

	assert.NoError(t, err)
	assert.NotNil(t, config)
	// Should use current directory config (port 5555) not home config (port 4444)
	assert.Equal(t, 5555, config.Port)
	assert.Equal(t, currentMediaDir, config.MediaSources[0])
}
