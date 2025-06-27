// The Login package provides functionality for logging into SDT Cloud on a device.
// User authentication with SDT Cloud is required for tasks such as creating app templates
// and deploying apps. User information must be registered on the device through login.
package login

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	sdtType "main/src/cliType"
)

// These are the global variables used in the Login package.
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

// SaveLoginInfo saves the user's login information through SDT Cloud, verifying
// it and storing the SDT Cloud user information on the device.
//
// Input:
//   - bwURL: SDT Cloud URL.
func SaveLoginInfo(bwURL string) {
	procLog.Info.Printf("Save user'info(login) in device.\n")
	// Check SDT Cloud user
	var userName, password string
	var tokenInfo sdtType.AccessInfo
	fmt.Printf("UserName for SDT Cloud: ")
	fmt.Scanln(&userName)
	fmt.Printf("PassWord for SDT Cloud: ")
	pw, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
	password = string(pw)

	apiUrl := fmt.Sprintf("%s/oauth/token", bwURL)
	payload := strings.NewReader(fmt.Sprintf("grantType=password&email=%s&password=%s", userName, password))

	req, err := http.NewRequest("POST", apiUrl, payload)
	if err != nil {
		procLog.Error.Printf("Http not connected. : %v\n", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		procLog.Error.Printf("Failed call api: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	statusArr := strings.Split(resp.Status, " ")
	statusValue, _ := strconv.Atoi(statusArr[0])
	//fmt.Printf("%s / %s \n", statusArr, statusValue)

	if statusValue == 200 {
		procLog.Info.Printf("Success login. \n")
		fmt.Printf("\nSuccess login. \n")
	} else {
		procLog.Error.Printf("Failed login. Please check your ID. \n")
		fmt.Printf("\nFailed login. Please check your ID. \n")
		os.Exit(1)
	}

	// Get TokenInfo
	err = json.Unmarshal(body, &tokenInfo)
	if err != nil {
		procLog.Error.Printf("Failed get tokeninfo: %v\n", err)
	}

	// Add Access info in device config file.
	configFile := "/etc/sdt/device.config/config.json"

	jsonFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		procLog.Error.Printf("Failed load app's file: %v\n", err)
	}
	var jsonData sdtType.ConfigInfo
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("Failed save app's Unmarshal: %v\n", err)
	}

	jsonData.SdtcloudId = userName
	jsonData.SdtcloudPw = password
	jsonData.AccessToken = tokenInfo.AccessToken

	saveJson, _ := json.MarshalIndent(&jsonData, "", "\t")
	err = ioutil.WriteFile(configFile, saveJson, 0644)
	if err != nil {
		procLog.Error.Printf("Failed save app's file: %v\n", err)
	}
	procLog.Info.Printf("Successfully save user'info(login) in device.\n")
}
