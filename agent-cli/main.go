// The BWC-CLI main package allows you to use SDT Cloud functionality in a terminal environment.
// BWC-CLI enables management of devices from the terminal. Some features of BWC-CLI send result
// messages to SDT Cloud. The functionalities supported by BWC-CLI are as follows:
//   - Checking device status, processes, and BWC agent status
//   - Deploying, deleting, running, stopping apps on devices, uploading apps (code repository), downloading apps
//   - Logging into SDT Cloud, checking SDT Cloud login information
//
// BWC-CLI operates on both windows and linux systems.
package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"syscall"

	sdtCli "main/src/cli"
	sdtType "main/src/cliType"
	sdtCreate "main/src/create"
	sdtDelete "main/src/delete"
	sdtDeploy "main/src/deploy"
	sdtGet "main/src/get"
	sdtGitea "main/src/gitea"
	sdtHelp "main/src/help"
	sdtInit "main/src/init"
	sdtLogin "main/src/login"
	sdtLogs "main/src/logs"
	sdtMessage "main/src/message"
	sdtUpdate "main/src/update"
	sdtUtil "main/src/util"
)

// These are the global variables used in the BWC-CLI package.
// - procLog: This is the Struct that defines the format of the Log.
var (
	procLog sdtType.Logger
)

// initError defines and initializes the log format. The log formats are defined as Info,
// Warn, and Error, and the output is done using Printf. If you call the function to define and
// initialize the log formats, you can use it to output logs as follows:
// The way to log the text "Hello World" as an Info log type is shown below.
// procLog.Info.Printf("Hello World\n")
// Output: [INFO] Hello World
func initError(logFile io.Writer) {
	procLog.Info = log.New(logFile, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	procLog.Warn = log.New(logFile, "[WARNING] ", log.Ldate|log.Ltime|log.Lshortfile)
	procLog.Error = log.New(logFile, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
}

func isExistFile(fname string) bool {
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		return false
	}
	return true
}

// GetConfigJson is a function that reads the BWC config file of the device.
//
// Output:
//   - ConfigInfo: This is the config struct of BWC.
func GetConfigJson() sdtType.ConfigInfo {
	targetFile := "/etc/sdt/device.config/config.json"
	jsonFile, err := ioutil.ReadFile(targetFile)
	if err != nil {
		fmt.Printf("Not found file Error: %v\n", err)
	}
	var jsonData sdtType.ConfigInfo
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		fmt.Printf("Unmarshal Error: %v\n", err)
	}

	return jsonData
}

// GetFrameworks is a function that reads the framework file of the app.
//
// Input:
//   - dirName: This is the path of the app.
//
// Output:
//   - Framework: This is the framework struct of the app.
func GetFrameworks(dirName string) sdtType.Framework {
	var bwcFramework sdtType.Framework
	targetFile := fmt.Sprintf("%s/framework.json", dirName)
	// Get  framework
	// Frist. JSON
	if isExistFile(targetFile) {
		//procLog.Info.Printf("Get bwcFramework from JSON file.\n")
		jsonFile, err := ioutil.ReadFile(targetFile)
		if err != nil {
			fmt.Printf("Not found file Error: %v\n", err)
		}
		err = json.Unmarshal(jsonFile, &bwcFramework)
		if err != nil {
			fmt.Printf("Unmarshal Error: %v\n", err)
		}

		return bwcFramework
	}

	// Second. YAML
	targetFile = fmt.Sprintf("%s/framework.yaml", dirName)
	if isExistFile(targetFile) {
		//procLog.Info.Printf("Get bwcFramework from YAML file.\n")
		yamlFile, err := ioutil.ReadFile(targetFile)
		if err != nil {
			fmt.Printf("Not found file Error: %v\n", err)
		}
		err = yaml.Unmarshal(yamlFile, &bwcFramework)
		if err != nil {
			fmt.Printf("Unmarshal Error: %v\n", err)
		}

		return bwcFramework
	}
	return bwcFramework
}

// GetFrameworksYaml is a function that reads the framework.yaml file of the app.
//
// Input:
//   - dirName: This is the path of the app.
//
// Output:
//   - Framework: This is the framework struct of the app.
//func GetFrameworksYaml(dirName string) sdtType.Framework {
//	targetFile := fmt.Sprintf("%s/framework.yaml", dirName)
//	yamlFile, err := ioutil.ReadFile(targetFile)
//	if err != nil {
//		fmt.Printf("Not found file Error: %v\n", err)
//	}
//	var yamlData sdtType.Framework
//	err = yaml.Unmarshal(yamlFile, &yamlData)
//	if err != nil {
//		fmt.Printf("Unmarshal Error: %v\n", err)
//	}
//
//	return yamlData
//}

// GetUser is a function that retrieves the hostname of the device. The hostname
// can be obtained on a Linux system using the "whoami" terminal command.
//
// Output:
//   - String: This is the hostname.
func GetUser() string {
	username := os.Getenv("SUDO_USER")
	if username == "" {
		// Get current user's env.
		username = os.Getenv("USER")
		if username == "" {
			fmt.Println("Error: Unable to determine the username.")
			os.Exit(1)
		}
	}
	return username
}

// This is the main function that parses the input command and dispatches it
// to each corresponding feature. BWC-CLI handles the following features:
// help, login, create, deploy, delete, update, get, status, init, logs, info.
// These features have subtypes, and the processing varies depending on the combination of types.
//
// Input:
//   - '-d', '--directory': This is the directory path to process.
//   - '-n', '--name': This is the name of the object to install.
//   - '-u', '--upload': This is the option for uploading to the code repository.
//   - '-f', '--follow': This is the option to choose whether to continue printing logs when checking logs. (similar to the 'tail' function.)
//   - '-l', '--line': This is the option to define how many lines to output when checking logs.
//   - '-t', '--template': This is the name of the app template to download.
//   - '-o', '--option': This is the option used to manage the app's state (e.g., restart) when managing the app's status.
func main() {
	// TODO
	// 	- 실패 했을때 롤백 기능

	// Check root
	euid := syscall.Geteuid()
	if euid != 0 {
		fmt.Printf("Please use 'sudo'. If you want to get app, you have to use fallow as:\n Command: sudo bwc get app\n")
		os.Exit(1)
	}

	// Set parameter
	var cliInfo sdtType.CliCmd
	var bwcFramework sdtType.Framework
	var configData sdtType.ConfigInfo
	var archType, rootPath, appPath, cmd string
	cmdArgs := os.Args
	configData = GetConfigJson()
	archType = "linux"
	cliInfo.UploadOption = true

	// Set Command parameter
	cliInfo.FirstCmd = cmdArgs[1]
	if len(cmdArgs) >= 3 {
		cliInfo.TargetCmd = cmdArgs[2]
		for key, val := range cmdArgs {
			if val == "-d" || val == "--directory" {
				cliInfo.DirOption = cmdArgs[key+1]
			} else if val == "-n" || val == "--name" {
				cliInfo.NameOption = cmdArgs[key+1]
			} else if val == "-u" || val == "--upload" {
				if cmdArgs[key+1] == "false" {
					cliInfo.UploadOption = false
				} else {
					cliInfo.UploadOption = true
				}
			} else if val == "-f" || val == "--follow" {
				cliInfo.TailOption = true
			} else if val == "-l" || val == "--line" {
				cliInfo.LineOption, _ = strconv.Atoi(cmdArgs[key+1])
			} else if val == "-t" || val == "--template" {
				cliInfo.TemplateOption = cmdArgs[key+1]
			} else if val == "-o" || val == "--option" {
				cliInfo.AppOption = cmdArgs[key+1]
			}
		}
	}

	// Set URL
	// Check parameter
	switch cliInfo.FirstCmd {
	case "help":
		sdtHelp.PrintHelp()
		os.Exit(1)

	case "login":
		cmd = cliInfo.FirstCmd
	case "create":
		if cliInfo.DirOption == "" {
			fmt.Printf("Please input directory option(-d). \n")
			os.Exit(1)
		} else {
			bwcFramework = GetFrameworks(cliInfo.DirOption)
		}

		if cliInfo.TargetCmd == "venv" {
			fmt.Printf("Create virtual environment in your device. \n")
			if bwcFramework.Spec.Env.VirtualEnv == "" {
				fmt.Printf("Check spec's virtualEnvironment value in framework.yaml. \n")
				os.Exit(1)
			} else if bwcFramework.Spec.Env.Package == "" {
				fmt.Printf("Check spec's package value in framework.yaml. \n")
				os.Exit(1)
			}

			if bwcFramework.Spec.Env.RunTime == "" {
				bwcFramework.Spec.Env.Bin = "python3"
			}
		} else if cliInfo.TargetCmd == "app" {
			fmt.Printf("Excute app in your device. \n")
			if cliInfo.NameOption != "" {
				bwcFramework.Spec.AppName = cliInfo.NameOption
			}
			// 빈 값 찾기
			if bwcFramework.Spec.AppName == "" {
				fmt.Printf("Check spec's appName value in framework.yaml. \n")
				os.Exit(1)
			} else if bwcFramework.Spec.Env.VirtualEnv == "" {
				fmt.Printf("Check spec's virtualEnvironment value in framework.yaml. \n")
				os.Exit(1)
			}
			if bwcFramework.Spec.RunFile == "" {
				bwcFramework.Spec.RunFile = "main.py"
			}
		} else {
			fmt.Printf("Please enter the variable value.\n")
			fmt.Printf(" - Your Cmd: create <target resource> -d <target directory> -u \n")
			fmt.Printf(" - target resource: app or venv\n")
			fmt.Printf(" - target directory: app's directory path\n")

			os.Exit(1)
		}
		fmt.Printf("Venv created.\n")
		cmd = fmt.Sprintf("%s-%s", cliInfo.FirstCmd, cliInfo.TargetCmd)
	case "deploy":
		if cliInfo.DirOption == "" {
			fmt.Printf("Please input directory option(-d). \n")
			os.Exit(1)
		} else {
			bwcFramework = GetFrameworks(cliInfo.DirOption)
		}

		if cliInfo.TargetCmd == "app" {
			fmt.Printf("Deploy app in your device. \n")
			// 빈 값 찾기
			if bwcFramework.Spec.AppName == "" {
				fmt.Printf("Check spec's appName value in framework.yaml. \n")
				os.Exit(1)
			} else if cliInfo.UploadOption {
				if bwcFramework.Stackbase.TagName == "" {
					fmt.Printf("Check stackbase's tageName value in framework.yaml. \n")
					os.Exit(1)
				} else if bwcFramework.Stackbase.RepoName == "" {
					fmt.Printf("Check stackbase's repoName value in framework.yaml. \n")
					os.Exit(1)
				}
			} else if cliInfo.AppOption != "" {
				fmt.Printf("You can't use AppOption in deploy.\n")
				os.Exit(1)
			}

			if bwcFramework.Spec.Env.RunTime == "python" {
				if cliInfo.NameOption != "" {
					bwcFramework.Spec.AppName = cliInfo.NameOption
				}
				if bwcFramework.Spec.RunFile == "" {
					bwcFramework.Spec.RunFile = "main.py"
				}
				if bwcFramework.Spec.Env.VirtualEnv == "" {
					bwcFramework.Spec.Env.VirtualEnv = "base"
				}
			} else if bwcFramework.Spec.Env.RunTime == "go" {
				if cliInfo.NameOption != "" {
					bwcFramework.Spec.AppName = cliInfo.NameOption
				}
				if bwcFramework.Spec.RunFile == "" {
					fmt.Printf("Check spec's runFile value in framework.yaml. \n")
					os.Exit(1)
				}
				if bwcFramework.Spec.Env.VirtualEnv == "" {
					fmt.Printf("You not need virtual environment. \n")
				}
			}
		} else {
			fmt.Printf("Please enter the variable value.\n")
			fmt.Printf(" - Your Cmd: deploy <target resource> -d <target directory> -u \n")
			fmt.Printf(" - target resource: app or venv\n")
			fmt.Printf(" - target directory: app's directory path\n")
			fmt.Printf(" - -u: This is an option to upload to gitea or not. Enter -u if you are uploading, and leave out -u if you are not uploading.\n")

			os.Exit(1)
		}
		cmd = fmt.Sprintf("%s-%s", cliInfo.FirstCmd, cliInfo.TargetCmd)

	case "update":
		if cliInfo.DirOption == "" {
			fmt.Printf("Please input directory option(-d) \n")
			os.Exit(1)
		} else {
			bwcFramework = GetFrameworks(cliInfo.DirOption)
		}

		if cliInfo.TargetCmd == "venv" {
			fmt.Printf("Update venv in your device \n")
			if bwcFramework.Spec.Env.VirtualEnv == "" {
				fmt.Printf("Check spec's virtualEnvironment value in framework.yaml \n")
				os.Exit(1)
			} else if bwcFramework.Spec.Env.Package == "" {
				fmt.Printf("Check spec's package value in framework.yaml \n")
				os.Exit(1)
			}
			if bwcFramework.Spec.Env.Bin == "" {
				bwcFramework.Spec.Env.Bin = "python3"
			}
		} else if cliInfo.TargetCmd == "app" {
			fmt.Printf("Update app in your device. \n")
			// TODO App 업데이트가 정말 필요한 기능인가?
		} else {
			fmt.Printf("Please enter the variable value.\n")
			fmt.Printf(" - Your Cmd: update <target resource> -d <target directory>\n")
			fmt.Printf(" - target resource: app or venv\n")
			fmt.Printf(" - target directory: app's directory path\n")
			os.Exit(1)
		}
		cmd = fmt.Sprintf("%s-%s", cliInfo.FirstCmd, cliInfo.TargetCmd)
	case "delete":
		if cliInfo.TargetCmd == "app" {
			fmt.Printf("Delete app in your device \n")
		} else if cliInfo.TargetCmd == "venv" {
			fmt.Printf("Delete virtual environment in your device \n")
		} else {
			fmt.Printf("Please enter the variable value.\n")
			fmt.Printf(" - Your Cmd: delete <target resource> -n <target name>\n")
			fmt.Printf(" - target resource: app or venv\n")
			fmt.Printf(" - target name: app's name or venv's name\n")
			os.Exit(1)
		}
		cmd = fmt.Sprintf("%s-%s", cliInfo.FirstCmd, cliInfo.TargetCmd)
	case "get":
		if cliInfo.TargetCmd == "app" {
			fmt.Printf("Get app in your device \n")
		} else if cliInfo.TargetCmd == "venv" {
			fmt.Printf("Get virtual environment in your device \n")
		} else if cliInfo.TargetCmd == "bwc" {
			fmt.Printf("Get bwc process in your device \n")
		} else if cliInfo.TargetCmd == "template" {
			fmt.Printf("Get templates in your repos \n")
		} else {
			fmt.Printf("Please enter the variable value.\n")
			fmt.Printf(" - Your Cmd: get <target resource>\n")
			fmt.Printf(" - target resource: app or venv\n")
			os.Exit(1)
		}
		cmd = fmt.Sprintf("%s-%s", cliInfo.FirstCmd, cliInfo.TargetCmd)
	case "status":
		// fmt.Printf("Show status of device.\n")
		cmd = "status"
	case "info":
		// fmt.Printf("Show information of device.\n")
		cmd = "info"
	case "logs":
		if cliInfo.TargetCmd == "bwc" {
			fmt.Printf("Get bwc process's logs in your device \n")
		} else if cliInfo.TargetCmd == "app" {
			fmt.Printf("Get bwc process's logs in your device \n")
		} else {
			fmt.Printf("Please enter the variable value.\n")
			fmt.Printf(" - Your Cmd: logs <target resource>\n")
			fmt.Printf(" - target resource: bwc\n")
			os.Exit(1)
		}
		if cliInfo.NameOption == "" {
			fmt.Printf("Please enter name variable. (-n)")
			os.Exit(1)
		}

		cmd = fmt.Sprintf("%s-%s", cliInfo.FirstCmd, cliInfo.TargetCmd)
	case "init":
		if cliInfo.NameOption == "" {
			fmt.Printf("Please enter name variable. (-n)")
			os.Exit(1)
		}
		cmd = fmt.Sprintf("%s-%s", cliInfo.FirstCmd, cliInfo.TargetCmd)

	case "upload":
		if cliInfo.DirOption == "" {
			fmt.Printf("Please input directory option(-d) \n")
			os.Exit(1)
		} else {
			bwcFramework = GetFrameworks(cliInfo.DirOption)
		}
		cmd = "upload"
	case "wol":
		fmt.Printf("TEST - WOL \n")
		cmd = "wol"
	default:
		fmt.Printf("Not found: %s\n", cmd)
		fmt.Printf("Please enter the variable command value.\n")
		fmt.Printf("If you want to check what commands are available, run the help command.\n")
		fmt.Printf("Fallow as:\n")
		fmt.Printf("bwc help\n")
		os.Exit(1)
	}

	// Set Config PATH
	if archType == "win" {
		rootPath = "C:/sdt"
		appPath = "C:/sdt/app"
	} else {
		rootPath = "/etc/sdt"
		appPath = "/usr/local/sdt/app"
		// appPath = "."
	}

	// Set URL
	var svcInfo sdtType.ControlService
	if configData.ServiceType == "aws-dev" {
		svcInfo.GiteaURL = "http://43.200.53.170:32421"
		svcInfo.BwURL = "http://43.200.53.170:31731"
		svcInfo.GiteaIP = "43.200.53.170"
		svcInfo.MinioURL = "43.200.53.170:31191"
	} else if configData.ServiceType == "dev" {
		svcInfo.GiteaURL = "http://192.168.1.162:32421"
		svcInfo.BwURL = "http://192.168.1.162:31731"
		svcInfo.GiteaIP = "192.168.1.162"
		svcInfo.MinioURL = "192.168.1.162:31191"
	} else if configData.ServiceType == "eks" {
		svcInfo.GiteaURL = "http://cloud-repo.sdt.services"
		svcInfo.BwURL = "http://cloud-repo.sdt.services"
		svcInfo.GiteaIP = "cloud-repo.sdt.services"
		svcInfo.MinioURL = "s3"
	} else if configData.ServiceType == "onprem" {
		svcInfo.GiteaURL = fmt.Sprintf("http://%s:32421", configData.ServerIp)
		svcInfo.BwURL = fmt.Sprintf("http://%s:31731", configData.ServerIp)
		svcInfo.GiteaIP = configData.ServerIp
		svcInfo.MinioURL = fmt.Sprintf("%s:31191", configData.ServerIp)
	} else {
		fmt.Printf("Please check your device. Service Code not correct.")
		os.Exit(1)
	}

	// Set logger
	logFilePath := fmt.Sprintf("%s/device.logs/bwc-cli.log", rootPath)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	initError(logFile)
	sdtCli.Getlog(procLog)
	sdtCreate.Getlog(procLog)
	sdtDelete.Getlog(procLog)
	sdtDeploy.Getlog(procLog)
	sdtGet.Getlog(procLog)
	sdtGitea.Getlog(procLog)
	sdtInit.Getlog(procLog)
	sdtLogin.Getlog(procLog)
	sdtLogs.Getlog(procLog)
	sdtMessage.Getlog(procLog)
	sdtUpdate.Getlog(procLog)
	sdtUtil.Getlog(procLog)

	// Set HomeName
	bwcFramework.Spec.Env.HomeName = GetUser()

	sdtCli.RunBody(
		cmd, archType, rootPath,
		appPath, bwcFramework, cliInfo, svcInfo)
}
