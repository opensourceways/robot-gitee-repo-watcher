package main

import (
	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

type repoSigs struct {
	cli iClient

	sigs community.Sigs
	file repoFile
	sha  string
}

func (r *repoSigs) update(sha string) error {
	if sha == r.sha {
		return nil
	}

	c, err := r.cli.GetPathContent(r.file.org, r.file.repo, r.file.path, r.file.branch)
	if err != nil {
		return err
	}

	v := new(community.Sigs)
	if err = decodeYamlFile(c.Content, v); err != nil {
		return err
	}

	r.sigs = *v
	r.sha = c.Sha

	return nil
}

func (bot *robot) initRepoSigs(f repoFile) (*repoSigs, error) {
	r := &repoSigs{
		cli:  bot.cli,
		file: f,
	}

	return r, r.update("init")
}
