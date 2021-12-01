package community

import (
	"fmt"
	"strings"
)

const BranchProtected = "protected"

type Repos struct {
	Version      string       `json:"version,omitempty"`
	Community    string       `json:"community" required:"true"`
	Repositories []Repository `json:"repositories,omitempty"`

	repos map[string]*Repository `json:"-"`
}

func (r *Repos) GetCommunity() string {
	if r == nil {
		return ""
	}
	return r.Community
}

func (r *Repos) GetRepos() map[string]*Repository {
	if r == nil {
		return nil
	}

	return r.repos
}

func (r *Repos) Validate() error {
	if r == nil {
		return fmt.Errorf("empty repos")
	}

	for i := range r.Repositories {
		if err := r.Repositories[i].validate(); err != nil {
			return fmt.Errorf("validate %d repository, err:%s", i, err.Error())
		}
	}

	r.convert()
	return nil
}

func (r *Repos) convert() {
	v := make(map[string]*Repository)

	items := r.Repositories
	for i := range items {
		item := &items[i]
		v[item.Name] = item
	}

	r.repos = v
}

type Repository struct {
	Name              string       `json:"name" required:"true"`
	Type              string       `json:"type" required:"true"`
	RenameFrom        string       `json:"rename_from,omitempty"`
	Description       string       `json:"description,omitempty"`
	Commentable       bool         `json:"commentable,omitempty"`
	ProtectedBranches []string     `json:"protected_branches,omitempty"`
	Branches          []RepoBranch `json:"branches,omitempty"`

	RepoMember
}

func (r *Repository) IsPrivate() bool {
	return r.Type == "private"
}

func (r *Repository) validate() error {
	if r.Name == "" {
		return fmt.Errorf("missing repo name")
	}

	if r.Type == "" {
		return fmt.Errorf("missing repo type")
	}

	for i := range r.Branches {
		if err := r.Branches[i].validate(); err != nil {
			return fmt.Errorf("validate %d branch, err:%s", i, err)
		}
	}

	if n := len(r.ProtectedBranches); n > 0 {
		v := make([]RepoBranch, n)
		for i, item := range r.ProtectedBranches {
			v[i] = RepoBranch{Name: item, Type: BranchProtected}
		}

		if len(r.Branches) > 0 {
			r.Branches = append(r.Branches, v...)
		} else {
			r.Branches = v
		}
	}

	return nil
}

type RepoMember struct {
	Viewers    []string `json:"viewers,omitempty"`
	Managers   []string `json:"managers,omitempty"`
	Reporters  []string `json:"reporters,omitempty"`
	Developers []string `json:"developers,omitempty"`
}

type RepoBranch struct {
	Name       string `json:"name" required:"true"`
	Type       string `json:"type,omitempty"`
	CreateFrom string `json:"create_from,omitempty"`
}

func (r *RepoBranch) validate() error {
	if r.Name == "" {
		return fmt.Errorf("missing branch name")
	}
	return nil
}

type Sigs struct {
	Items []Sig `json:"sigs,omitempty"`

	repoWithMultiSigs map[string]int `json:"-"`
}

func (s *Sigs) GetSigs() []Sig {
	if s == nil {
		return nil
	}

	return s.Items
}

func (s *Sigs) GetRepoWithMultiSigs() map[string]int {
	if s == nil {
		return nil
	}

	return s.repoWithMultiSigs
}

func (s *Sigs) Validate() error {
	if s == nil {
		return fmt.Errorf("empty sigs")
	}

	for i := range s.Items {
		if err := s.Items[i].validate(); err != nil {
			return fmt.Errorf("validate %d sig, err:%s", i, err)
		}
	}

	s.doStat()
	return nil
}

func (s *Sigs) doStat() {
	m := make(map[string]int)
	for i := range s.Items {
		item := s.Items[i].Repositories
		for _, r := range item {
			m[r] += 1
		}
	}

	v := make(map[string]int)
	for k, n := range m {
		if n > 1 {
			v[k] = n
		}
	}

	s.repoWithMultiSigs = v
}

type Sig struct {
	Name         string   `json:"name" required:"true"`
	Repositories []string `json:"repositories,omitempty"`
	repos        []string `json:"-"`
}

func (s *Sig) GetRepos() []string {
	if s == nil {
		return nil
	}

	return s.repos
}

func (s *Sig) validate() error {
	if s.Name == "" {
		return fmt.Errorf("missing sig name")
	}

	s.convert()
	return nil
}

func (s *Sig) convert() {
	v := make([]string, len(s.Repositories))

	for i, r := range s.Repositories {
		if a := strings.Split(r, "/"); len(a) > 1 {
			v[i] = a[1]
		} else {
			v[i] = r
		}

	}
	s.repos = v
}

type RepoOwners struct {
	Maintainers []string `json:"maintainers,omitempty"`
	Committers  []string `json:"committers,omitempty"`
	all         []string `json:"-"`
}

func (r *RepoOwners) GetOwners() []string {
	if r == nil {
		return nil
	}

	return r.all
}

func (r *RepoOwners) Validate() error {
	if r == nil {
		return fmt.Errorf("empty repo owners")
	}

	r.convert()

	return nil
}

func (r *RepoOwners) convert() {
	o := make([]string, len(r.Maintainers)+len(r.Committers))
	i := 0

	for _, item := range r.Maintainers {
		o[i] = strings.ToLower(item)
		i++
	}

	for _, item := range r.Committers {
		o[i] = strings.ToLower(item)
		i++
	}

	r.all = o
}
