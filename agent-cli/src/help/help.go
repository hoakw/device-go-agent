// Help package provides descriptions of BWC-CLI commands.
package help

import (
	"fmt"
)

// PrintHelp function prints usage and description of commands provided by BWC-CLI.
func PrintHelp() {
	fmt.Printf("Init Example  : bwc init app -n <app name> \n")
	fmt.Printf("Create Example: bwc create app|venv -d <target directory> \n")
	fmt.Printf("Deploy Example: bwc deploy app -d <target directory> \n")
	fmt.Printf("Delete Example: bwc delete app|venv -n <target name>\n")
	fmt.Printf("Get Example   : bwc get app|venv\n")
	fmt.Printf("Status Example: bwc status\n")
	fmt.Printf("Info Example  : bwc info\n")
	fmt.Printf("Logs Example  : bwc logs bwc|app -n <service name|app name>\n")

	fmt.Printf("\n")
	fmt.Printf("[init] : It create app.\n")
	fmt.Printf("  - If you use this command, you must enter the following command:\n")
	fmt.Printf("  	- bwc init app [-n,-name] [-t,-template]\n")
	fmt.Printf("  	- [-n,-name]: Created directroy's name.\n")
	fmt.Printf("  	- [-t,-template]: Template name.\n")

	fmt.Printf("\n")
	fmt.Printf("[create] : It create app and venv in your device. In the case of app creation, this is to check whether the app operates well. To deploy an app, you must use the deploy command.\n")
	fmt.Printf("           In the case of app creation, this is to check whether the app operates well. To deploy an app, you must use the deploy command.\n")
	fmt.Printf("  - If you use this command, you must enter the following command:\n")
	fmt.Printf("  	- bwc create [app|venv] [-d,-directory] \n")
	fmt.Printf("  	- [app|venv]: Target resource.\n")
	fmt.Printf("  	- [-d,-directory]: App's directory or directory path's framework.yaml\n")

	fmt.Printf("\n")
	fmt.Printf("[deploy] : It deploy app in your device.\n")
	fmt.Printf("  - If you use this command, you must enter the following command:\n")
	fmt.Printf("  	- bwc deploy app [-d,-directory] [-u,-upload]\n")
	fmt.Printf("  	- app: Target resource.\n")
	fmt.Printf("  	- [-d,-directory]: App's directory or directory path's framework.yaml\n")
	fmt.Printf("  	- [-u,-upload]: This is an option to upload to gitea or not. Enter -u if you are uploading, and leave out -u if you are not uploading.\n")

	fmt.Printf("\n")
	fmt.Printf("[update] : It update venv's package in your device.\n")
	fmt.Printf("  - If you use this command, you must enter the following command:\n")
	fmt.Printf("  	- bwc update venv [-d,-directory]\n")
	fmt.Printf("  	- app: Target resource.\n")
	fmt.Printf("  	- [-d,-directory]: App's directory or directory path's framework.yaml\n")

	fmt.Printf("\n")
	fmt.Printf("[delete] : It delete app in your device.\n")
	fmt.Printf("  - If you use this command, you must enter the following command:\n")
	fmt.Printf("  	- bwc delete [app|venv] [-n,-name]\n")
	fmt.Printf("  	- [app|venv]: Target resource.\n")
	fmt.Printf("  	- [-n,-name]: App or virtual environment name.\n")

	fmt.Printf("\n")
	fmt.Printf("[get] : It show apps or virtual environments in your device.\n")
	fmt.Printf("  - If you use this command, you must enter the following command:\n")
	fmt.Printf("    - bwc get [app|venv|bwc]\n")
	fmt.Printf("  	- [app|venv]: Target resource.\n")

	fmt.Printf("\n")
	fmt.Printf("[status] : It show device. This shows the device's registration and connection status to SDT Cloud. \n")
	fmt.Printf("  - If you use this command, you must enter the following command:\n")
	fmt.Printf("    - bwc status\n")

	fmt.Printf("\n")
	fmt.Printf("[info] : It show information of device. This shows the device's projectcode, assetcode, type and etc.\n")
	fmt.Printf("  - If you use this command, you must enter the following command:\n")
	fmt.Printf("    - bwc info\n")

	fmt.Printf("\n")
	fmt.Printf("[logs] : It show log's bwc process or app in device. \n")
	fmt.Printf("  - If you use this command, you must enter the following command:\n")
	fmt.Printf("    - bwc logs [bwc|app] [-n,-name]\n")
	fmt.Printf("  	- [-n,-name]: process or app name.\n")
}
