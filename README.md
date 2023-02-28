# GitHub Backup

It can only be used to clone all repositories in an organization. With subsequent runs it will fetch instead of clone.

## Installation

Install [git](https://git-scm.com/) and [Go](https://go.dev/doc/install). Then run:

```
go install https://github.com/xtrafrancyz/gh-backup@latest
```

## Usage

Download all public repositories:

```bash
gh-backup --org EnterpriseQualityCoding --out ./my_org
```

Use an Access Token for private repositories:

```bash
GH_TOKEN="github_XXXXXX" gh-backup --org EnterpriseQualityCoding --out ./my_org
```
