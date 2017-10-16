GitHub Deployment Tool
======================

CLI tool for GitHub Deployments API:
https://developer.github.com/v3/repos/deployments/#create-a-deployment

## Usage

```
GITHUB_TOKEN=hello ./deploy -ref=refspec \
    -owner=owner -repo=reponame
```

## Exit codes

- `0`: Deployment was succesful
- `1`: Argument parsing error
- `2`: Auto-merge commit
- `3`: Unauthorized, OAuth token failed
- `4`: Resource not found (404)
- `5`: Merge conflict or Failed commit status checks
- `6`: Unknown error