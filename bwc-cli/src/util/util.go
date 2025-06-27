// The util package defines utility functions required by BWC-CLI functions.
package util

import (
	"errors"
	"fmt"
	"io"
	sdtType "main/src/cliType"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// These are the global variables used in the util package.
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

// CopyDir function copies a directory to a specified destination path.
//
// Input:
//   - srcPath: The source directory to copy.
//   - destPath: The destination path where the directory will be copied.
//
// Output:
//   - error: Error message if CopyDir command encounters an issue.
func CopyDir(srcPath, destPath string) error {
	procLog.Info.Printf("Copy dir - %s to %s.\n", srcPath, destPath)
	// Create the destination directory
	err := os.MkdirAll(destPath, os.ModePerm)
	if err != nil {
		procLog.Error.Printf("Error creating destination directory: %v\n", err)
		return err
	}

	// Get a list of all files and subdirectories in the source directory
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		procLog.Error.Printf("Error reading source directory: %v\n", err)
		return err
	}

	// Copy each file or subdirectory
	for _, entry := range entries {
		srcEntry := filepath.Join(srcPath, entry.Name())
		destEntry := filepath.Join(destPath, entry.Name())

		// if weight dir, not push.
		//if entry.Name() == "weight" || entry.Name() == "weights" {
		//	os.MkdirAll(destEntry, os.ModePerm)
		//	fmt.Println(destEntry)
		//	continue
		//}

		if entry.IsDir() {
			// Recursively copy subdirectories
			err := CopyDir(srcEntry, destEntry)
			if err != nil {
				procLog.Error.Printf("Copy error: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Copy files
			err := CopyFile(srcEntry, destEntry)
			if err != nil {
				procLog.Error.Printf("Copy error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	procLog.Info.Printf("Successfully copy dir - %s to %s.\n", srcPath, destPath)
	return nil
}

// CopyFile function copies a file to a specified destination path.
//
// Input:
//   - srcPath: The source file to copy.
//   - destPath: The destination path where the file will be copied.
//
// Output:
//   - error: Error message if CopyFile command encounters an issue.
func CopyFile(srcPath, destPath string) error {
	//procLog.Info.Printf("Copy file - %s to %s.\n", srcPath, destPath)
	srcFile, err := os.Open(srcPath)
	if err != nil {
		procLog.Error.Printf("Error opening source file: %v\n", err)
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		procLog.Error.Printf("Error creating destination file: %v\n", err)
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		procLog.Error.Printf("Error copying file content: %v\n", err)
		return err
	}

	procLog.Info.Printf("Successfully copy file - %s to %s.\n", srcPath, destPath)
	return nil
}

// GetDirectorySize function calculates the size of a directory.
//
// Input:
//   - path: The directory path.
//
// Output:
//   - int64: The size of the directory in bytes.
//   - error: Error message if GetDirectorySize command encounters an issue.
func GetDirectorySize(path string) (int64, error) {
	procLog.Info.Printf("Get dir's size: %s\n", path)
	var size int64

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories to avoid double-counting
		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	procLog.Info.Printf("Successfully get dir's size.\n")
	return size, nil
}

// GetPid function retrieves the PID of an application.
//
// Input:
//   - appName: The name of the application.
//
// Output:
//   - int: The PID (Process ID) of the application.
//   - error: Error message if GetPid command encounters an issue.
func GetPid(appName string) (int, error) {
	procLog.Info.Printf("Get %s app's pid.\n", appName)
	getPid := fmt.Sprintf("systemctl show --property MainPID %s", appName)
	cmd_run := exec.Command("sh", "-c", getPid)
	stdout1, err := cmd_run.Output()
	if err != nil {
		procLog.Error.Printf("Failed get pid: %v\n", err)
		return -1, err
	}
	strOut := string(stdout1)
	pidStr := strings.Split(strOut[:len(strOut)-1], "=")
	if pidStr[len(pidStr)-1] == "" {
		procLog.Error.Printf("%s not found.\n", appName)
		err = errors.New("process not found")
		return -1, err
	}
	pid, err := strconv.Atoi(pidStr[len(pidStr)-1])
	if err != nil {
		procLog.Error.Printf("Failed convert pid (string -> int) error:  %v\n", err)
		return -1, err
	} else if pid == 0 {
		procLog.Error.Printf("%s not found.\n", appName)
		err = errors.New("App not found")
		return -1, err
	}
	return pid, err
}

func GetJournalCtl(appName string) string {
	cmd_log := exec.Command("journalctl", "-u", appName, "-n", "30")
	stdout, _ := cmd_log.Output()
	return string(stdout)
}

// ChownCmd function changes the ownership of a directory.
//
// Input:
//   - targetDir: The directory path.
//   - usrname: The username to assign ownership.
//
// Output:
//   - string: Result of the ChownCmd command.
func ChownCmd(targetDir string, usrname string) string {
	cmd := fmt.Sprintf("chown -R %s:%s %s", usrname, usrname, targetDir)
	cmd_log := exec.Command("sh", "-c", cmd)
	stdout, err := cmd_log.Output()
	if err != nil {
		procLog.Error.Printf("chown cmd failed: %v\n", err)
	}
	return string(stdout)
}

// ReadLastNLines function extracts the last N lines including newline characters from a string.
//
// Input:
//   - contents: The string including newline characters.
//   - n: The number of lines to retrieve.
//
// Output:
//   - []string: Array of strings containing the extracted lines.
func ReadLastNLines(contents string, n int) []string {
	lines := strings.Split(contents, "\n")
	n = n + 1
	return lines[len(lines)-n:]
}

// Contains function checks if a specific string exists in a list of strings.
//
// Input:
//   - elems: List of strings.
//   - v: String to check for.
//
// Output:
//   - bool: True if the string exists in the list, false otherwise.
func Contains(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func CreateInferenceDir(appDir string) error {
	procLog.Info.Printf("[Deploy] Make inference directory(weights, result, logs, data).\n")
	weightFile := fmt.Sprintf("%s/weights", appDir)
	err := os.MkdirAll(weightFile, os.ModePerm)
	if err != nil {
		procLog.Error.Printf("Error creating destination directory: %v\n", err)
		return err
	}
	resultFile := fmt.Sprintf("%s/result", appDir)
	err = os.MkdirAll(resultFile, os.ModePerm)
	if err != nil {
		procLog.Error.Printf("Error creating destination directory: %v\n", err)
		return err
	}
	logsFile := fmt.Sprintf("%s/logs", appDir)
	err = os.MkdirAll(logsFile, os.ModePerm)
	if err != nil {
		procLog.Error.Printf("Error creating destination directory: %v\n", err)
		return err
	}
	dataFile := fmt.Sprintf("%s/data", appDir)
	err = os.MkdirAll(dataFile, os.ModePerm)
	if err != nil {
		procLog.Error.Printf("Error creating destination directory: %v\n", err)
		return err
	}

	return nil
}
