package main

import (
	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

type expectSigOwners struct {
	cli iClient

	owners community.RepoOwners
	file   repoFile
	sha    string
}

func (r *expectSigOwners) update(sha string) error {
	if sha == r.sha {
		return nil
	}

	c, err := r.cli.GetPathContent(r.file.org, r.file.repo, r.file.path, r.file.branch)
	if err != nil {
		return err
	}

	v := new(community.RepoOwners)
	if err = decodeYamlFile(c.Content, v); err != nil {
		return err
	}

	r.owners = *v
	r.sha = c.Sha

	return nil
}

func (bot *robot) initExpectSigOwners(f repoFile) (*expectSigOwners, error) {
	r := &expectSigOwners{
		cli:  bot.cli,
		file: f,
	}

	return r, r.update("init")
}
