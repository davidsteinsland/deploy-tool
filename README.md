GitHub Deployment Tool
======================

CLI tool for GitHub Deployments API:
https://developer.github.com/v3/repos/deployments/#create-a-deployment

## Example

```
GITHUB_TOKEN=hello ./deploy -ref=refspec \
    -owner=owner -repo=reponame
```

## Usage

```
$ ./deploy-tool --help
Usage of ./deploy-tool:
  -description string
    	Description of the deploy
  -environment string
    	Environment of the deploy
  -merge
    	Merge the default branch into the requested ref if it's behind
  -owner string
    	GitHub repo owner
  -payload string
    	A custom JSON encoded payload
  -ref string
    	The ref to deploy
  -repo string
    	GitHub repo name
  -token string
    	GitHub OAuth token
```

* You can specify `GITHUB_DEPLOYMENTS_URL` (env var) instead of both `owner` and `repo`
* `GITHUB_TOKEN` (env var) may be used instead of `token`
* `Ref` will be set from `TRAVIS_COMMIT` or `GIT_COMMIT` if they are available
* `Owner` and `Repo` will be inferred from `TRAVIS_REPO_SLUG`

## Exit codes

- `0`: Deployment was succesful
- `1`: Argument parsing error
- `2`: Auto-merge commit
- `3`: Unauthorized, OAuth token failed
- `4`: Resource not found (404)
- `5`: Merge conflict or Failed commit status checks
- `10`: Unknown error