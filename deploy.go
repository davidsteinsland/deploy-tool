package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

const (
	EXIT_SUCCESS      = 0
	EXIT_INVALID_ARGS = 1
	EXIT_MERGE_COMMIT = 2
	EXIT_UNAUTHORIZED = 3
	EXIT_NOT_FOUND    = 4
	EXIT_CONFLICT     = 5
	EXIT_UNEXPECTED   = 10
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
	Payload     interface{}
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

type DeploymentErrorResponse struct {
	Message string
}

type httpClient interface {
	Do(r *http.Request) (*http.Response, error)
}

func (dep DeploymentRequest) Send(client httpClient, url string, token string) (*http.Response, error) {
	if len(dep.Ref) == 0 {
		return nil, fmt.Errorf("Ref is empty")
	}

	body := new(bytes.Buffer)
	json.NewEncoder(body).Encode(dep)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	return client.Do(req)
}

type deploymentError struct {
	ExitCode int
	Message  string
}

func (e *deploymentError) Error() string {
	return e.Message
}

func NewDeploymentError(exitCode int, message string) *deploymentError {
	return &deploymentError{
		ExitCode: exitCode,
		Message:  message,
	}
}

func HandleResponse(response *http.Response, body []byte) *deploymentError {
	if response.StatusCode == 201 {
		return nil
	}

	if response.StatusCode == 202 {
		var deploymentError DeploymentErrorResponse
		if err := json.Unmarshal(body, &deploymentError); err != nil {
			return NewDeploymentError(EXIT_UNEXPECTED, err.Error())
		}
		// we will exit with a special exit code so caller can either try again or not
		// (The auto-merge commit may spawn a new CI build, which in turn will spawn
		// a new deployment request)
		return NewDeploymentError(EXIT_MERGE_COMMIT, deploymentError.Message)
	}

	if response.StatusCode == 401 {
		return NewDeploymentError(EXIT_UNAUTHORIZED, "Unauthorized. Is the OAuth token provided correct?")
	}

	if response.StatusCode == 404 {
		return NewDeploymentError(EXIT_NOT_FOUND, "Resource not found. Did you type the correct owner/repo?")
	}

	if response.StatusCode == 419 {
		var deploymentError DeploymentErrorResponse
		if err := json.Unmarshal(body, &deploymentError); err != nil {
			return NewDeploymentError(EXIT_UNEXPECTED, err.Error())
		}

		return NewDeploymentError(EXIT_CONFLICT, "Error: "+deploymentError.Message)
	}

	return NewDeploymentError(EXIT_UNEXPECTED, "Unexpected error. Status code: "+strconv.Itoa(response.StatusCode)+"\nBody:\n"+string(body))
}

type CliArgs struct {
	Token       string
	Url         string
	Ref         string
	Owner       string
	Repo        string
	Payload     string
	Description string
	Environment string
	Merge       bool
}

func NewCliArgs() *CliArgs {
	args := &CliArgs{}

	flag.StringVar(&args.Ref, "ref", "", "The ref to deploy")
	flag.StringVar(&args.Owner, "owner", "", "GitHub repo owner")
	flag.StringVar(&args.Repo, "repo", "", "GitHub repo name")
	flag.StringVar(&args.Token, "token", os.Getenv("GITHUB_TOKEN"), "GitHub OAuth token")
	flag.StringVar(&args.Payload, "payload", "", "A custom JSON encoded payload")
	flag.StringVar(&args.Description, "description", "", "Description of the deploy")
	flag.StringVar(&args.Environment, "environment", "", "Environment of the deploy")
	flag.BoolVar(&args.Merge, "merge", false, "Merge the default branch into the requested ref if it's behind")

	return args
}

func (a *CliArgs) Parse() error {
	flag.Parse()

	if len(a.Ref) == 0 {
		return fmt.Errorf("Ref is required")
	}

	a.Url = os.Getenv("GITHUB_DEPLOYMENTS_URL")
	if len(a.Url) == 0 {
		if len(a.Owner) == 0 || len(a.Repo) == 0 {
			return fmt.Errorf("Owner and repo must be provided")
		}
		a.Url = "https://api.github.com/repos/" + a.Owner + "/" + a.Repo + "/deployments"
	}

	if len(a.Token) == 0 {
		return fmt.Errorf("GitHub token is required")
	}

	var payloadJson map[string]interface{}
	if len(a.Payload) > 0 && json.Unmarshal([]byte(a.Payload), &payloadJson) != nil {
		return fmt.Errorf("Invalid JSON in Payload")
	}

	return nil
}

func main() {
	args := NewCliArgs()

	if err := args.Parse(); err != nil {
		fmt.Println(err)
		os.Exit(EXIT_INVALID_ARGS)
	}

	deploymentRequest := DeploymentRequest{
		Ref:         args.Ref,
		Payload:     args.Payload,
		Description: args.Description,
		Environment: args.Environment,
		AutoMerge:   args.Merge,
	}

	client := http.DefaultClient
	resp, err := deploymentRequest.Send(client, args.Url, args.Token)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	responseBody, _ := ioutil.ReadAll(resp.Body)

	if err := HandleResponse(resp, responseBody); err != nil {
		fmt.Println(err.Error())
		os.Exit(err.ExitCode)
	}

	var deploymentResponse DeploymentResponse
	if err := json.Unmarshal(responseBody, &deploymentResponse); err != nil {
		fmt.Println("Cannot unmarshal response:")
		fmt.Println(string(responseBody))

		panic(err)
	}

	fmt.Println("Deployment successful.")
	os.Exit(EXIT_SUCCESS)
}
