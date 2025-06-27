// The Deploy package handles functionalities to deploy apps on the device.
// Apps are installed to "/usr/local/sdt/app" on the device. Apps are deployed
// using Systemd and Dockerd. Deployed apps can be monitored via the SDT Cloud
// console and device terminal.
package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"

	sdtType "main/src/cliType"
	sdtGitea "main/src/gitea"
	sdtUtil "main/src/util"
)

// These are the global variables used in the deploy package.
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

// UploadStackbase function uploads the app to the code repository. The upload path
// is defined by the value specified in the stackbase field of the framework.
// When uploading the app, a new repository is created and a new release is generated.
// Once the release creation is complete, the app can be created in the SDT Cloud console's app store.
//
// Input:
//   - bwcFramework: Struct containing information about the app's framework.
//   - targetDir: Path of the app to upload.
//   - giteaURL: URL of the code repository.
//   - configData: Struct containing configuration information for BWC.
//   - ownerName: User ID of the code repository.
func UploadStackbase(
	bwcFramework sdtType.Framework,
	targetDir string, giteaURL string,
	configData sdtType.ConfigInfo,
	ownerName string,
) {
	procLog.Info.Printf("Upload app in code repository.\n")
	var username, password, localRepoPath, releaseTitle string
	// Set parameter
	username = configData.SdtcloudId
	password = configData.SdtcloudPw

	localRepoPath = fmt.Sprintf("/etc/sdt/gitea-repo/%s", bwcFramework.Stackbase.RepoName)
	releaseTitle = bwcFramework.Stackbase.TagName

	procLog.Info.Printf("Code repository spec: repoName=%s, tag=%s, username=%s \n", bwcFramework.Stackbase.RepoName, bwcFramework.Stackbase.TagName, username)

	// Upload app in gitea
	sdtGitea.CreateGiteaRepo(giteaURL, username, password, bwcFramework.Stackbase.RepoName)
	gitClient := sdtGitea.CloneGiteaRepo(giteaURL, username, password, bwcFramework.Stackbase.RepoName, localRepoPath, ownerName)

	// Copy app dir in clone dir
	sdtUtil.CopyDir(targetDir, localRepoPath)
	uploadError := sdtGitea.PushGiteaRepo(gitClient, username, password)
	if uploadError != nil {
		fmt.Printf("Error upload: %v\n", uploadError)
		os.RemoveAll(localRepoPath)
		os.Exit(1)
	} else {
		fmt.Printf("Successfully upload app in code repository.\n")
	}
	time.Sleep(2 * time.Second)
	releaseError := sdtGitea.ReleaseGiteaRepo(giteaURL, username, password, ownerName, bwcFramework.Stackbase.RepoName, bwcFramework.Stackbase.TagName, releaseTitle)
	if releaseError != nil {
		fmt.Printf("Error release: %v\n", releaseError)
		fmt.Printf("There is already a release version of the app registered in the code repository.\n" +
			"If you want to upload a new release version, please modify the stackbase.tagName value in the framework.yaml file.\n" +
			"If you only want to deploy the app without uploading, please use the command below.\n")
		fmt.Printf("bwc deploy app -d . -u false\n")
		os.RemoveAll(localRepoPath)
		os.Exit(1)
	} else {
		fmt.Printf("Successfully released app in code repository.\n")
	}
	err := os.RemoveAll(localRepoPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	procLog.Info.Printf("Successfully upload app in code repository.\n")
}

// CreateGoService function creates a systemd service file (.service) for a Golang app.
//
// Input:
//   - appDir: Path of the app to be installed on the device.
//   - bwcFramework: Struct containing information about the app's framework.
//
// Output:
//   - error: Error message for the CreateGoService command.
func CreateGoService(appDir string, bwcFramework sdtType.Framework) error {
	procLog.Info.Printf("Create app[Golang] service file in device.\n")
	appName := bwcFramework.Spec.AppName
	runCmd := bwcFramework.Spec.RunFile
	// Specify the file name and path
	filePath := fmt.Sprintf("%s/%s.service", appDir, appName)

	// Create a new file or truncate an existing file
	file, err := os.Create(filePath)
	if err != nil {
		procLog.Error.Printf("error creating service file : %v\n", err)
		return err
	}
	defer file.Close()

	// Write content to the file
	content := fmt.Sprintf(`[Unit]
Description=%s

[Service]
WorkingDirectory=%s
ExecStart=%s/%s
Restart=always
RestartSec=10
StandardOutput=file:/%s/app.log
StandardError=file:/%s/app-error.log

[Install]
WantedBy=multi-user.target
	`, appName, appDir, appDir, runCmd, appDir, appDir)
	_, err = file.WriteString(content)
	if err != nil {
		procLog.Error.Printf("Error writing to the file: %v\n", err)
		return err
	}

	svcFile := fmt.Sprintf("/etc/systemd/system/%s.service", appName)
	err = sdtUtil.CopyFile(filePath, svcFile)

	if err != nil {
		procLog.Error.Printf("Error svc file copy to systemd: %v\n", err)
		return err
	}

	// Set chmod
	startCmd := fmt.Sprintf("chmod +x %s/%s", appDir, runCmd)
	cmd_run := exec.Command("sh", "-c", startCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Error.Printf("Failed app's service file: %v\n", cmd_err)
		procLog.Error.Printf("Failed app's service file - Result: %s\n", string(stdout))
		return cmd_err
	}
	procLog.Info.Printf("Successfully create app[Golang] service file in device.\n")
	return nil
}

// CreatePythonService function creates a systemd service file (.service) for a Python app.
//
// Input:
//   - appDir: Path of the app to be installed on the device.
//   - bwcFramework: Struct containing information about the app's framework.
//
// Output:
//   - error: Error message for the CreatePythonService command.
func CreatePythonService(appDir string, bwcFramework sdtType.Framework) error {
	procLog.Info.Printf("Create app[Python] service file in device.\n")
	appName := bwcFramework.Spec.AppName
	appVenv := bwcFramework.Spec.Env.VirtualEnv
	runCmd := bwcFramework.Spec.RunFile
	// Specify the file name and path
	filePath := fmt.Sprintf("%s/%s.service", appDir, appName)

	// Create a new file or truncate an existing file
	file, err := os.Create(filePath)
	if err != nil {
		procLog.Error.Printf("error creating service file: %v\n", err)
		return err
	}
	defer file.Close()

	// Write content to the file
	content := fmt.Sprintf(`[Unit]
Description=%s

[Service]
WorkingDirectory=%s
Environment=PATH=/etc/sdt/venv/%s/bin:$PATH
ExecStart=/etc/sdt/venv/%s/bin/python %s
Restart=always
RestartSec=10
StandardOutput=file:/%s/app.log
StandardError=file:/%s/app-error.log

[Install]
WantedBy=multi-user.target
	`, appName, appDir, appVenv, appVenv, runCmd, appDir, appDir)
	_, err = file.WriteString(content)
	if err != nil {
		procLog.Error.Printf("Error writing to the file: %v\n", err)
		return err
	}

	svcFile := fmt.Sprintf("/etc/systemd/system/%s.service", appName)
	err = sdtUtil.CopyFile(filePath, svcFile)

	if err != nil {
		procLog.Error.Printf("Error svc file copy to systemd: %v\n", err)
		return err
	}
	procLog.Info.Printf("Successfully create app[Python] service file in device.\n")
	return nil
}

// DeployApp function deploys an app on the device. When deploying the app, the app
// directory and systemd (.service) file are created.
//
// Input:
//   - appDir: Path of the app to be installed on the device.
//   - bwcFramework: Struct containing information about the app's framework.
//
// Output:
//   - error: Error message for the DeployApp command.
func DeployApp(bwcFramework sdtType.Framework, appDir string, minioURL string) error {
	// TODO Env.Bin 삭제
	appType := bwcFramework.Spec.AppType
	procLog.Info.Printf("[Deploy] Check appType.(=%s)\n", appType)
	if appType == "inference" {
		err := sdtUtil.CreateInferenceDir(appDir)
		if err != nil {
			procLog.Error.Printf("Failed creating inference directory.\n")
			return err
		}
		procLog.Info.Printf("[Deploy] App is inference. So, device download weight file.(=%s)\n", appType)
		err = DownloadWeight(bwcFramework, appDir, minioURL)
		if err != nil {
			procLog.Error.Printf("Failed download inference model.\n")
			return err
		}

	}

	procLog.Info.Printf("[Deploy] Check runtime.\n")
	if strings.Contains(bwcFramework.Spec.Env.RunTime, "python") {
		CreatePythonService(appDir, bwcFramework)
	} else if strings.Contains(bwcFramework.Spec.Env.RunTime, "go") {
		CreateGoService(appDir, bwcFramework)
	} else {
		procLog.Error.Printf("Failed creating %s's svc file.\n", bwcFramework.Spec.Env.RunTime)
		return errors.New(fmt.Sprintf("Not found runtime[%s]\n", bwcFramework.Spec.Env.RunTime))
	}
	procLog.Info.Printf("App's lang is %s.\n", bwcFramework.Spec.Env.RunTime)

	procLog.Info.Printf("[systemd] Start app servie.\n")
	startCmd := fmt.Sprintf("systemctl start %s", bwcFramework.Spec.AppName)
	cmd_run := exec.Command("sh", "-c", startCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Error.Printf("App deployment failed: %v\n", cmd_err)
		procLog.Error.Printf("App deployment failed - Result: %v\n", string(stdout))
		return cmd_err
	}
	procLog.Info.Printf("[systemd] Successfully start app servie.\n")
	return nil
}

// SaveAppInfo function saves metadata of the deployed app on the device.
// BWC manages the metadata of the deployed app as a JSON file on the device.
//
// Input:
//   - appName: Name of the app.
//   - appId: ID of the app.
//   - appVenv: Virtual environment used by the app.
//
// Output:
//   - error: Error message for the SaveAppInfo command.
func SaveAppInfo(appName string, appId string, appVenv string, appManaged string) error {
	procLog.Info.Printf("Record app's info in app.json\n")
	appInfoFile := "/etc/sdt/device.config/app.json"
	// check app's info file
	if _, err := os.Stat(appInfoFile); os.IsNotExist(err) {
		appInfo := []sdtType.AppInfo{
			{
				AppName: appName,
				AppId:   appId,
				AppVenv: appVenv,
				Managed: appManaged,
			},
		}
		jsonData := sdtType.AppConfig{
			AppInfoList: appInfo,
		}
		saveJson, err := json.MarshalIndent(jsonData, "", "\t")
		if err != nil {
			procLog.Error.Printf("Failed save app's Marshal: %v\n", err)
			return err
		}

		err = ioutil.WriteFile(appInfoFile, saveJson, 0644)
		if err != nil {
			procLog.Error.Printf("Failed save app's info: %v\n", err)
			return err
		}
	} else {
		jsonFile, err := ioutil.ReadFile(appInfoFile)
		if err != nil {
			procLog.Error.Printf("Failed load app's file: %v\n", err)
			return err
		}
		var jsonData sdtType.AppConfig
		err = json.Unmarshal(jsonFile, &jsonData)
		if err != nil {
			procLog.Error.Printf("Failed save app's Unmarshal: %v\n", err)
			return err
		}

		newApp := sdtType.AppInfo{
			AppName: appName,
			AppId:   appId,
			AppVenv: appVenv,
			Managed: appManaged,
		}

		jsonData.AppInfoList = append(jsonData.AppInfoList, newApp)
		saveJson, _ := json.MarshalIndent(&jsonData, "", "\t")
		err = ioutil.WriteFile(appInfoFile, saveJson, 0644)
		if err != nil {
			procLog.Error.Printf("Failed save app's file: %v\n", err)
			return err
		}
	}
	procLog.Info.Printf("Record app's info completed in app.json\n")
	return nil
}

// GetConfig function collects the config information of the deployed app on the device.
//
// Input:
//   - appId: ID of the app.
//   - appName: Name of the app.
//   - archType: Architecture of the device.
//
// Output:
//   - map[string]interface{}: Config values of the app.
func GetAppConfig(appId string, appName string, archType string) map[string]interface{} {
	procLog.Info.Printf("Get app's config.\n")
	var targetDir string
	if archType == "win" {
		targetDir = fmt.Sprintf("C:/sdt/app/%s_%s", appName, appId)
	} else {
		targetDir = fmt.Sprintf("/usr/local/sdt/app/%s_%s", appName, appId)
	}
	procLog.Info.Printf("Read config file: %s\n", targetDir)
	fileList, err := ioutil.ReadDir(targetDir)
	if err != nil {
		procLog.Error.Printf("App not found: %v\n", err)
		return nil
	}

	procLog.Info.Printf("Convert files to json.\n")
	var allConfig = make(map[string]interface{})
	for _, file := range fileList {
		fileName := fmt.Sprintf("%s", file.Name())
		fileType := strings.Split(fileName, ".")

		if fileType[len(fileType)-1] == "json" {
			result := GetJson(fileName, targetDir)
			allConfig[fileName] = result
		}

	}
	// Not found config file.
	if len(allConfig) == 0 {
		procLog.Warn.Printf("App's config not found.\n")
		return allConfig
	}

	procLog.Info.Printf("Successfully get app's config.\n")
	return allConfig
}

// GetJson function reads a JSON file.
//
// Input:
//   - appId: ID of the app.
//   - appName: Name of the app.
//   - fileName: Name of the JSON file to read.
//   - targetDir: Directory where the JSON file is located. (Equals the app's location.)
//
// Output:
//   - map[string]interface{}: Config values of the app.
func GetJson(fileName string, targetDir string) map[string]interface{} {
	procLog.Info.Printf("Read app's config. Unmarshal config file.\n")
	targetFile := fmt.Sprintf("%s/%s", targetDir, fileName)
	jsonFile, err := ioutil.ReadFile(targetFile)
	if err != nil {
		procLog.Error.Printf("File not found: %v\n", err)
		return nil
	}
	var jsonData map[string]interface{}
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("Unmarshal Error: %v\n", err)
		return nil
	}
	procLog.Info.Printf("Successfully read and unmarshal app's config.\n")
	return jsonData
}

// DownloadWeight function download model weight file from storage. Weight file used in inference app.
// Weight file store objectstorage. So, this function need accesskey and secretkey about objectstorage.
//
// Input:
//   - bwcFramework: The name of the virtual environment.
//   - appDir: The IP address of SDT Cloud.
//   - minioURL: The port number of the code repository.
//
// Output:
//   - error: An error message if weight file download fail.
func DownloadWeight(bwcFramework sdtType.Framework, appDir string, minioURL string) error {
	endpoint := minioURL
	minioKey := "shWhLpEhJA8mlMLcldCT"
	minioSecret := "QONntgD3bww2CGVKKDz5Qtg3CWzP1FMqWyatBU5P"
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioKey, minioSecret, ""),
		Secure: useSSL,
	})
	if err != nil {
		procLog.Error.Printf("Failed access minio: %v\n", err)
		return err
	}

	// Set minio bucket and file.
	bucketName := bwcFramework.Inference.Bucket
	objectName := fmt.Sprintf("%s/%s", bwcFramework.Inference.Path, bwcFramework.Inference.WeightFile)
	filePath := fmt.Sprintf("%s/weights/%s", appDir, bwcFramework.Inference.WeightFile)

	// Download model file.
	fmt.Printf("Download model file from object storage.\n")
	err = minioClient.FGetObject(context.Background(), bucketName, objectName, filePath, minio.GetObjectOptions{})
	if err != nil {
		fmt.Printf("Failed download model file from object storage. %v\n", err)
		procLog.Error.Printf("Failed download model: %v\n", err)
		return err
	}

	fmt.Printf("Successfully download model file.\n")
	procLog.Info.Printf("Successfully downloaded %s to %s\n", objectName, filePath)
	return nil
}

func WolTest(macAddress string) {
	startCmd := fmt.Sprintf("wakeonlan -p 8 %s", macAddress)
	fmt.Printf("[WOL] Info: %s\n", startCmd)
	cmd_run := exec.Command("sh", "-c", startCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		fmt.Println("[WOL] ERROR: ", cmd_err, "\n", string(stdout))
		os.Exit(1)
	}
}
