package main

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
)

func (bot *robot) handleMember(expectRepo expectRepoInfo, localMembers []string, log *logrus.Entry) []string {
	org := expectRepo.org
	repo := expectRepo.getNewRepoName()

	if len(localMembers) == 0 {
		v, err := bot.cli.GetRepo(org, repo)
		if err != nil {
			log.WithError(err).Errorf("get repo:%s when handling repo members", repo)
			return nil
		}
		localMembers = v.Members
	}

	expect := sets.NewString(expectRepo.expectOwners...)
	lm := sets.NewString(localMembers...)
	r := expect.Intersection(lm).UnsortedList()

	// add new
	if v := expect.Difference(lm); v.Len() > 0 {
		for k := range v {
			// how about adding a member but he/she exits? see the comment of 'addRepoMember'
			if err := bot.addRepoMember(org, repo, k); err != nil {
				log.WithError(err).Errorf("add member:%s to repo:%s", k, repo)
			} else {
				r = append(r, k)
			}
		}
	}

	// remove
	if v := lm.Difference(expect); v.Len() > 0 {
		for k := range v {
			if err := bot.cli.RemoveRepoMember(org, repo, k); err != nil {
				log.WithError(err).Errorf("remove member:%s from repo:%s", k, repo)

				r = append(r, k)
			}
		}
	}

	return r
}

// Gitee api will be successful even if adding a member repeatedly.
func (bot *robot) addRepoMember(org, repo, login string) error {
	return bot.cli.AddRepoMember(org, repo, login, "push")
}
