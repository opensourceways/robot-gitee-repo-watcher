package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func (bot *robot) createOBSMetaProject(repo string, log *logrus.Entry) {
	cfg := bot.cfg
	if !cfg.EnableCreatingOBSMetaProject {
		return
	}

	cfgOBS := &cfg.ObsMetaProject

	path := cfgOBS.genProjectFilePath(repo)
	b := &cfgOBS.Branch

	// file exists
	if _, err := bot.cli.GetPathContent(b.Org, b.Repo, path, b.Branch); err == nil {
		return
	}

	content, err := cfgOBS.genProjectFileContent(repo)
	if err != nil {
		log.WithError(err).Errorf("generate file of project:%s", repo)
		return
	}

	w := &cfg.WatchingFiles
	msg := fmt.Sprintf(
		"add project according to the file: %s/%s/%s:%s",
		w.Org, w.Repo, w.Branch, w.RepoFilePath,
	)

	_, err = bot.cli.CreateFile(b.Org, b.Repo, b.Branch, path, content, msg)
	if err != nil {
		log.WithError(err).Errorf("ceate file: %s", path)
	}
}
