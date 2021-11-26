package main

import (
	"encoding/base64"
	"fmt"
	"path"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

type watchingFile struct {
	log      *logrus.Entry
	loadFile func(string) (string, string, error)

	file string
	sha  string
	obj  interface{}
}

type getSHAFunc func(string) string

func (w *watchingFile) update(f getSHAFunc, newObject func() interface{}) {
	if sha := f(w.file); sha == "" || sha == w.sha {
		return
	}

	c, sha, err := w.loadFile(w.file)
	if err != nil {
		w.log.WithError(err).Errorf("load file:%s", w.file)
		return
	}

	v := newObject()
	if err := decodeYamlFile(c, v); err != nil {
		w.log.WithError(err).Errorf("decode file:%s", w.file)
	} else {
		w.obj = v
		w.sha = sha
	}
}

type expectRepos struct {
	wf watchingFile
}

func (e *expectRepos) refresh(f getSHAFunc) *community.Repos {
	e.wf.update(f, func() interface{} {
		return new(community.Repos)
	})

	if v, ok := e.wf.obj.(*community.Repos); ok {
		return v
	}
	return nil
}

type orgSigs struct {
	wf watchingFile
}

func (s *orgSigs) refresh(f getSHAFunc) *community.Sigs {
	s.wf.update(f, func() interface{} {
		return new(community.Sigs)
	})

	if v, ok := s.wf.obj.(*community.Sigs); ok {
		return v
	}
	return nil
}

type expectSigOwners struct {
	wf watchingFile
}

func (e *expectSigOwners) refresh(f getSHAFunc) *community.RepoOwners {
	e.wf.update(f, func() interface{} {
		return new(community.RepoOwners)
	})

	if v, ok := e.wf.obj.(*community.RepoOwners); ok {
		return v
	}
	return nil
}

type expectState struct {
	log *logrus.Entry
	cli iClient

	w         repoBranch
	sig       orgSigs
	repos     expectRepos
	sigDir    string
	sigOwners map[string]*expectSigOwners
}

func (s *expectState) init(repoFilePath, sigFilePath, sigDir string) (string, error) {
	s.repos = expectRepos{s.newWatchingFile(repoFilePath)}

	v := s.repos.refresh(func(string) string {
		return "init"
	})

	org := v.GetCommunity()
	if org == "" {
		return "", fmt.Errorf("load repository failed")
	}

	s.sig = orgSigs{s.newWatchingFile(sigFilePath)}

	s.sigDir = sigDir

	return org, nil
}

func (s *expectState) check(isStopped func() bool, callback func(*community.Repository, []string, *logrus.Entry)) {
	allFiles, err := s.listAllFilesOfRepo()
	if err != nil {
		s.log.WithError(err).Error("list all file")

		allFiles = make(map[string]string)
	}

	getSHA := func(p string) string {
		return allFiles[p]
	}

	allRepos := s.repos.refresh(getSHA)
	repoMap := allRepos.GetRepos()

	allSigs := s.sig.refresh(getSHA)
	sigs := allSigs.GetSigs()
	for i := range sigs {
		sig := &sigs[i]

		sigOwner := s.getSigOwner(sig.Name)
		owners := sigOwner.refresh(getSHA)

		for _, repoName := range sig.Repositories {
			if isStopped() {
				break
			}

			callback(repoMap[repoName], owners.GetOwners(), s.log)

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

		callback(repo, nil, s.log)
	}
}

func (s *expectState) getSigOwner(sigName string) *expectSigOwners {
	o, ok := s.sigOwners[sigName]
	if !ok {
		o = &expectSigOwners{
			wf: s.newWatchingFile(
				path.Join(s.sigDir, sigName, "OWNERS"),
			),
		}

		s.sigOwners[sigName] = o
	}

	return o
}

func (s *expectState) newWatchingFile(p string) watchingFile {
	return watchingFile{
		file:     p,
		log:      s.log,
		loadFile: s.loadFile,
	}
}

func (s *expectState) listAllFilesOfRepo() (map[string]string, error) {
	trees, err := s.cli.GetDirectoryTree(s.w.Org, s.w.Repo, s.w.Branch, 1)
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

func (s *expectState) loadFile(f string) (string, string, error) {
	c, err := s.cli.GetPathContent(s.w.Org, s.w.Repo, f, s.w.Branch)
	if err != nil {
		return "", "", err
	}

	return c.Content, c.Sha, nil
}

func decodeYamlFile(content string, v interface{}) error {
	c, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(c, v)
}
