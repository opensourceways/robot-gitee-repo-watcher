package main

import (
	"encoding/base64"
	"fmt"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

type watchingFile struct {
	loadFile func(string) (string, string, error)

	path string
	sha  string
	obj  interface{}
}

type getSHAFunc func(string) string

func (w *watchingFile) update(f getSHAFunc, newObject func() interface{}) {
	if sha := f(w.path); sha == "" || sha == w.sha {
		return
	}

	c, sha, err := w.loadFile(w.path)
	if err != nil { // log
		return
	}

	v := newObject()
	if err := decodeYamlFile(c, v); err != nil {
		// log
	} else {
		w.obj = v
		w.sha = sha
	}
}

type expectRepos struct {
	wf watchingFile
}

func (r *expectRepos) refresh(f getSHAFunc) *community.Repos {
	r.wf.update(f, func() interface{} {
		return new(community.Repos)
	})

	if v, ok := r.wf.obj.(*community.Repos); ok {
		return v
	}
	return nil
}

type orgSigs struct {
	wf watchingFile
}

func (r *orgSigs) refresh(f getSHAFunc) *community.Sigs {
	r.wf.update(f, func() interface{} {
		return new(community.Sigs)
	})

	if v, ok := r.wf.obj.(*community.Sigs); ok {
		return v
	}
	return nil
}

type expectSigOwners struct {
	wf watchingFile
}

func (r *expectSigOwners) refresh(f getSHAFunc) *community.RepoOwners {
	r.wf.update(f, func() interface{} {
		return new(community.RepoOwners)
	})

	if v, ok := r.wf.obj.(*community.RepoOwners); ok {
		return v
	}
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

	sig       orgSigs
	repos     expectRepos
	sigDir    string
	sigOwners map[string]*expectSigOwners
}

func (s *expectState) init(repoFilePath, sigFilePath, sigDir string) (string, error) {
	s.repos = expectRepos{
		wf: watchingFile{
			loadFile: s.loadFile,
			path:     repoFilePath,
		},
	}

	v := s.repos.refresh(func(string) string {
		return "init"
	})

	org := v.GetCommunity()
	if org == "" {
		return "", fmt.Errorf("load repository failed")
	}

	s.sig = orgSigs{
		wf: watchingFile{
			loadFile: s.loadFile,
			path:     sigFilePath,
		},
	}

	s.sigDir = sigDir

	return org, nil
}

func (s *expectState) check(isStopped func() bool, callback func(*community.Repository, []string)) {
	allFiles, err := s.listAllFilesOfRepo()
	if err != nil {
		// log
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

			callback(repoMap[repoName], owners.GetOwners())

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

		callback(repo, nil)
	}
}

func (s *expectState) getSigOwner(sigName string) *expectSigOwners {
	o, ok := s.sigOwners[sigName]
	if !ok {
		o = &expectSigOwners{
			wf: watchingFile{
				path:     filepath.Join(s.sigDir, sigName, "OWNERS"),
				loadFile: s.loadFile,
			},
		}

		s.sigOwners[sigName] = o
	}

	return o
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

func (r *expectState) loadFile(path string) (string, string, error) {
	c, err := r.cli.GetPathContent(r.w.org, r.w.repo, path, r.w.branch)
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
