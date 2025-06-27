// Gitea package handles functionalities related to the code repository (Gitea)
// which serves as the SDT Cloud's repository for storing apps. It collects
// information about apps, releases, and user details, and provides functionalities
// such as repository creation for app uploads, app downloads, etc.
package gitea

import (
	"bytes"
	"encoding/json"
	"fmt"
	sdtType "main/src/cliType"
	bhttp "net/http"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// These are the global variables used in the gitea package.
// - procLog: This is the struct that defines the format of the log.
var procLog sdtType.Logger

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// ReleaseGiteaRepo function creates a release in the code repository. When deploying apps
// via BWC-CLI, you can store the app in the code repository and create a release.
//
// Input:
//   - giteaURL: URL of the code repository.
//   - username: Username of the code repository.
//   - password: Password for the username.
//   - ownerName: Owner's username of the code repository.
//   - repoName: Name of the code repository.
//   - tagName: Release tag name in the code repository.
//   - releaseTitle: Title of the release.
//
// Output:
//   - error: Error message for the ReleaseGiteaRepo command.
func ReleaseGiteaRepo(
	giteaURL string,
	username string,
	password string,
	ownerName string,
	repoName string,
	tagName string,
	releaseTitle string) error {
	procLog.Info.Printf("Release app in code repository.\n")
	// Create a new release on Gitea using the API
	createReleaseURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/releases", giteaURL, ownerName, repoName)
	procLog.Info.Printf("Release path: %s\n", createReleaseURL)

	// JSON payload for creating a new release
	payload := map[string]interface{}{
		"tag_name":         tagName,
		"target_commitish": "main",
		"name":             releaseTitle,
		"body":             "Upload in device.",
		"draft":            false, // Set to true if you want to create a draft release
		"prerelease":       false, // Set to true if this is a pre-release
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		procLog.Error.Printf("Error marshaling JSON: %v\n", err)
		return err
	}

	req, err := bhttp.NewRequest("POST", createReleaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		procLog.Error.Printf("Error creating HTTP request: %v\n", err)
		return err
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")

	client := bhttp.Client{}
	resp, err := client.Do(req)
	if err != nil {
		procLog.Error.Printf("Error making HTTP request: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != bhttp.StatusCreated {
		procLog.Error.Printf("Failed to create release. Status code: %d\n", resp.StatusCode)
		return fmt.Errorf("Status code: %s\n", resp.Status)
	}

	procLog.Info.Printf("Successfully release app in code repository.\n")
	return nil
}

// CreateGiteaRepo function creates a code repository for the app in the code repository.
// When deploying apps via BWC-CLI, the app's code repository is created.
//
// Input:
//   - giteaURL: URL of the code repository.
//   - username: Username of the code repository.
//   - password: Password for the username.
//   - repoName: Name of the code repository.
func CreateGiteaRepo(
	giteaURL string,
	username string,
	password string,
	repoName string) {
	procLog.Info.Printf("Create app's repo in code repository.\n")

	// Step 1: Create a new repository on Gitea using the API
	createRepoURL := fmt.Sprintf("%s/api/v1/user/repos", giteaURL)

	// JSON payload for creating a new repository
	payload := map[string]interface{}{
		"name":        repoName,
		"description": "Your repository description",
		"private":     false, // Set to true for a private repository
		"auto_init":   true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		procLog.Error.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	req, err := bhttp.NewRequest("POST", createRepoURL, bytes.NewBuffer(jsonData))
	if err != nil {
		procLog.Error.Printf("Error creating HTTP request: %v\n", err)
		return
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")

	client := bhttp.Client{}
	resp, err := client.Do(req)
	if err != nil {
		procLog.Error.Printf("Error making HTTP request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == bhttp.StatusConflict {
		procLog.Warn.Printf("Repository is already exist!!!\n")
		fmt.Println("[Warning] Repository is already exist!!!")
	} else if resp.StatusCode != bhttp.StatusCreated {
		procLog.Error.Printf("Failed to create repository. Status code: %d\n", resp.StatusCode)
		return
	}
	procLog.Info.Printf("Successfully create app's repo in code repository.\n")
}

// CloneGiteaRepo function clones the code from a code repository.
//
// Input:
//   - giteaURL: URL of the code repository.
//   - username: Username of the code repository.
//   - password: Password for the username.
//   - repoName: Name of the code repository.
//   - localRepoPath: Path on the device where the repository will be cloned.
//   - ownerName: Owner's username of the code repository.
//
// Output:
//   - *git.Repository: Git repository variable.
func CloneGiteaRepo(
	giteaURL string,
	username string,
	password string,
	repoName string,
	localRepoPath string,
	owerName string,
) *git.Repository {
	procLog.Info.Printf("Clone app's repo in device.\n")
	var gitClient *git.Repository
	gitClient, openErr := git.PlainOpen(localRepoPath)
	if openErr != nil {
		fmt.Printf("Cloning repository %s/%s...\n", owerName, repoName)
		cloneRepo, cloneErr := git.PlainClone(localRepoPath, false, &git.CloneOptions{
			URL:  fmt.Sprintf("%s/%s/%s.git", giteaURL, owerName, repoName),
			Auth: &http.BasicAuth{Username: username, Password: password},
		})
		if cloneErr != nil {
			procLog.Error.Printf("Clone error: %v\n", cloneErr)
			// Handle the error as needed
		}
		gitClient = cloneRepo
	} else {
		procLog.Warn.Printf("Repository already exists. Skipping clone.\n")
		fmt.Println("Repository already exists. Skipping clone.")
	}

	procLog.Info.Printf("Successfully clone app's repo in device.\n")
	return gitClient
}

// CloneGiteaTemplate function clones an app template from the code repository to the device.
// App templates are basic app codes that perform basic examples such as MQTT, S3, Hello World, etc.
//
// Input:
//   - giteaURL: URL of the code repository.
//   - repoName: Name of the code repository.
//   - localRepoPath: Path on the device where the repository will be cloned.
//   - templateOwner: Username of the owner of the app template.
func CloneGiteaTemplate(
	giteaURL string,
	repoName string,
	localRepoPath string,
	templateOwner string,
) {
	// Set template's owner
	//templateOwner = "sujune"
	procLog.Info.Printf("Clone app template in device.\n")
	procLog.Info.Printf("Cloning repository %s/%s...\n", templateOwner, repoName)
	_, cloneErr := git.PlainClone(localRepoPath, false, &git.CloneOptions{
		URL: fmt.Sprintf("%s/%s/%s.git", giteaURL, templateOwner, repoName),
	})
	if cloneErr != nil {
		procLog.Error.Printf("Clone error: %v\n", cloneErr)
		fmt.Printf("Clone error:  %v\n", cloneErr)
		// Handle the error as needed
	}
	procLog.Info.Printf("Successfully clone app template in device.\n")
}

// PushGiteaRepo function creates a release in the code repository. When deploying an app through BWC-CLI,
// the app is stored in the code repository and a release is created.
//
// Input:
//   - gitClient: git.Repository variable.
//   - username: Username of the code repository.
//   - password: Password of the username.
//
// Output:
//   - error: Error message of the PushGiteaRepo command.
func PushGiteaRepo(gitClient *git.Repository, username string, password string) error {
	procLog.Info.Printf("Push app repo in code repository.\n")
	w, err := gitClient.Worktree()
	if err != nil {
		procLog.Error.Printf("Error worktree: %v\n", err)
	}

	status, err := w.Status()
	if err != nil {
		procLog.Error.Printf("Repo status: %v\n", err)
	}

	if status.IsClean() {
		procLog.Warn.Printf("Warnning: No changes to commit. \n")
		procLog.Warn.Printf("Status: %s\n", status)
	}

	_, err = w.Add(".")
	if err != nil {
		procLog.Error.Printf("Error add: %v\n", err)
	}

	// Example: Commit changes
	_, err = w.Commit("Commit message", &git.CommitOptions{
		Author: &object.Signature{
			Name:  strings.Split(username, "@")[0],
			Email: username,
			When:  time.Now(),
		},
	})
	if err != nil {
		procLog.Error.Printf("Error commit: push: %v\n", err)
	}

	// push
	err = gitClient.Push(&git.PushOptions{
		RemoteName: "origin",
		//RefSpecs:   []config.RefSpec{config.RefSpec("refs/heads/main:refs/heads/main")},
		Auth: &http.BasicAuth{Username: username, Password: password},
	})
	if err != nil {
		procLog.Error.Printf("Error: push: %v\n", err)
		return err
	}

	procLog.Info.Printf("Successfully clone app template in device.\n")
	return nil
}
