package local

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/hupe1980/mwaacli/pkg/util"
)

type InstallerOptions struct {
	RepoURL   string
	ClonePath string
	DagsPath  string
}

type Installer struct {
	airflowVersion string
	cwd            string
	opts           InstallerOptions
}

func NewInstaller(version string, optFns ...func(o *InstallerOptions)) (*Installer, error) {
	opts := InstallerOptions{
		RepoURL:   MWAALocalRunnerRepoURL,
		ClonePath: DefaultClonePath,
		DagsPath:  ".",
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	return &Installer{
		airflowVersion: version,
		cwd:            cwd,
		opts:           opts,
	}, nil
}

func (i *Installer) Run() error {
	// Check if directory exists and is not empty
	if err := util.EnsurePathIsEmptyOrNonExistent(i.opts.ClonePath); err != nil {
		return err
	}

	// Clone repository
	memStore := memory.NewStorage()
	fs := memfs.New()

	repo, err := git.Clone(memStore, fs, &git.CloneOptions{
		URL:           i.opts.RepoURL,
		ReferenceName: plumbing.ReferenceName(i.airflowVersion),
		Progress:      os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get repository head: %w", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return fmt.Errorf("failed to get commit object: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get tree from commit: %w", err)
	}

	err = tree.Files().ForEach(func(f *object.File) error {
		if matched, _ := regexp.MatchString(`^(mwaa-local-env|.github)`, f.Name); matched {
			// Skip files and directories
			return nil
		} else if strings.HasPrefix(f.Name, "dags") {
			return createFile(filepath.Join(i.cwd, i.opts.DagsPath), f)
		} else if matched, _ := regexp.MatchString(`^(plugins|requirements|startup_script)`, f.Name); matched {
			return createFile(filepath.Join(i.cwd, i.opts.ClonePath), f)
		}

		return createFile(filepath.Join(i.cwd, i.opts.ClonePath), f)
	})
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	// Create an empty directory for "db-data"
	dbDataPath := filepath.Join(i.cwd, i.opts.ClonePath, "db-data")
	if err := os.MkdirAll(dbDataPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create db-data directory: %w", err)
	}

	return nil
}

func createFile(path string, f *object.File) error {
	filePath := filepath.Join(path, f.Name)
	dirPath := filepath.Dir(filePath)

	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	reader, err := f.Blob.Reader()
	if err != nil {
		return fmt.Errorf("failed to get blob reader: %w", err)
	}
	defer reader.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
