package models

import (
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

var empty = struct{}{}

type Repo1 struct {
	name      string
	available bool
	property  RepoProperty

	start chan struct{}

	branches []community.RepoBranch
	members  sets.String

	state RepoState
}

type AfterUpdate struct {
	NewCreated bool
	Branches   []community.RepoBranch
	Members    []string
	Property   RepoProperty
}

type RepoState struct {
	Available bool
	Branches  []community.RepoBranch
	Members   []string
	Property  RepoProperty
}

type RepoProperty struct {
	Private    bool
	CanComment bool
}

type Repo struct {
	name  string
	state RepoState
	start chan struct{}
}

func NewRepo(repo string, state RepoState) *Repo {
	return &Repo{
		name:  repo,
		state: state,
		start: make(chan struct{}),
	}
}

func (r *Repo) Update(f func(RepoState) RepoState) {
	select {
	case r.start <- empty:
		defer func() {
			<-r.start
		}()

		r.state = f(r.state)
	default:
	}
}
