package main

import (
	"strconv"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
	"github.com/opensourceways/robot-gitee-repo-watcher/models"
)

func (bot *robot) createRepo(expectRepo expectRepoInfo, log *logrus.Entry) models.RepoState {
	org := expectRepo.org
	repo := expectRepo.expectRepoState
	repoName := repo.Name

	if repo.RenameFrom != "" && repo.RenameFrom != repoName {
		return bot.renameRepo(org, repo, log)
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
		log.WithError(err).Errorf("create repo:%s", repoName)

		return models.RepoState{}
	}

	if err := bot.initRepoReviewer(org, repoName); err != nil {
		log.WithError(err).Errorf("initialize the reviewers for new created repo:%s", repoName)
	}

	branches := []community.RepoBranch{}
	for _, item := range repo.Branches {
		if item.Name == "master" {
			continue
		}

		if item.CreateFrom == "" {
			item.CreateFrom = "master"
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

func (bot *robot) renameRepo(org string, repo *community.Repository, log *logrus.Entry) models.RepoState {
	repoName := repo.Name

	err := bot.cli.UpdateRepo(
		org,
		repo.RenameFrom,
		sdk.RepoPatchParam{
			Name: repoName,
			Path: repoName,
		},
	)

	if err != nil {
		log.WithError(err).Errorf("rename repo:%s to %s", repo.RenameFrom, repoName)

		return models.RepoState{}
	}

	r := models.RepoState{Available: true}

	branches, err := bot.listAllBranchOfRepo(org, repoName)
	if err != nil {
		log.WithError(err).Errorf(
			"list branch of repo:%s which is renamed from:%s",
			repoName, repo.RenameFrom,
		)
	} else {
		r.Branches = branches
	}

	newRepo, err := bot.cli.GetGiteeRepo(org, repoName)
	if err != nil {
		log.WithError(err).Errorf(
			"get repo:%s which is renamed from:%s",
			repoName, repo.RenameFrom,
		)
	} else {
		r.Members = newRepo.Members
		r.Property = models.RepoProperty{
			Private:    newRepo.Private,
			CanComment: newRepo.CanComment,
		}
	}

	return r
}

func (bot *robot) initRepoReviewer(org, repo string) error {
	return bot.cli.SetRepoReviewer(
		org,
		repo,
		sdk.SetRepoReviewer{
			Assignees:       " ", //TODO need set to " "?
			Testers:         " ",
			AssigneesNumber: 0,
			TestersNumber:   0,
		},
	)
}

func (bot *robot) updateRepo(expectRepo expectRepoInfo, lp models.RepoProperty, log *logrus.Entry) models.RepoProperty {
	org := expectRepo.org
	repo := expectRepo.expectRepoState
	repoName := repo.Name

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
