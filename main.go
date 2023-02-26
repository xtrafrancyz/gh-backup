package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v50/github"
	"github.com/spf13/pflag"
)

var (
	token = os.Getenv("GH_TOKEN")
	org   string
	out   string
)

func init() {
	pflag.StringVar(&org, "org", "", "Name of GitHub organization")
	pflag.StringVarP(&out, "out", "o", "data", "Output directory")
}

func main() {
	pflag.Parse()
	if org == "" {
		log.Fatalln("Error: flag -org is not set")
	}

	var client *github.Client
	if token == "" {
		client = github.NewClient(nil)
	} else {
		client = github.NewTokenClient(context.Background(), token)
	}

	repos, err := listRepos(client)
	if err != nil {
		log.Fatalln("Error: GitHub list repositories:", err)
	}
	for _, repo := range repos {
		err = cloneRepo(*repo.Name)
		if err != nil {
			log.Printf("Error: Unable to clone repo %s: %v", *repo.FullName, err)
		}
	}

	dirList, err := os.ReadDir(out)
	if err != nil {
		log.Fatalln("Error: listing out directory:", err)
	}
dirListLoop:
	for _, entry := range dirList {
		if !entry.IsDir() {
			continue
		}
		for _, repo := range repos {
			if *repo.Name == entry.Name() {
				continue dirListLoop
			}
		}
		log.Printf("Removing dir %s", entry.Name())
		_ = os.RemoveAll(filepath.Join(out, entry.Name()))
	}
}

func listRepos(client *github.Client) ([]*github.Repository, error) {
	const pageSize = 100
	var list []*github.Repository
	for page := 1; ; page++ {
		repos, _, err := client.Repositories.ListByOrg(context.Background(), org, &github.RepositoryListByOrgOptions{
			Sort: "full_name",
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: pageSize,
			},
		})
		if err != nil {
			return nil, err
		}
		list = append(list, repos...)
		if len(repos) < pageSize {
			break
		}
	}
	return list, nil
}

func cloneRepo(name string) error {
	repoPath := filepath.Join(out, name)

	var auth transport.AuthMethod
	if token != "" {
		auth = &http.BasicAuth{
			Username: "abc123", // yes, this can be anything except an empty string
			Password: token,
		}
	}
	url := fmt.Sprintf("https://github.com/%s/%s", org, name)

	var err error
	var repo *git.Repository
	if _, err = os.Stat(filepath.Join(repoPath, "config")); os.IsNotExist(err) {
		log.Printf("Cloning repo %s/%s", org, name)
		repo, err = git.PlainClone(repoPath, true, &git.CloneOptions{
			URL:        url,
			Auth:       auth,
			RemoteName: "origin",
		})
		if err != nil {
			return err
		}
	} else {
		log.Printf("Fetching repo %s/%s", org, name)
		repo, err = git.PlainOpen(repoPath)
	}
	if err != nil {
		return err
	}

	err = repo.Fetch(&git.FetchOptions{
		RemoteURL: url,
		Auth:      auth,
		Force:     true,
		// https://github.com/go-git/go-git/issues/293#issuecomment-1065877209
		RefSpecs: []config.RefSpec{"+refs/*:refs/*"},
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}
