package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	content := "# comment\n" +
		"TEST_DOTENV_PLAIN=hello\n" +
		"TEST_DOTENV_QUOTED=\"world\"\n" +
		"TEST_DOTENV_EXISTING=from_file\n"
	require.NoError(t, os.WriteFile(envPath, []byte(content), 0o600))

	t.Setenv("TEST_DOTENV_EXISTING", "from_os")
	t.Cleanup(func() {
		_ = os.Unsetenv("TEST_DOTENV_PLAIN")
		_ = os.Unsetenv("TEST_DOTENV_QUOTED")
	})

	require.NoError(t, config.LoadDotEnv(envPath))
	assert.Equal(t, "hello", os.Getenv("TEST_DOTENV_PLAIN"))
	assert.Equal(t, "world", os.Getenv("TEST_DOTENV_QUOTED"))
	assert.Equal(t, "from_os", os.Getenv("TEST_DOTENV_EXISTING"))
}

func TestLoadDotEnv_MissingFileIgnored(t *testing.T) {
	err := config.LoadDotEnv(filepath.Join(t.TempDir(), "nonexistent.env"))
	assert.NoError(t, err)
}
