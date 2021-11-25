package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/opensourceways/community-robot-lib/giteeclient"
	"github.com/opensourceways/community-robot-lib/interrupts"
	"github.com/opensourceways/community-robot-lib/logrusutil"
	liboptions "github.com/opensourceways/community-robot-lib/options"
	"github.com/opensourceways/community-robot-lib/secret"
	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
)

type options struct {
	gitee          liboptions.GiteeOptions
	watchingRepo   string
	repoFilePath   string
	sigFilePath    string
	sigDir         string
	concurrentSize int
}

func (o *options) Validate() error {
	_, err := o.parseWatchingRepo()
	if err != nil {
		return err
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

	if o.concurrentSize <= 0 {
		return fmt.Errorf("concurrent size must be bigger than 0")
	}

	return o.gitee.Validate()
}

func (o *options) parseWatchingRepo() (watchingRepoInfo, error) {
	r := watchingRepoInfo{}

	v := o.watchingRepo
	items := strings.Split(v, "/")
	if len(items) != 3 {
		return r, fmt.Errorf("invalid watching-repo:%s", v)
	}

	for _, item := range items {
		if item == "" {
			return r, fmt.Errorf("invalid watching-repo:%s", v)
		}
	}

	r.org = items[0]
	r.repo = items[1]
	r.branch = items[2]
	return r, nil
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options

	o.gitee.AddFlags(fs)
	fs.StringVar(&o.watchingRepo, "watching-repo", "", "The repo which includes the repository and sig information that will be watched. The format is: org/repo/branch. For example: openeuler/community/master")
	fs.StringVar(&o.repoFilePath, "repo-file-path", "", "Path to repo file. For example: repository/openeuler.yaml")
	fs.StringVar(&o.sigFilePath, "sig-file-path", "", "Path to sig file. For example: sig/sigs.yaml")
	fs.StringVar(&o.sigDir, "sig-dir", "", "The directory which includes all the sigs. For example: sig")
	fs.IntVar(&o.concurrentSize, "concurrent-size", 500, "The concurrent size for doing task.")

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

	pool, err := newPool(o.concurrentSize, logWapper{})
	if err != nil {
		logrus.WithError(err).Fatal("Error starting goroutine pool.")
	}
	defer pool.Release()

	p := newRobot(c, pool)

	defer interrupts.WaitForGracefulShutdown()

	ctx, cancel := context.WithCancel(context.Background())

	interrupts.OnInterrupt(func() {
		cancel()
	})

	p.run(ctx, &o)
}

func newPool(size int, log ants.Logger) (*ants.Pool, error) {
	return ants.NewPool(size, ants.WithOptions(ants.Options{
		Logger: log,
	}))
}

type logWapper struct{}

func (l logWapper) Printf(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}
