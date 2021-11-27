package main

import (
	"strconv"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
	"github.com/opensourceways/robot-gitee-repo-watcher/models"
)

func (bot *robot) createRepo(
	expectRepo expectRepoInfo,
	log *logrus.Entry,
	hook func(string, *logrus.Entry),
) models.RepoState {
	org := expectRepo.org
	repo := expectRepo.expectRepoState
	repoName := expectRepo.getNewRepoName()

	if n := repo.RenameFrom; n != "" && n != repoName {
		return bot.renameRepo(org, repoName, n, log, hook)
	}

	err := bot.cli.CreateRepo(org, sdk.RepositoryPostParam{
		Name:        repoName,
		Description: repo.Description,
		HasIssues:   true,
		HasWiki:     true,
		AutoInit:    true, // set `auto_init` as true to initialize `master` branch with README after repo creation
		CanComment:  repo.Commentable,
		Private:     repo.IsPrivate(),
	})
	if err != nil {
		l := log.WithField("action", "create repo")
		if s, b := bot.getRepoState(org, repoName, l); b {
			return s
		}

		log.WithError(err).Errorf("create repo:%s", repoName)

		return models.RepoState{}
	}

	defer func() {
		hook(repoName, log)
	}()

	if err := bot.initRepoReviewer(org, repoName); err != nil {
		log.WithError(err).Errorf("initialize the reviewers for new created repo:%s", repoName)
	}

	branches := []community.RepoBranch{}
	for _, item := range repo.Branches {
		if item.Name == "master" {
			continue
		}

		if b, ok := bot.createBranch(org, repoName, item, log); ok {
			branches = append(branches, b)
		}
	}

	members := []string{}
	for _, item := range expectRepo.expectOwners {
		if err := bot.addRepoMember(org, repoName, item); err != nil {
			logrus.WithError(err).Errorf(
				"add member:%s to repo:%s when creating it", item, repoName,
			)
		} else {
			members = append(members, item)
		}
	}

	return models.RepoState{
		Available: true,
		Branches:  branches,
		Members:   members,
		Property: models.RepoProperty{
			CanComment: repo.Commentable,
			Private:    repo.IsPrivate(),
		},
	}
}

func (bot *robot) renameRepo(
	org, newRepo, oldRepo string,
	log *logrus.Entry,
	hook func(string, *logrus.Entry),
) models.RepoState {
	err := bot.cli.UpdateRepo(
		org,
		oldRepo,
		sdk.RepoPatchParam{
			Name: newRepo,
			Path: newRepo,
		},
	)

	defer func(b bool) {
		if b {
			hook(newRepo, log)
		}
	}(err == nil)

	// if the err == nil, invoke 'getRepoState' obviously.
	// if the err != nil, it is better to call 'getRepoState' to
	// avoid the case that the repo already exists.
	l := log.WithField("action", "rename from repo:"+oldRepo)
	if s, b := bot.getRepoState(org, newRepo, l); b {
		return s
	}

	if err != nil {
		log.WithError(err).Errorf("rename repo:%s to %s", oldRepo, newRepo)

		return models.RepoState{}
	}

	return models.RepoState{Available: true}
}

func (bot *robot) getRepoState(org, repo string, log *logrus.Entry) (models.RepoState, bool) {
	newRepo, err := bot.cli.GetRepo(org, repo)
	if err != nil {
		log.WithError(err).Errorf("get repo:%s", repo)
		return models.RepoState{}, false
	}

	r := models.RepoState{
		Available: true,
		Members:   newRepo.Members,
		Property: models.RepoProperty{
			Private:    newRepo.Private,
			CanComment: newRepo.CanComment,
		},
	}

	branches, err := bot.listAllBranchOfRepo(org, repo)
	if err != nil {
		log.WithError(err).Errorf("list branch of repo:%s", repo)
	} else {
		r.Branches = branches
	}
	return r, true
}

func (bot *robot) initRepoReviewer(org, repo string) error {
	return bot.cli.SetRepoReviewer(
		org,
		repo,
		sdk.SetRepoReviewer{
			Assignees:       " ", // This parameter is a required one according to the Gitee API
			Testers:         " ", // Ditto
			AssigneesNumber: 0,
			TestersNumber:   0,
		},
	)
}

func (bot *robot) updateRepo(expectRepo expectRepoInfo, lp models.RepoProperty, log *logrus.Entry) models.RepoProperty {
	org := expectRepo.org
	repo := expectRepo.expectRepoState
	repoName := expectRepo.getNewRepoName()

	ep := repo.IsPrivate()
	ec := repo.Commentable
	if ep != lp.Private || ec != lp.CanComment {
		err := bot.cli.UpdateRepo(
			org,
			repoName,
			sdk.RepoPatchParam{
				Name:       repoName,
				CanComment: strconv.FormatBool(ec),
				Private:    strconv.FormatBool(ep),
			},
		)
		if err == nil {
			return models.RepoProperty{
				Private:    ep,
				CanComment: ec,
			}
		}

		log.WithError(err).WithFields(logrus.Fields{
			"Private":    ep,
			"CanComment": ec,
		}).Errorf("update repo:%s", repoName)
	}
	return lp
}
