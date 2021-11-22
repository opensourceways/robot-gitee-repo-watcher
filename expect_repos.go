package main

import (
	"encoding/base64"

	"sigs.k8s.io/yaml"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

type repoFile struct {
	org    string
	repo   string
	branch string
	path   string
}

type expectRepos struct {
	cli iClient

	repos community.Repos
	file  repoFile
	sha   string
}

func (r *expectRepos) update(sha string) error {
	if sha == r.sha {
		return nil
	}

	c, err := r.cli.GetPathContent(r.file.org, r.file.repo, r.file.path, r.file.branch)
	if err != nil {
		return err
	}

	v := new(community.Repos)
	if err = decodeYamlFile(c.Content, v); err != nil {
		return err
	}

	r.repos = *v
	r.sha = c.Sha

	return nil
}

func (bot *robot) initExpectRepos(f repoFile) (*expectRepos, error) {
	r := &expectRepos{
		cli:  bot.cli,
		file: f,
	}

	return r, r.update("init")
}

func decodeYamlFile(content string, v interface{}) error {
	c, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(c, v)
}
