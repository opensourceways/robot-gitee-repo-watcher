package main

import (
	"sync"

	"github.com/opensourceways/robot-gitee-repo-watcher/models"
)

type actualRepoState struct {
	repos map[string]*models.Repo
	mut   sync.RWMutex
}

func (r *actualRepoState) getOrNewARepo(repo string) *models.Repo {
	r.mut.RLock()
	v, ok := r.repos[repo]
	r.mut.RUnlock()

	if !ok {
		r.mut.Lock()
		if v, ok = r.repos[repo]; !ok {
			v = models.NewRepo(repo, false, nil)
			r.repos[repo] = v
		}
		r.mut.Unlock()
	}

	return v
}

func (bot *robot) loadALLRepos(org string) (*actualRepoState, error) {
	items, err := bot.cli.GetRepos(org)
	if err != nil {
		return nil, err
	}

	r := actualRepoState{
		repos: make(map[string]*models.Repo),
	}

	for i := range items {
		item := &items[i]
		r.repos[item.Path] = models.NewRepo(item.Path, true, item.Members)
	}

	return &r, nil
}
