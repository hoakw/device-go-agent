// The controltype package defines structs for controls.
package controlType

import (
	"log"
	"time"
)

const (
	// Status Code
	OK           = 200
	BAD_REQUEST  = 400
	UNAUTHORIZED = 401
	FORBIDDEN    = 403
	NOTFOUND     = 404
	CONFICT      = 409
)

// Struct defining the format of logs.
//   - Warn: Warning log type.
//   - Info: General information log type.
//   - Error: Error type information.
type Logger struct {
	Warn  *log.Logger
	Info  *log.Logger
	Error *log.Logger
}

// ConfigInfo defines the structure of the config file used by SDT Cloud on the device.
//   - AssetCode: Serial number of the device.
//   - MqttUrl: MQTT URL of SDT Cloud.
//   - ProjectCode: Project ID to which the device belongs.
//   - DeviceType: Type of the device.
//   - Reboot: Reboot status of the device (used in reboot commands).
//   - RequestId: Request ID of the command.
//   - ServiceCode: SDT Cloud service code.
//   - ServiceType: SDT Cloud service type (EKS, DEV, OnPerm).
type ConfigInfo struct {
	AssetCode   string `json:"assetcode"`
	DeviceType  string `json:"devicetype"`
	MqttUrl     string `json:"mqtturl"`
	ProjectCode string `json:"projectcode"`
	Reboot      string `json:"reboot"`
	RequestId   string `json:"requestid"`
	ServiceCode string `json:"servicecode"`
	ServiceType string `json:"servicetype"`
	ServerIp    string `json:"serverip"`
}

// ControlService defines the structure for the environment information of the control agent.
//   - MqttType: MQTT service type.
//   - ArchType: Device architecture.
//   - RootPath: Root path of BWC.
//   - HomeUser: Hostname of the device.
//   - SdtcloudIP: IP address of SDT Cloud.
//   - GiteaPort: Port value of the code repository.
type ControlService struct {
	MqttType         string
	ArchType         string
	RootPath         string
	AppPath          string
	MinicondaPath    string
	CommonPythonPath string
	VenvPath         string
	HomeUser         string
	SdtcloudIP       string
	GiteaPort        int
	MinioURL         string
	BaseCmd          [2]string
}

// CmdControl defines the structure for control command information.
//   - CmdInfo: Detailed information about the control command.
//   - CmdType: Type of the control command.
//   - AssetCode: Device serial number.
//   - SubCmdType: Subtype of the control command.
//   - RequestId: Request ID of the control command.
type CmdControl struct {
	CmdInfo    interface{} `json: "cmdInfo"`
	CmdType    string      `json: "cmdType"`
	AssetCode  string      `json: "assetCode"`
	SubCmdType string      `json: "subCmdType"`
	RequestId  string      `json: "requestId"`
}

// CmdBash defines the structure for bash control command information.
//   - Cmd: String value of the command to be executed.
type CmdBash struct {
	Cmd string `json: "cmd"`
}

// CmdSystemd defines the structure for systemd control command information.
//   - Cmd: Name of the systemd command to be executed (restart, start, stop, status, etc.).
//   - Service: Name of the Systemd service targeted by the command.
type CmdSystemd struct {
	Cmd     string `json: "cmd"`
	Service string `json: "service"`
}

// CmdDocker defines the structure for docker control command information.
//   - Cmd: Name of the docker command to be executed (run, stop, etc.).
//   - Image: Container image name.
//   - AppId: ID of the application.
//   - AppName: Name of the application.
//   - Options: Container options.
type CmdDocker struct {
	Cmd     string                 `json: "cmd"`
	Image   string                 `json: "image"`
	AppId   string                 `json: "appId"`
	AppName string                 `json: "appName"`
	Options map[string]interface{} `json: "options"`
}

// CmdVenv defines the structure for virtual environment control command information.
//   - VenvName: Name of the virtual environment.
//   - Requirement: Package requirements for virtual environment installation.
//   - BinFile: Binary file of the virtual environment.
//   - RunTime: Runtime environment of the virtual environment.
type CmdVenv struct {
	VenvName    string `json:"venvName"`
	Requirement string `json:"requirement"`
	BinFile     string `json:"binFile"`
	RunTime     string `json:"runTime"`
}

// CmdDeploy defines the structure for deployment control command information.
//   - AppId: ID of the application.
//   - AppName: Name of the application.
//   - App: Application name in the code repository.
//   - Image: Container image of the application (used only when deploying with Docker).
//   - FileUrl: Download path of the application.
//   - VenvName: Virtual environment name of the application.
//   - Env: The environment of app manager.
type CmdDeploy struct {
	AppId    string `json:"appId"`
	AppName  string `json:"appName"`
	App      string `json:"app"`
	AppType  string `json:"appType"`
	Image    string `json:"image"`
	FileUrl  string `json:"fileUrl"`
	VenvName string `json:"venvName"`
	// for inference
	Apps       []InferenceDeploy `json:"apps"`
	AppGroupId string            `json:"appGroupId"`
}

type InferenceDeploy struct {
	AppType        string                 `json:"appType"`
	AppId          string                 `json:"appId"`
	AppName        string                 `json:"appName"`
	App            string                 `json:"app"`
	FileUrl        string                 `json:"fileUrl"`
	VenvName       string                 `json:"venvName"`
	Parameter      map[string]interface{} `json:"parameter"`
	ModelId        string                 `json:"modelId"`
	ModelName      string                 `json:"modelName"`
	ModelVersion   int                    `json:"modelVersion"`
	ModelUrl       string                 `json:"modelUrl"`
	ModelFileKey   string                 `json:"modelFileKey"`
	ModelParameter map[string]interface{} `json:"modelParameter"`
	GpuIndex       int                    `json:"gpuIndex"`
}

type CmdModel struct {
	AppId        string                 `json:"appId"`
	AppName      string                 `json:"appName"`
	ModelId      string                 `json:"modelId"`
	ModelName    string                 `json:"modelName"`
	ModelVersion int                    `json:"modelVersion"`
	ModelUrl     string                 `json:"modelUrl"`
	ModelFileKey string                 `json:"modelFileKey"`
	Parameter    map[string]interface{} `json:"parameter"`
}

// CmdAppStart defines the structure for application start control command information.
//   - AppId: ID of the application.
//   - AppName: Name of the application.
type CmdAppStart struct {
	AppId   string `json:"appId"`
	AppName string `json:"appName"`
}

// CmdAppStop defines the structure for application stop control command information.
//   - AppId: ID of the application.
//   - AppName: Name of the application.
type CmdAppStop struct {
	AppId   string `json:"appId"`
	AppName string `json:"appName"`
}

// CmdDelete defines the structure for application delete control command information.
//   - AppId: ID of the application.
//   - AppName: Name of the application.
type CmdDelete struct {
	AppId   string `json:"appId"`
	AppName string `json:"appName"`
}

// CmdPid defines the structure for application PID information.
//   - AppId: ID of the application.
//   - AppName: Name of the application.
type CmdPid struct {
	AppId   string `json:"appId"`
	AppName string `json:"appName"`
}

// CmdJson defines the structure for application configuration control command information.
//   - AppId: ID of the application.
//   - AppName: Name of the application.
//   - FileName: Name of the config file to modify.
//   - Parameter: JSON content to modify.
type CmdJson struct {
	// Cmd      	string `json: "cmd"`
	AppId     string                 `json:"appId"`
	AppName   string                 `json:"appName"`
	FileName  string                 `json:"fileName"`
	ModelUrl  string                 `json:"modelUrl"`
	Parameter map[string]interface{} `json:"parameter"`
}

// Struct defining configuration information for managing app metadata on the device.
//   - AppInfoList: List variable of AppInfo Struct.
type AppConfig struct {
	AppInfoList []AppInfo `json:"AppInfoList"`
}

// AppInfo defines the structure for application metadata information.
//   - AppName: Name of the application.
//   - AppId: ID of the application.
//   - AppVenv: Virtual environment used by the application.
type AppInfo struct {
	AppName      string            `json:"AppName"`
	AppId        string            `json:"AppId"`
	AppVenv      string            `json:"AppVenv"`
	Managed      string            `json:"Managed"`
	AppGroupId   string            `json:"AppGroupId"`
	AppInference *AppInferenceInfo `json:"AppInference,omitempty"`
}

type AppInferenceInfo struct {
	ModelId      string `json:"ModelId"`
	ModelName    string `json:"ModelName"`
	ModelVersion int    `json:"ModelVersion"`
}

// Inference defining information about the inference type variable in the framework file of the app.
//   - WeightFile: Name of weight file
//   - Bucket: Bucket name in objectstorage.
//   - Path: Path of weight file in objectstorage.
//   - AccessKey: Access key for objectstoreage.
//   - SecretKey: Secret key for objectstoreage.
type Inference struct {
	WeightFile string `yaml:"weightFile" json:"weightFile"`
	Bucket     string `yaml:"bucket" json:"bucket"`
	Path       string `yaml:"path" json:"path"`
	AccessKey  string `yaml:"accessKey" json:"accessKey"`
	SecretKey  string `yaml:"secretKey" json:"secretKey"`
}

// Struct defining information about the Framework file of the app.
//   - Version: Version name of the Framework.
//   - Spec: Struct containing app specification information.
//   - Stackbase: Struct containing code repository information of the app.
type Framework struct {
	Version   string    `yaml:"version" json:"version"`
	Spec      Spec      `yaml:"spec" json:"spec"`
	Stackbase Stackbase `yaml:"stackbase" json:"stackbase"`
	Inference Inference `yaml:"inference" json:"inference"`
}

// Struct defining information about the spec type variable in the framework file of the app.
//   - AppName: Name of the app.
//   - RunFile: File for running the app.
//   - Env: Struct containing app environment information.
type Spec struct {
	AppName string `yaml:"appName" json:"appName"`
	AppType string `yaml:"appType" json:"appType"`
	RunFile string `yaml:"runFile" json:"runFile"`
	Env     Env    `yaml:"env" json:"env"`
}

// Struct defining information about the stackbase type variable in the framework file of the app.
//   - TagName: Release tag name to be stored in the code repository.
//   - RepoName: Repository name to be stored in the code repository.
type Stackbase struct {
	TagName  string `yaml:"tagName" json:"tagName"`
	RepoName string `yaml:"repoName" json:"repoName"`
}

// Struct defining information about the spec.env type variable in the framework file of the app.
//   - Bin: Binary file to execute the app.
//   - RunTime: Runtime of the app.
//   - VirtualEnv: Virtual environment of the app.
//   - HomeName: Hostname of the device.
//   - Package: File listing the required packages for the app.
type Env struct {
	Bin        string `yaml:"bin" json:"bin"`
	RunTime    string `yaml:"runtime" json:"runtime"`
	VirtualEnv string `yaml:"virtualEnv" json:"virtualEnv"`
	HomeName   string `yaml:"homeUser" json:"homeUser"`
	Package    string `yaml:"package" json:"package"`
}

type ResultMsg struct {
	AssetCode  string       `yaml:"assetCode" json:"assetCode"`
	Result     *CmdResult   `yaml:"result" json:"result,omitempty"`
	Results    *[]CmdResult `yaml:"results" json:"results,omitempty"`
	Status     CmdStatus    `yaml:"status" json:"status"`
	RequestId  string       `yaml:"requestId" json:"requestId"`
	AppGroupId string       `yaml:"appGroupId" json:"appGroupId"`
}

type CmdResult struct {
	Command         string                 `yaml:"command" json:"command"`
	SubCommand      string                 `yaml:"subCommand" json:"subCommand"`
	Pid             int                    `yaml:"pid" json:"pid"`
	Size            int64                  `yaml:"size" json:"size"`
	AppName         string                 `yaml:"appName" json:"appName"`
	VenvName        string                 `yaml:"venvName" json:"venvName"`
	VenvRequirement string                 `yaml:"venvRequirement" json:"venvRequirement,omitempty"`
	AppId           string                 `yaml:"appId" json:"appId"`
	Message         string                 `yaml:"message" json:"message"`
	Parameter       map[string]interface{} `yaml:"parameter" json:"parameter,omitempty"`
	ReleasedAt      int64                  `yaml:"releasedAt" json:"releasedAt"`
	UpdatedAt       int64                  `yaml:"updatedAt" json:"updatedAt"`
	AppRepoPath     string                 `yaml:"appRepoPath" json:"appRepoPath"`
	ModelName       string                 `yaml:"modelName" json:"modelName"`
	ModelVersion    int                    `yaml:"modelVersion" json:"modelVersion"`
	ModelId         string                 `yaml:"modelId" json:"modelId"`
	//Parameters   *[]map[string]interface{} `yaml:"parameters" json:"parameters,omitempty"`
}

type CmdStatus struct {
	Succeed    int    `yaml:"succeed" json:"succeed"`
	StatusCode int    `yaml:"statusCode" json:"statusCode"`
	ErrMsg     string `yaml:"errMsg" json:"errMsg"`
}

func NewCmdResult(command string, subCommand string, message string) CmdResult {
	return CmdResult{
		Command:      command,
		SubCommand:   subCommand,
		Pid:          -1,
		Size:         -1,
		AppName:      "",
		VenvName:     "",
		AppId:        "",
		Message:      message,
		Parameter:    nil,
		ReleasedAt:   int64(time.Now().UTC().Unix() * 1000),
		UpdatedAt:    int64(time.Now().UTC().Unix() * 1000),
		AppRepoPath:  "",
		ModelName:    "",
		ModelVersion: -1,
	}
}

func NewCmdStatus(statusCode int) CmdStatus {
	return CmdStatus{
		Succeed:    1,
		StatusCode: statusCode,
		ErrMsg:     "",
	}
}

func NewInferenceInfo() InferenceDeploy {
	return InferenceDeploy{
		ModelId:      "",
		ModelName:    "",
		ModelVersion: 0,
	}
}
