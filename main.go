package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/opensourceways/community-robot-lib/giteeclient"
	"github.com/opensourceways/community-robot-lib/interrupts"
	"github.com/opensourceways/community-robot-lib/logrusutil"
	liboptions "github.com/opensourceways/community-robot-lib/options"
	"github.com/opensourceways/community-robot-lib/secret"
	"github.com/sirupsen/logrus"
)

type options struct {
	gitee        liboptions.GiteeOptions
	watchingRepo string
	repoFilePath string
	sigFilePath  string
	sigDir       string
}

func (o *options) Validate() error {
	items := strings.Split(o.watchingRepo, "/")
	if len(items) != 3 {
		return fmt.Errorf("invalid watching_repo:%s", o.watchingRepo)
	}

	for _, item := range items {
		if item == "" {
			return fmt.Errorf("invalid watching_repo:%s", o.watchingRepo)
		}
	}

	v := map[string]string{
		o.repoFilePath: "repo-file-path",
		o.sigFilePath:  "sig-file-path",
		o.sigDir:       "sig-dir",
	}
	for p, s := range v {
		if p == "" {
			return fmt.Errorf("missing %v", s)
		}
	}

	return o.gitee.Validate()
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options

	o.gitee.AddFlags(fs)
	fs.StringVar(&o.watchingRepo, "watching_repo", "", "The repo which includes the repository and sig information that will be watched. The format is: org/repo/branch. For example: openeuler/community/master")
	fs.StringVar(&o.repoFilePath, "repo-file-path", "", "Path to repo file. For example: repository/openeuler.yaml")
	fs.StringVar(&o.sigFilePath, "sig-file-path", "", "Path to sig file. For example: sig/sigs.yaml")
	fs.StringVar(&o.sigDir, "sig-dir", "", "The directory which includes all the sigs. For example: sig")

	fs.Parse(args)
	return o
}

func main() {
	logrusutil.ComponentInit(botName)

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	secretAgent := new(secret.Agent)
	if err := secretAgent.Start([]string{o.gitee.TokenPath}); err != nil {
		logrus.WithError(err).Fatal("Error starting secret agent.")
	}
	defer secretAgent.Stop()

	c := giteeclient.NewClient(secretAgent.GetTokenGenerator(o.gitee.TokenPath))

	p := newRobot(c)

	defer interrupts.WaitForGracefulShutdown()

	interrupts.OnInterrupt(func() {
		p.stop()
	})

	p.run(o.watchingRep, o.repoFilePath, o.sigFilePath, o.sigDir)
}
