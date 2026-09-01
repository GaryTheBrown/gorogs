package system

import (
	"fmt"
	"gorogs/logger"

	"os"
	"path/filepath"
	"plugin"
)

func LoadPlugins(pluginDir string) error {
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		logger.DebugF(logName, "Plugin allocation space %s missing; skipping scanning.", pluginDir)
		return nil
	}

	files, err := filepath.Glob(filepath.Join(pluginDir, "gorogs-*.so"))
	if err != nil {
		return fmt.Errorf("failed to scan for plugins: %w", err)
	}

	for _, file := range files {
		logger.DebugF(logName, "Mounting appliance module: %s", filepath.Base(file))

		p, err := plugin.Open(file)
		if err != nil {
			return fmt.Errorf("runtime linker failure loading %s: %w", file, err)
		}

		symInstance, err := p.Lookup("SystemInstance")
		if err != nil {
			return fmt.Errorf("plugin object %s is missing exported symbol 'SystemInstance': %w", file, err)
		}

		sys, ok := symInstance.(System)
		if !ok {
			return fmt.Errorf("plugin object %s does not satisfy systeminterface.System", file)
		}

		Register(sys)
		logger.InfoF(logName, "Successfully registered plugin: %s", sys.Name())
	}

	logger.Info(logName, "Running case-insensitive topological dependency sort loop...")
	return InitializeAndSort()
}
