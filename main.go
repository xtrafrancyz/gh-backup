package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

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

	url := fmt.Sprintf("https://github.com/%s/%s.git", org, name)
	authUrl := url
	if token != "" {
		authUrl = fmt.Sprintf("https://123asd:%s@github.com/%s/%s.git", token, org, name)
	}

	if _, err := os.Stat(filepath.Join(repoPath, "config")); os.IsNotExist(err) {
		log.Printf("Cloning repo %s/%s", org, name)

		if err = os.MkdirAll(repoPath, 0644); err != nil {
			return err
		}

		cmd := exec.Command("git", "clone", "--quiet", "--mirror", authUrl, ".")
		cmd.Dir = repoPath
		if err = cmd.Run(); err != nil {
			return err
		}

		if authUrl != url {
			cmd = exec.Command("git", "remote", "set-url", "origin", url)
			cmd.Dir = repoPath
			if err = cmd.Run(); err != nil {
				return err
			}
		}
	} else {
		log.Printf("Updating repo %s/%s", org, name)

		// set url with credentials
		if authUrl != url {
			cmd := exec.Command("git", "remote", "set-url", "origin", authUrl)
			cmd.Dir = repoPath
			if err = cmd.Run(); err != nil {
				return err
			}
		}

		// updaet repository
		cmd := exec.Command("git", "remote", "update", "--prune")
		cmd.Dir = repoPath
		if err = cmd.Run(); err != nil {
			return err
		}

		// remove credentials from url
		if authUrl != url {
			cmd = exec.Command("git", "remote", "set-url", "origin", url)
			cmd.Dir = repoPath
			if err = cmd.Run(); err != nil {
				return err
			}
		}

		// Run gc
		cmd = exec.Command("git",
			"-c", "gc.auto=1000",
			"-c", "gc.autoPackLimit=10",
			"-c", "gc.autoDetach=false",
			"gc", "--auto", "--quiet")
		cmd.Dir = repoPath
		if err = cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
