package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"recall/internal/vault"
)

// Init initializes the recall workspace: it asks where the project should be
// stored (defaulting to an OS-appropriate path), persists that choice to the
// config file, and creates the project directory.
//
// The stored path is permanent: if a config already exists, Init reports it and
// makes no changes rather than overriding it.
func Init(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	pathFlag := fs.String("path", "", "project directory")
	force := fs.Bool("force", false, "overwrite existing config without prompting")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, found, err := loadConfig()
	if err != nil {
		return err
	}
	if found && cfg.ProjectPath != "" && !*force {
		fmt.Printf("recall already initialized.\nproject stored at: %s\n", cfg.ProjectPath)

		overwrite, err := promptYN("re-initialize and overwrite existing config?")
		if err != nil {
			return err
		}

		if !overwrite {
			fmt.Println("aborting.")
			return nil
		}
	}

	def, err := defaultProjectPath()
	if err != nil {
		return err
	}

	var path string
	if *pathFlag != "" {
		path, err = resolvePath(*pathFlag)
	} else {
		path, err = promptPath(def)
	}
	if err != nil {
		return err
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("creating project dir %s: %w", path, err)
	}

	cfg.ProjectPath = path
	if err := saveConfig(cfg); err != nil {
		return err
	}

	if err := createDataDirectoryScaffold(path); err != nil {
		return err
	}
	fmt.Println("created project directory scaffold.")

	cfgPath, _ := configPath()
	fmt.Printf("initialized recall.\nproject stored at: %s\nconfig: %s\n", path, cfgPath)
	return nil
}

// promptPath asks the user where to store the project, returning an absolute,
// cleaned path. An empty answer selects def.
func promptPath(def string) (string, error) {
	fmt.Printf("Where should recall store the project? [%s]: ", def)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		// EOF with no input (e.g. piped/non-interactive): fall back to default.
		return resolvePath(def)
	}

	in := strings.TrimSpace(line)
	if in == "" {
		return resolvePath(def)
	}
	return resolvePath(in)
}

// resolvePath expands a leading ~ and returns an absolute, cleaned path.
func resolvePath(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expanding ~: %w", err)
		}
		p = filepath.Join(home, strings.TrimPrefix(p, "~"))
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("resolving path %s: %w", p, err)
	}
	return abs, nil
}

// createDataDirectoryScaffold lays out the project: the db/ folder for the
// SQLite index and a fully scaffolded vault/ (predefined domain folders, each
// with a README describing what belongs there, plus the top-level index).
func createDataDirectoryScaffold(path string) error {
	if err := os.MkdirAll(filepath.Join(path, "db"), 0o755); err != nil {
		return fmt.Errorf("creating directory db: %w", err)
	}
	if err := vault.Open(filepath.Join(path, "vault")).Scaffold(); err != nil {
		return err
	}
	return nil
}
