// The clitype package defines structs for controls.
package cliType

import (
	"log"
)

// ControlService defines the structure for the environment information of the control agent.
//   - GiteaURL: URL of the code repository.
//   - BwURL: URL of BW API.
//   - GiteaIP: IP address of the code repository.
//   - GiteaPort: Port value of the code repository.
//   - MinioURL: URL of Minio.
type ControlService struct {
	GiteaURL  string
	BwURL     string
	GiteaIP   string
	GiteaPort int
	MinioURL  string
}

// This is the struct defining BWC-CLI command information.
//   - FirstCmd: Main function.
//   - TargetCmd: Subtype of the function.
//   - NameOption: Object name. (Objects can be apps, virtual environments, etc.)
//   - DirOption: Directory path.
//   - UploadOption: Option to upload an app. (Uploads to the code repository.)
//   - TailOption: Option to use the 'tail' function for logs.
//   - LineOption: Number of lines to display for logs.
//   - TemplateOption: App template name.
//   - AppOption: App status processing value. (For example, there is 'Restart'.)
type CliCmd struct {
	FirstCmd       string
	TargetCmd      string
	NameOption     string
	DirOption      string
	UploadOption   bool
	TailOption     bool
	LineOption     int
	TemplateOption string
	AppOption      string
}

// Definition of ConfigInfo Struct. ConfigInfo is the SDT Cloud config file used on devices.
//   - ModelName: Device model name.
//   - Admin: Device administrator.
//   - Group: Device group.
//   - AssetCode: Device serial number.
//   - MqttUrl: MQTT URL of SDT Cloud.
//   - ProjectCode: Project ID to which the device belongs.
//   - Organzation: Organzation ID to which the device belongs.
//   - DeviceType: Type of the device.
//   - Reboot: Reboot status of the device. (Used in reboot command.)
//   - RequestId: Command request ID.
//   - ServiceCode: SDT Cloud service code.
//   - ServiceType: SDT Cloud type. (EKS, DEV, OnPrem)
//   - SdtcloudId: SDT Cloud user ID.
//   - SdtcloudPw: SDT Cloud user password.
//   - AccessToken: Access token of SDT Cloud user.
type ConfigInfo struct {
	ModelName   string `json:"modelname"`
	Admin       string `json:"admin"`
	Group       string `json:"group"`
	AssetCode   string `json:"assetcode"`
	MqttUrl     string `json:"mqtturl"`
	ProjectCode string `json:"projectcode"`
	Organzation string `json:"organzation"`
	DeviceType  string `json:"devicetype"`
	Reboot      string `json:"reboot"`
	RequestId   string `json:"requestid"`
	ServiceCode string `json:"servicecode"`
	ServiceType string `json:"servicetype"`
	ServerIp    string `json:"serverip"`
	SdtcloudId  string `json:"sdtcloudid"`
	SdtcloudPw  string `json:"sdtcloudpw"`
	AccessToken string `json:"accesstoken"`
}

// Struct defining the format of logs.
//   - Warn: Warning log type.
//   - Info: General information log type.
//   - Error: Error type information.
type Logger struct {
	Warn  *log.Logger
	Info  *log.Logger
	Error *log.Logger
}

// Struct defining information about the device's project and PEM files.
//   - ProjectCode: Project ID to which the device belongs.
//   - RootCA: File path of RootCA for the associated project.
//   - PrivateKey: File path of PrivateKey for the associated project.
//   - Cert: File path of Cert for the associated project.
type ProjectControl struct {
	ProjectCode string `json: "projectCode"`
	RootCA      string `json: "rootCA"`
	PrivateKey  string `json: "privateKey"`
	Cert        string `json: "cert"`
}

// Struct defining app metadata information.
//   - AppName: Name of the app.
//   - AppId: ID of the app.
//   - AppVenv: Virtual environment used by the app.
type AppInfo struct {
	AppName string `json:"AppName"`
	AppId   string `json:"AppId"`
	AppVenv string `json:"AppVenv"`
	Managed string `json:"Managed"`
}

// Struct defining configuration information for managing app metadata on the device.
//   - AppInfoList: List variable of AppInfo Struct.
type AppConfig struct {
	AppInfoList []AppInfo `json:"AppInfoList"`
}

// Struct used for querying the list of apps.
//   - AppName: Name of the app.
//   - Status: Status of the app.
//   - AppId: ID of the app.
//   - AppVenv: Virtual environment used by the app.
type AppStatus struct {
	AppName string
	Status  string
	AppId   string
	AppVenv string
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

type Inference struct {
	WeightFile string `yaml:"weightFile" json:"weightFile"`
	Bucket     string `yaml:"bucket" json:"bucket"`
	Path       string `yaml:"path" json:"path"`
}

// Struct defining information about the framework file of the app.
//   - Version: Version name of the framework.
//   - Spec: Struct containing app specification information.
//   - Stackbase: Struct containing code repository information of the app.
type Framework struct {
	Version   string    `yaml:"version" json:"version"`
	Spec      Spec      `yaml:"spec" json:"spec"`
	Stackbase Stackbase `yaml:"stackbase" json:"stackbase"`
	Inference Inference `yaml:"inference" json:"inference"`
}

// Struct defining access credentials for SDT Cloud API calls.
//   - TokenType: Type of token.
//   - AccessToken: Access token for SDT Cloud API.
//   - RefreshToken: Refresh token.
//   - GiteaToken: Token for the code repository.
type AccessInfo struct {
	TokenType    string `json:"tokenType"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	GiteaToken   string `json:"giteaToken"`
}

// Struct defining repository information for the code repository.
//   - ID: Repository ID.
//   - FullName: Name including the repository path.
//   - Name: Repository name.
//   - Owner: User name of the repository owner.
type Repos struct {
	ID       int       `json:"id"`
	FullName string    `json:"full_name"`
	Name     string    `json:"name"`
	Owner    OwnerInfo `json:"owner"`
	// 여기에 다른 필요한 필드 추가
}

// Struct defining the username of the owner of the code repository.
//   - Username: Profile name of the user.
type OwnerInfo struct {
	Username string `json:"username"`
}

// Struct used for querying the list of app templates.
//   - Content: Struct containing app template information.
type TemplateInfo struct {
	Content []Repos
}

// Struct defining the user token and username for the code repository.
//   - AccessToken: Access token of the repository.
//   - Username: Profile name of the user.
type GiteaUser struct {
	AccessToken string `json:"accessToken"`
	Username    string `json:"userName"`
}
