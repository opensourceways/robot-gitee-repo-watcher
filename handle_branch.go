package main

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

func (bot *robot) handleBranch(expectRepo expectRepoInfo, localBranches []community.RepoBranch, log *logrus.Entry) []community.RepoBranch {
	org := expectRepo.org
	repo := expectRepo.expectRepoState.Name

	if len(localBranches) == 0 {
		v, err := bot.listAllBranchOfRepo(org, repo)
		if err != nil {
			log.WithError(err).Errorf("list all branch of repo:%s", repo)
			return nil
		}
		localBranches = v
	}

	bsExpect := genBranchSets(expectRepo.expectRepoState.Branches)
	bsLocal := genBranchSets(localBranches)
	newState := []community.RepoBranch{}

	// update
	if v := bsExpect.intersectionByName(&bsLocal); len(v) > 0 {
		for name := range v {
			eb := bsExpect.get(name)
			lb := bsLocal.get(name)
			if eb.Type != lb.Type {
				err := bot.updateBranch(
					org, repo, name, eb.Type == community.BranchProtected,
				)
				if err == nil {
					newState = append(newState, *eb)
					continue
				} else {
					logrus.WithError(err).WithField("type", eb.Type).Errorf(
						"update branch:%s of repo:%s", name, repo,
					)
				}
			}
			newState = append(newState, *lb)
		}
	}

	// add new
	if v := bsExpect.differenceByName(&bsLocal); len(v) > 0 {
		for _, item := range v {
			// 1. create branch but it exits;
			if b, ok := bot.createBranch(org, repo, item, log); ok {
				newState = append(newState, b)
			}
		}
	}

	return newState
}

func (bot *robot) createBranch(org, repo string, branch community.RepoBranch, log *logrus.Entry) (community.RepoBranch, bool) {
	err := bot.cli.CreateBranch(org, repo, branch.Name, branch.CreateFrom)
	if err != nil {
		log.WithError(err).WithField("CreateFrom", branch.CreateFrom).Errorf(
			"create branch:%s of repo:%s", branch.Name, repo,
		)
		return community.RepoBranch{}, false
	}

	if branch.Type == community.BranchProtected {
		if err := bot.cli.SetProtectionBranch(org, repo, branch.Name); err != nil {
			log.WithError(err).Errorf(
				"set the branch:%s of repo:%s to be protected when creating it",
				branch.Name, repo,
			)

			return community.RepoBranch{
				Name:       branch.Name,
				CreateFrom: branch.CreateFrom,
			}, true
		}
	}

	return branch, true
}

func (bot *robot) updateBranch(org, repo, branch string, protected bool) error {
	if protected {
		return bot.cli.SetProtectionBranch(org, repo, branch)
	}
	return bot.cli.CancelProtectionBranch(org, repo, branch)
}

func (bot *robot) listAllBranchOfRepo(org, repo string) ([]community.RepoBranch, error) {
	items, err := bot.cli.GetRepoAllBranch(org, repo)
	if err != nil {
		return nil, err
	}

	v := make([]community.RepoBranch, len(items))

	for i := range items {
		item := &items[i]

		v[i] = community.RepoBranch{Name: item.Name}
		if item.Protected {
			v[i].Type = community.BranchProtected
		}
	}
	return v, nil
}

type branchSets struct {
	b     []community.RepoBranch
	s     sets.String
	index map[string]int
}

func (bs *branchSets) intersectionByName(bs1 *branchSets) sets.String {
	return bs.s.Intersection(bs1.s)
}

func (bs *branchSets) differenceByName(bs1 *branchSets) []community.RepoBranch {
	v := bs.s.Difference(bs1.s)
	n := v.Len()
	if n == 0 {
		return nil
	}

	r := make([]community.RepoBranch, n)
	i := 0
	for k := range v {
		r[i] = *bs.get(k)
		i++
	}
	return r
}

func (bs *branchSets) get(name string) *community.RepoBranch {
	if i, ok := bs.index[name]; ok {
		return &bs.b[i]
	}

	return nil
}

func genBranchSets(b []community.RepoBranch) branchSets {
	index := map[string]int{}
	s := make([]string, len(b))
	for i := range b {
		name := b[i].Name
		index[name] = i
		s[i] = name
	}

	return branchSets{b, sets.NewString(s...), index}
}
