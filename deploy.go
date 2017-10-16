package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type DeploymentRequest struct {
	Ref              string   `json:"ref"`                         // Required. The ref to deploy. This can be a branch, tag, or SHA.
	Task             string   `json:"task,omitempty"`              // Specifies a task to execute (e.g., deploy or deploy:migrations). Default: deploy
	AutoMerge        bool     `json:"auto_merge"`                  // Attempts to automatically merge the default branch into the requested ref, if it is behind the default branch. Default: true
	RequiredContexts []string `json:"required_contexts,omitempty"` // The status contexts to verify against commit status checks. If this parameter is omitted, then all unique contexts will be verified before a deployment is created. To bypass checking entirely pass an empty array. Defaults to all unique contexts.
	Payload          string   `json:"payload,omitempty"`           // JSON payload with extra information about the deployment. Default: ""
	Environment      string   `json:"environment,omitempty"`       // Name for the target deployment environment (e.g., production, staging, qa). Default: production
	Description      string   `json:"description,omitempty"`       // Short description of the deployment. Default: ""
}

type DeploymentResponse struct {
	Id          int
	Url         string
	Sha         string
	Ref         string
	Task        string
	Payload     string
	Environment string
	Description string
	CreatedAt   string
	UpdatedAt   string
	StatusesUrl string
	Creator
}

type Creator struct {
	Id        int
	Login     string
	SiteAdmin bool
}

type ErrorResponse struct {
	Message string
}

func main() {
	var ref, owner, repo, ghToken, payload, description, environment string

	flag.StringVar(&ref, "ref", "", "The ref to deploy")
	flag.StringVar(&owner, "owner", "", "GitHub repo owner")
	flag.StringVar(&repo, "repo", "", "GitHub repo name")
	flag.StringVar(&ghToken, "token", os.Getenv("GITHUB_TOKEN"), "GitHub OAuth token")
	flag.StringVar(&payload, "payload", "", "A custom JSON encoded payload")
	flag.StringVar(&description, "description", "", "Description of the deploy")
	flag.StringVar(&environment, "environment", "", "Environment of the deploy")

	var merge bool
	flag.BoolVar(&merge, "merge", false, "Merge the default branch into the requested ref if it's behind")

	flag.Parse()

	if len(ref) == 0 {
		fmt.Println("Ref is required")
		os.Exit(1)
	}

	deploymentUrl := os.Getenv("GITHUB_DEPLOYMENTS_URL")
	if len(deploymentUrl) == 0 {
		if len(owner) == 0 || len(repo) == 0 {
			fmt.Println("Owner and repo must be provided")
			os.Exit(1)
		}
		deploymentUrl = "https://api.github.com/repos/" + owner + "/" + repo + "/deployments"
	}

	if len(ghToken) == 0 {
		fmt.Println("GitHub token is required")
		os.Exit(1)
	}

	deploymentRequest := DeploymentRequest{
		Ref:         ref,
		Payload:     payload,
		Description: description,
		Environment: environment,
		AutoMerge:   merge,
	}

	body := new(bytes.Buffer)
	json.NewEncoder(body).Encode(deploymentRequest)

	fmt.Println(string(body.Bytes()))
	os.Exit(0)

	req, err := http.NewRequest("POST", deploymentUrl, body)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", "token "+ghToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	responseBody, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 201 {
		var deploymentResponse DeploymentResponse
		if err := json.Unmarshal(responseBody, &deploymentResponse); err != nil {
			panic(err)
		}

		fmt.Println("Deployment successful.")
		os.Exit(0)
	}

	if resp.StatusCode == 202 {
		var deploymentError ErrorResponse
		if err := json.Unmarshal(responseBody, &deploymentError); err != nil {
			panic(err)
		}
		// we will exit with a special exit code so caller can either try again or not
		// (The auto-merge commit may spawn a new CI build, which in turn will spawn
		// a new deployment request)
		fmt.Println(deploymentError.Message)
		os.Exit(2)
	}

	if resp.StatusCode == 401 {
		fmt.Println("Unauthorized. Is the OAuth token provided correct?")
		os.Exit(3)
	}

	if resp.StatusCode == 404 {
		fmt.Println("Resource not found. Did you type the correct owner/repo?")
		os.Exit(4)
	}

	if resp.StatusCode == 419 {
		var deploymentError ErrorResponse
		if err := json.Unmarshal(responseBody, &deploymentError); err != nil {
			panic(err)
		}

		fmt.Println("Error: ", deploymentError.Message)
		os.Exit(5)
	}

	fmt.Println("Unexpected status code: ", resp.StatusCode)
	fmt.Println("Body:")
	fmt.Println(string(responseBody))
	os.Exit(10)
}
