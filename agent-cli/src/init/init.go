// Init package handles downloading app templates to the device.
// App templates are templates provided by the code repository,
// including examples like MQTT, S3, and Hello World.
package init

import (
	"encoding/json"
	"fmt"
	sdtType "main/src/cliType"
	sdtGitea "main/src/gitea"
	sdtUtil "main/src/util"
	"os"

	"github.com/google/uuid"
)

// These are the global variables used in the init package.
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

// CreateFramework function creates a framework.yaml file that records metadata
// and deployment information of an app.
//
// Input:
//   - appName: The name of the app.
func CreateFramework(appName string) {
	procLog.Info.Printf("Create framework.yaml file.\n")
	filePath := fmt.Sprintf("./%s/framework.yaml", appName)

	// Create a new file or truncate an existing file
	file, err := os.Create(filePath)
	if err != nil {
		procLog.Error.Printf("error creating service file : %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	content := fmt.Sprintf(`version: bwc/v2 # bwc 버전 정보입니다.
spec: 
  appName: %s # 앱의 이름입니다.
  runFile: main.py # 앱의 실행 파일입니다.
  env:
    bin: <BIN_FILE(miniconda3 or python3)> # 앱을 실행할 바이너라 파일 종류입니다.(장비에 따라 다르므로 확인 후 정의해야 합니다.)
    virtualEnv: <VIRTUAL_NAME> # 사용할 가상환경 이름입니다.
    package: requirement.txt # 설치할 Python 패키지 정보 파일입니다.(기본 값은 requirement.txt 입니다.)
	runtime: <RUNTIE(python3.9.1)
stackbase:
  userName: <SDTCLOUD_ID> # Stackbase(gitea) ID 입니다.
  tagName: <TAG_NAME> # Stackbase(gitea)에 릴리즈 태그명 입니다.
  repoName: <REPO_NAME> # Stackbase(gitea)에 저장될 저장소 이릅니다.`, appName)

	_, err = file.WriteString(content)
	if err != nil {
		procLog.Error.Printf("Error writing to the file: %v\n", err)
		os.Exit(1)
	}
	procLog.Info.Printf("Successfully create framework.yaml file.\n")
}

// CreateConfig function creates a config.json file for the app.
//
// Input:
//   - appName: The name of the app.
func CreateConfig(appName string) {
	procLog.Info.Printf("Create config.json file.\n")
	// Create a new config.json file.
	content := map[string]interface{}{
		"KEY1": "VALUE1",
		"KEY2": "VALUE2",
	}
	filePath := fmt.Sprintf("./%s/config.json", appName)
	file, err := os.Create(filePath)
	if err != nil {
		procLog.Error.Printf("Creation failed: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	saveJson, _ := json.MarshalIndent(content, "", "\t")
	_, err = file.Write(saveJson)
	//encoder := json.NewEncoder(file)
	//err = encoder.Encode(content)
	if err != nil {
		procLog.Error.Printf("Marshal failed: %v\n", err)
		os.Exit(1)
	}

	procLog.Info.Printf("Successfully create config.json file.\n")
}

// CreateMainPy function creates the main.py python script for the app.
//
// Input:
//   - appName: The name of the app.
func CreateMainPy(appName string) {
	procLog.Info.Printf("Create main.py file.\n")
	filePath := fmt.Sprintf("./%s/main.py", appName)

	// Create a new file or truncate an existing file
	file, err := os.Create(filePath)
	if err != nil {
		procLog.Error.Printf("Error creating service file : %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	//create UUID
	newUUID := uuid.New()
	clientId := newUUID.String()

	content := fmt.Sprintf(`import sdtcloudpubsub
import time
import uuid

def runAction():
    sdtcloud = sdtcloudpubsub.sdtcloudpubsub()
    sdtcloud.setClient(f"device-app-{uuid.uuid1()}")
    msg = {
        "test": "good"
    }
    while True:  
        sdtcloud.pubMessage(msg)
        time.sleep(2)

if __name__ == "__main__":
    runAction()	`, clientId)

	_, err = file.WriteString(content)
	if err != nil {
		procLog.Error.Printf("Error writing to the file: %v\n", err)
		os.Exit(1)
	}
	procLog.Info.Printf("Successfully create main.py file.\n")
}

// CreateRequirement function creates the requirement.txt file listing
// Python package dependencies for the app.
//
// Input:
//   - appName: The name of the app.

func CreateRequirement(appName string) {
	procLog.Info.Printf("Create requirement.txt file.\n")
	filePath := fmt.Sprintf("./%s/requirement.txt", appName)

	// Create a new file or truncate an existing file
	file, err := os.Create(filePath)
	if err != nil {
		procLog.Error.Printf("Error creating service file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	content := "# Write package's name that need your app.\n"

	_, err = file.WriteString(content)
	if err != nil {
		procLog.Error.Printf("Error writing to the file: %v\n", err)
		os.Exit(1)
	}
	procLog.Info.Printf("Successfully create requirement.txt file.\n")
}

// CreateApp function downloads an app onto the device. If a templateName is specified,
// it downloads the app template from the code repository. If no templateName is provided,
// it generates necessary files for app execution.
//
// Input:
//   - appName: The name of the app.
//   - templateName: The name of the app template.
//   - giteaURL: The URL of the code repository.
//   - templateOwner: The username of the app template owner.
//   - homename: The hostname of the device.

func CreateApp(appName string, templateName string, giteaURL string, templateOwner string, homename string) {
	procLog.Info.Printf("Init app template in device.\n")
	if templateName == "" {
		//Get base App
		err := os.Mkdir(appName, 0755) // 0755는 디렉토리의 퍼미션을 나타냅니다.
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		CreateFramework(appName)
		CreateMainPy(appName)
		CreateConfig(appName)
		CreateRequirement(appName)
	} else {
		//Get App Template
		sdtGitea.CloneGiteaTemplate(giteaURL, templateName, appName, templateOwner)

		// Remove .git file
		err := os.RemoveAll(fmt.Sprintf("%s/.git", appName))
		if err != nil {
			procLog.Error.Printf("Can't remove .git file.(not found)\n")
		}
	}

	// Chown cmd
	sdtUtil.ChownCmd(appName, homename)
	procLog.Info.Printf("Successfully init app template in device.\n")
}
