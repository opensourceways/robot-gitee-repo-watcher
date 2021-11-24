package main

import (
	"k8s.io/apimachinery/pkg/util/sets"
)

func (bot *robot) handleMember(expectRepo *expectRepoInfo, localMembers []string) []string {
	org := expectRepo.org
	repo := expectRepo.expectRepoState.Name

	if len(localMembers) == 0 {
		v, err := bot.cli.GetGiteeRepo(org, repo)
		if err != nil {
			return nil
		}
		localMembers = v.Members
	}

	expect := sets.NewString(expectRepo.expectOwners...)
	lm := sets.NewString(localMembers...)

	newMembers := expect.Intersection(lm).UnsortedList()

	// add new
	if v := expect.Difference(lm); v.Len() > 0 {
		for k := range v {
			// 2. add memeber but it exits
			if err := bot.addRepoMember(org, repo, k); err != nil {
				// log
			} else {
				newMembers = append(newMembers, k)
			}

		}
	}

	// remove
	if v := lm.Difference(expect); v.Len() > 0 {
		for k := range v {
			if err := bot.cli.RemoveRepoMember(org, repo, k); err != nil {
				// log
				newMembers = append(newMembers, k)
			}

		}
	}

	return newMembers
}

func (bot *robot) addRepoMember(org, repo, login string) error {
	return bot.cli.AddRepoMember(org, repo, login, "push")
}
