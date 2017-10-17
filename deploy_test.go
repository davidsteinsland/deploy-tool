package main

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

type MockClient struct {
	t            *testing.T
	expectedJson string
}

func (m MockClient) Do(r *http.Request) (*http.Response, error) {

	if r.URL.String() != "http://url" {
		m.t.Error("URL should be <http://url>, is ", r.URL.String())
	}

	if r.Method != "POST" {
		m.t.Error("Request method should be POST, is ", r.Method)
	}

	defer r.Body.Close()
	body, _ := ioutil.ReadAll(r.Body)
	responseBody := string(body)

	if strings.TrimSpace(responseBody) != m.expectedJson {
		m.t.Errorf("Response body should be <%s>, is <%s>", m.expectedJson, strings.TrimSpace(responseBody))
	}

	if header := r.Header.Get("Authorization"); header != "token mytoken" {
		m.t.Error("Authorization should be <token mytoken>, is ", header)
	}

	if header := r.Header.Get("Content-Type"); header != "application/json" {
		m.t.Error("Content-Type should be <application/json>, is ", header)
	}

	return nil, nil
}

func TestDeploymentRequest_Send(t *testing.T) {

	t.Run("Ref=Empty", func(t *testing.T) {
		req := &DeploymentRequest{}
		resp, err := req.Send(MockClient{}, "http://url", "mytoken")

		if resp != nil {
			t.Error("Response is not nil")
		}

		if err == nil {
			t.Error("Err is nil")
		} else if err.Error() != "Ref is empty" {
			t.Error("Error should be <Ref is empty>, is ", err.Error())
		}
	})

	t.Run("Ref=NotEmpty", func(t *testing.T) {
		client := MockClient{t, "{\"ref\":\"myref\",\"auto_merge\":false}"}

		req := &DeploymentRequest{
			Ref: "myref",
		}
		resp, err := req.Send(client, "http://url", "mytoken")

		if err != nil {
			t.Error("Err is not nil")
		}

		if resp != nil {
			t.Error("Response is not nil")
		}
	})
	t.Run("AllValues", func(t *testing.T) {
		client := MockClient{t, "{\"ref\":\"myref\",\"task\":\"task\",\"auto_merge\":true,\"required_contexts\":[\"ci/travis\"],\"payload\":\"{\\\"foo\\\":\\\"bar\\\"}\",\"environment\":\"production\",\"description\":\"deploy to production\"}"}

		req := &DeploymentRequest{
			Ref:              "myref",
			Task:             "task",
			AutoMerge:        true,
			RequiredContexts: []string{"ci/travis"},
			Payload:          "{\"foo\":\"bar\"}",
			Environment:      "production",
			Description:      "deploy to production",
		}
		resp, err := req.Send(client, "http://url", "mytoken")

		if err != nil {
			t.Error("Err is not nil")
		}

		if resp != nil {
			t.Error("Response is not nil")
		}
	})
}

func TestHandleResponse(t *testing.T) {

	t.Run("Status=201", func(t *testing.T) {
		response := &http.Response{
			StatusCode: 201,
		}

		if HandleResponse(response, nil) != nil {
			t.Error("Err should be nil")
		}
	})
	t.Run("Status=202,JSON=Invalid", func(t *testing.T) {
		response := &http.Response{
			StatusCode: 202,
		}
		body := []byte("invalid json")

		err := HandleResponse(response, body)

		if err == nil {
			t.Error("Err should not be nil")
		} else {
			if err.ExitCode != EXIT_UNEXPECTED {
				t.Error("ExitCode should be ", EXIT_UNEXPECTED)
			}

			if err.Message != "invalid character 'i' looking for beginning of value" {
				t.Error("Message should be <invalid character 'i' looking for beginning of value>, was ", err.Message)
			}
		}
	})
	t.Run("Status=202,JSON=Valid", func(t *testing.T) {
		response := &http.Response{
			StatusCode: 202,
		}
		body := []byte("{\"message\": \"Auto-merged develop into master on deployment.\"}")

		err := HandleResponse(response, body)

		if err == nil {
			t.Error("Err should not be nil")
		} else {
			if err.ExitCode != EXIT_MERGE_COMMIT {
				t.Error("ExitCode should be ", EXIT_MERGE_COMMIT)
			}

			if err.Message != "Auto-merged develop into master on deployment." {
				t.Error("Message should be <Auto-merged develop into master on deployment.>, was ", err.Message)
			}
		}
	})
	t.Run("Status=401", func(t *testing.T) {
		response := &http.Response{
			StatusCode: 401,
		}
		err := HandleResponse(response, nil)

		if err == nil {
			t.Error("Err should not be nil")
		} else {
			if err.ExitCode != EXIT_UNAUTHORIZED {
				t.Error("ExitCode should be ", EXIT_UNAUTHORIZED)
			}

			if err.Message != "Unauthorized. Is the OAuth token provided correct?" {
				t.Error("Message should be <Unauthorized. Is the OAuth token provided correct?>, was ", err.Message)
			}
		}
	})
	t.Run("Status=404", func(t *testing.T) {
		response := &http.Response{
			StatusCode: 404,
		}
		err := HandleResponse(response, nil)

		if err == nil {
			t.Error("Err should not be nil")
		} else {
			if err.ExitCode != EXIT_NOT_FOUND {
				t.Error("ExitCode should be ", EXIT_NOT_FOUND)
			}

			if err.Message != "Resource not found. Did you type the correct owner/repo?" {
				t.Error("Message should be <Resource not found. Did you type the correct owner/repo?>, was ", err.Message)
			}
		}
	})
	t.Run("Status=419,JSON=Invalid", func(t *testing.T) {
		response := &http.Response{
			StatusCode: 419,
		}
		body := []byte("invalid json")

		err := HandleResponse(response, body)

		if err == nil {
			t.Error("Err should not be nil")
		} else {
			if err.ExitCode != EXIT_UNEXPECTED {
				t.Error("ExitCode should be ", EXIT_UNEXPECTED)
			}

			if err.Message != "invalid character 'i' looking for beginning of value" {
				t.Error("Message should be <invalid character 'i' looking for beginning of value>, was ", err.Message)
			}
		}
	})
	t.Run("Status=419,JSON=Valid", func(t *testing.T) {
		response := &http.Response{
			StatusCode: 419,
		}
		body := []byte("{\"message\": \"Conflict merging master into topic-branch\"}")

		err := HandleResponse(response, body)

		if err == nil {
			t.Error("Err should not be nil")
		} else {
			if err.ExitCode != EXIT_CONFLICT {
				t.Error("ExitCode should be ", EXIT_CONFLICT)
			}

			if err.Message != "Error: Conflict merging master into topic-branch" {
				t.Error("Message should be <Error: Conflict merging master into topic-branch>, was ", err.Message)
			}
		}
	})
	t.Run("Status=500", func(t *testing.T) {
		response := &http.Response{
			StatusCode: 500,
		}
		body := []byte("Something has happened")

		err := HandleResponse(response, body)

		if err == nil {
			t.Error("Err should not be nil")
		} else {
			if err.ExitCode != EXIT_UNEXPECTED {
				t.Error("ExitCode should be ", EXIT_UNEXPECTED)
			}

			if err.Message != "Unexpected error. Status code: 500\nBody:\nSomething has happened" {
				t.Error("Message should be <Unexpected error. Status code: 500\nBody:\nSomething has happened>, was ", err.Message)
			}
		}
	})
}

func TestCliArgs_Parse(t *testing.T) {
	t.Run("Ref=Empty", func(t *testing.T) {
		args := &CliArgs{}
		err := args.Parse()
		if err == nil {
			t.Error("Err is nil")
		} else if err.Error() != "Ref is required" {
			t.Errorf("Err should be <Ref is required>, is <%s>", err.Error())
		}
	})
	t.Run("Owner=Empty", func(t *testing.T) {
		args := &CliArgs{
			Ref: "myref",
		}
		err := args.Parse()
		if err == nil {
			t.Error("Err is nil")
		} else if err.Error() != "Owner and repo must be provided" {
			t.Errorf("Err should be <Owner and repo must be provided>, is <%s>", err.Error())
		}
	})
	t.Run("Repo=Empty", func(t *testing.T) {
		args := &CliArgs{
			Ref:   "myref",
			Owner: "myowner",
		}
		err := args.Parse()
		if err == nil {
			t.Error("Err is nil")
		} else if err.Error() != "Owner and repo must be provided" {
			t.Errorf("Err should be <Owner and repo must be provided>, is <%s>", err.Error())
		}
	})
	t.Run("Token=Empty", func(t *testing.T) {
		args := &CliArgs{
			Ref:   "myref",
			Owner: "myowner",
			Repo:  "myrepo",
		}
		err := args.Parse()

		if args.Url != "https://api.github.com/repos/myowner/myrepo/deployments" {
			t.Errorf("Url should be <https://api.github.com/repos/myowner/myrepo/deployments, is <%s>", args.Url)
		}

		if err == nil {
			t.Error("Err is nil")
		} else if err.Error() != "GitHub token is required" {
			t.Errorf("Err should be <GitHub token is required>, is <%s>", err.Error())
		}
	})
	t.Run("Payload=Invalid", func(t *testing.T) {
		args := &CliArgs{
			Ref:     "myref",
			Owner:   "myowner",
			Repo:    "myrepo",
			Token:   "mytoken",
			Payload: "invalid",
		}
		err := args.Parse()

		if err == nil {
			t.Error("Err is nil")
		} else if err.Error() != "Invalid JSON in Payload" {
			t.Errorf("Err should be <Invalid JSON in Payload>, is <%s>", err.Error())
		}
	})
	t.Run("Payload=Valid", func(t *testing.T) {
		args := &CliArgs{
			Ref:     "myref",
			Owner:   "myowner",
			Repo:    "myrepo",
			Token:   "mytoken",
			Payload: "{\"foo\":\"bar\"}",
		}
		err := args.Parse()

		if err != nil {
			t.Error("Err is not nil")
		}
	})
}
