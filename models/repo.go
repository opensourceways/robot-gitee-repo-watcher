package models

import (
	"k8s.io/apimachinery/pkg/util/sets"
)

type Repo struct {
	name      string
	available bool
	start     chan bool
	branches  sets.String
	members   sets.String
}

type AfterUpdate struct {
	Available bool
	Branches  []string
	Members   []string
}

func (r *Repo) Update(f func(bool, sets.String, sets.String) (AfterUpdate, error)) {
	select {
	case r.start <- true:
		defer func() {
			<-r.start
		}()

		v, err := f(r.available, r.branches, r.members)
		if err == nil {
			r.available = v.Available
			r.branches = sets.NewString(v.Branches...)
			r.members = sets.NewString(v.Members...)
		}

	default:
	}
}

func NewRepo(repo string, available bool, members []string) *Repo {
	return &Repo{
		name:      repo,
		available: available,
		start:     make(chan bool),
		branches:  sets.NewString(),
		members:   sets.NewString(members...),
	}
}
