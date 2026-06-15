package cmd

import (
	"flag"
	"fmt"
	"path/filepath"
)

type projectPaths struct {
	ProjectPath string
	VaultPath   string
	DBPath      string
}

// Use changes the saved Recall project directory without prompting. It is safe
// to point at an existing folder: existing files are left in place and missing
// Recall scaffold directories are created.
func Use(args []string) error {
	fs := flag.NewFlagSet("use", flag.ContinueOnError)
	pathFlag := fs.String("path", "", "project directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var path string
	if *pathFlag != "" {
		if fs.NArg() != 0 {
			return fmt.Errorf("usage: recall use <path>")
		}
		path = *pathFlag
	} else if fs.NArg() == 1 {
		path = fs.Arg(0)
	} else {
		return fmt.Errorf("usage: recall use <path>")
	}

	out, err := useProject(path)
	if err != nil {
		return err
	}
	fmt.Printf("project stored at: %s\nvault: %s\ndb: %s\n", out.ProjectPath, out.VaultPath, out.DBPath)
	return nil
}

func useProject(path string) (projectPaths, error) {
	resolved, err := resolvePath(path)
	if err != nil {
		return projectPaths{}, err
	}
	if err := createDataDirectoryScaffold(resolved); err != nil {
		return projectPaths{}, err
	}
	if err := saveConfig(Config{ProjectPath: resolved}); err != nil {
		return projectPaths{}, err
	}
	return projectPaths{
		ProjectPath: resolved,
		VaultPath:   filepath.Join(resolved, "vault"),
		DBPath:      filepath.Join(resolved, "db", "recall.sqlite"),
	}, nil
}
