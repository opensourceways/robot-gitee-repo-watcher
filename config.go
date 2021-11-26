package main

import (
	"fmt"
	"path/filepath"
)

type configuration struct {
	Config botConfig `json:"config"`
}

func (c *configuration) Validate() error {
	if c == nil {
		return nil
	}

	return c.Config.validate()
}

func (c *configuration) SetDefault() {
}

type repoBranch struct {
	Org    string `json:"org" required:"true"`
	Repo   string `json:"repo" required:"true"`
	Branch string `json:"branch" required:"true"`
}

// The repo which includes the repository and sig information that will be watched
type watchingFiles struct {
	repoBranch

	// RepoFilePath is the path to repo file. For example: repository/openeuler.yaml
	RepoFilePath string `json:"repo_file_path" required:"true"`

	// SigFilePath is the path to sig file. For example: sig/sigs.yaml
	SigFilePath string `json:"sig_file_path" required:"true"`

	// SigDir is the directory which includes all the sigs. For example: sig
	SigDir string `json:"sig_dir" required:"true"`
}

func (w *watchingFiles) validate() error {
	return nil
}

// obsMetaProject includes the information about the obs meta repo and the new project
type obsMetaProject struct {
	// Branch is the one which the project file will be writed to
	Branch repoBranch `json:"obs_repo" required:"true"`

	// ProjectDir is the diectory of the new project
	ProjectDir string `json:"project_dir" required:"true"`

	// ProjectFileName is the file name of new project
	ProjectFileName string `json:"project_file_name" required:"true"`

	// ProjectTemplatePath is the template file path which describes the new project
	ProjectTemplatePath string `json:"project_template_path" required:"true"`
}

func (o *obsMetaProject) genProjectFilePath(p string) string {
	return filepath.Join(o.ProjectDir, p, o.ProjectFileName)
}

func (o *obsMetaProject) validate() error {
	return nil
}

type botConfig struct {
	WatchingFiles watchingFiles `json:"watching_files" required:"true"`

	// ConcurrentSize is the concurrent size for doing task
	ConcurrentSize int `json:"concurrent_size" required:"true"`

	// Interval is the one between repo checkes. 0 or unset means check repos consecutively.
	// The unit is minute.
	Interval int `json:"interval,omitempty"`

	// EnableCreatingOBSMetaProject is the switch of creating project in obs meta repo
	EnableCreatingOBSMetaProject bool `json:"enable_creating_obs_meta_project,omitempty"`

	ObsMetaProject obsMetaProject `json:"obs_meta_project"`
}

func (c *botConfig) validate() error {
	if err := c.WatchingFiles.validate(); err != nil {
		return err
	}

	if c.ConcurrentSize <= 0 {
		return fmt.Errorf("concurrent_size must be bigger than 0")
	}

	if c.EnableCreatingOBSMetaProject {
		return c.ObsMetaProject.validate()
	}
	return nil
}
