package main

import (
	"encoding/base64"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

type expectRepos struct {
	loadFile func(string, interface{}) (string, error)

	path  string
	sha   string
	repos community.Repos
}

func (r *expectRepos) update(sha string) error {
	if sha == "" || sha == r.sha {
		return nil
	}

	v := new(community.Repos)
	sha, err := r.loadFile(r.path, v)
	if err != nil {
		return err
	}

	r.repos = *v
	r.sha = sha

	return nil
}

type orgSigs struct {
	loadFile func(string, interface{}) (string, error)

	path string
	sha  string
	sigs community.Sigs
}

func (r *orgSigs) update(sha string) error {
	if sha == r.sha {
		return nil
	}

	v := new(community.Sigs)
	sha, err := r.loadFile(r.path, v)
	if err != nil {
		return err
	}

	r.sigs = *v
	r.sha = sha

	return nil
}

type expectSigOwners struct {
	loadFile func(string, interface{}) (string, error)

	path   string
	sha    string
	owners community.RepoOwners
}

func (r *expectSigOwners) getOwners() []string {
	return r.owners.Maintainers
}

func (r *expectSigOwners) update(sha string) error {
	if sha == "" || sha == r.sha {
		return nil
	}

	v := new(community.RepoOwners)
	sha, err := r.loadFile(r.path, v)
	if err != nil {
		return err
	}

	r.owners = *v
	r.sha = sha

	return nil
}

type watchingRepoInfo struct {
	org    string
	repo   string
	branch string
}

type expectState struct {
	w   watchingRepoInfo
	cli iClient

	repos expectRepos

	sig orgSigs

	sigDir    string
	sigOwners map[string]*expectSigOwners
}

func (r *expectState) loadFile(path string, v interface{}) (string, error) {
	c, err := r.cli.GetPathContent(r.w.org, r.w.repo, path, r.w.branch)
	if err != nil {
		return "", err
	}

	if err = decodeYamlFile(c.Content, v); err != nil {
		return "", err
	}

	return c.Sha, nil
}

func (s *expectState) init(repoFilePath, sigFilePath, sigDir string) (string, error) {
	s.repos = expectRepos{
		loadFile: s.loadFile,
		path:     repoFilePath,
	}
	if err := s.repos.update("init"); err != nil {
		return "", err
	}

	s.sig = orgSigs{
		loadFile: s.loadFile,
		path:     sigFilePath,
	}
	if err := s.sig.update("init"); err != nil {
		return "", err
	}

	s.sigDir = sigDir

	return s.repos.repos.Community, nil
}

func (s *expectState) getSigOwner(sigName string) *expectSigOwners {
	o, ok := s.sigOwners[sigName]
	if !ok {
		o = &expectSigOwners{
			path:     filepath.Join(s.sigDir, sigName, "OWNERS"),
			loadFile: s.loadFile,
		}

		s.sigOwners[sigName] = o
	}

	return o
}

func (s *expectState) getAllRepos() []community.Repository {
	return s.repos.repos.Repositories
}

func (s *expectState) getAllSigs() []community.Sig {
	return s.sig.sigs.Sigs
}

func decodeYamlFile(content string, v interface{}) error {
	c, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(c, v)
}

func (s *expectState) check(isStopped func() bool, callback func([]string, *community.Repository)) {
	allFiles, err := s.listAllFilesOfRepo()
	if err != nil {
		// log
	}

	if err := s.repos.update(allFiles[s.repos.path]); err != nil {
		// log
	}

	repoMap := make(map[string]*community.Repository)
	v := s.getAllRepos()
	for i := range v {
		item := &v[i]
		repoMap[item.Name] = item
	}

	if err := s.sig.update(allFiles[s.sig.path]); err != nil {
		// log
	}

	sigs := s.getAllSigs()
	for i := range sigs {
		sig := &sigs[i]

		sigOwner := s.getSigOwner(sig.Name)

		if err := sigOwner.update(allFiles[sigOwner.path]); err != nil {
			//log
			// it should think about the unormal input data
		}

		for _, repoName := range sig.Repositories {
			if isStopped() {
				break
			}

			callback(sigOwner.getOwners(), repoMap[repoName])

			delete(repoMap, repoName)
		}

		if isStopped() {
			break
		}
	}

	for _, repo := range repoMap {
		if isStopped() {
			break
		}

		callback(nil, repo)
	}
}

func (s *expectState) listAllFilesOfRepo() (map[string]string, error) {
	trees, err := s.cli.GetDirectoryTree(s.w.org, s.w.repo, s.w.branch, 1)
	if err != nil || len(trees.Tree) == 0 {
		return nil, err
	}

	r := make(map[string]string)
	for i := range trees.Tree {
		item := &trees.Tree[i]
		r[item.Path] = item.Sha
	}

	return r, nil
}
