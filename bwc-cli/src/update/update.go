// The update package handles updating Python virtual environments. If additional
// Python packages need to be installed in the virtual environment, update manages
// the installation process.
package update

import (
	"fmt"
	sdtType "main/src/cliType"
	sdtUtil "main/src/util"
	"os"
	"os/exec"
)

// These are the global variables used in the Update package.
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

// UpdateVenv updates the virtual environment. Updating the virtual environment
// involves refreshing python packages, including adding or removing python packages.
// Virtual environments are managed at '/etc/sdt/venv' directory.
//
// Input:
//   - bwcFramework: App's framework information struct.
//   - envHome: Device's virtual environment management directory path.
//   - dirPath: Path to the package list (requirements.txt) file to update.
func UpdateVenv(bwcFramework sdtType.Framework, envHome string, dirPath string) {
	procLog.Info.Printf("Update %s venv.\n", bwcFramework.Spec.Env.VirtualEnv)
	envPath := fmt.Sprintf("%s/%s", envHome, bwcFramework.Spec.Env.VirtualEnv)

	// install pkg
	pkgFileName := fmt.Sprintf("%s/%s", dirPath, bwcFramework.Spec.Env.Package)
	pkgCmd := fmt.Sprintf("%s/bin/pip install -r %s", envPath, pkgFileName)
	cmd_run := exec.Command("sh", "-c", pkgCmd)
	_, cmd_err := cmd_run.Output()
	if cmd_err != nil {
		procLog.Error.Printf("Install package Error: %v\n", cmd_err)
		os.Exit(1)
	}

	// Copy requirment file in env path.
	destPath := fmt.Sprintf("%s/requirements.txt", envPath)
	sdtUtil.CopyFile(pkgFileName, destPath)
	procLog.Info.Printf("Successfully update venv.\n")
}
