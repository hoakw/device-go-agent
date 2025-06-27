package docker

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	sdtDeploy "main/src/deploy"
	"net/http"
	"os"

	containerType "github.com/docker/docker/api/types/container"

	sdtType "main/src/controlType"
)

// These are the global variables used in the deploy package.
// - procLog: This is the struct that defines the format of the Log.
var procLog sdtType.Logger

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// The StopContainer function stops an application deployed on the device.
//
// Input:
//   - cli: The variable docker client.
//   - appName: The name of the application.
//   - appId: The ID of the application.
//
// Output:
//   - error: Error message in case of issues with the stop command.
//   - int: Status code of the command execution.
func StopContainer(cli *client.Client, appName string, appId string) (error, int) {
	removeName := fmt.Sprintf("%s-%s", appName, appId)
	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		procLog.Error.Printf("[DOCKER] Dockerclient error: %v\n", err)
		return err, http.StatusBadRequest
	}

	for _, container := range containers {
		//procLog.Info.Printf("%s %s\n", container.ID[:10], container.Image)
		containerName := container.Names[0][1:]
		if removeName == containerName {
			procLog.Info.Printf("[DOCKER] Get %s / %s-> Stop\n", removeName, containerName)
			if err = cli.ContainerStop(ctx, container.ID, containerType.StopOptions{}); err != nil {
				procLog.Error.Printf("[ERROR] Cannot stop container: %v\n", err)
				return err, http.StatusBadRequest
			}
			return nil, http.StatusOK
		}
	}

	err = errors.New("Not found container.")
	return err, http.StatusNotFound
}

// The StartContainer function starts an application deployed on the device.
//
// Input:
//   - cli: The variable docker client.
//   - appName: The name of the application.
//   - appId: The ID of the application.
//
// Output:
//   - error: Error message in case of issues with the start command.
//   - int: Status code of the command execution.
func StartContainer(cli *client.Client, appName string, appId string) (error, int) {
	removeName := fmt.Sprintf("%s-%s", appName, appId)
	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		procLog.Error.Printf("[DOCKER] Dockerclient error: %v\n", err)
		return err, http.StatusBadRequest
	}

	for _, container := range containers {
		//procLog.Info.Printf("%s %s\n", container.ID[:10], container.Image)
		containerName := container.Names[0][1:]
		if removeName == containerName {
			procLog.Info.Printf("[DOCKER] Get %s / %s-> Stop\n", removeName, containerName)
			if err = cli.ContainerStart(ctx, container.ID, types.ContainerStartOptions{}); err != nil {
				procLog.Error.Printf("[ERROR] Cannot start container: %v\n", err)
				return err, http.StatusBadRequest
			}
			return nil, http.StatusOK
		}
	}

	err = errors.New("Not found container.")
	return err, http.StatusNotFound
}

// The DeleteContainer function deletes an application deployed on the device.
//
// Input:
//   - cli: The variable docker client.
//   - appName: The name of the application.
//   - appId: The ID of the application.
//
// Output:
//   - error: Error message in case of issues with the delete command.
//   - int: Status code of the command execution.
func DeleteContainer(cli *client.Client, appName string, appId string, rootPath string) (error, int) {
	removeName := fmt.Sprintf("%s-%s", appName, appId)
	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		procLog.Error.Printf("[DOCKER] Dockerclient error: %v\n", err)
		return err, http.StatusBadRequest
	}

	for _, container := range containers {
		//procLog.Info.Printf("%s %s\n", container.ID[:10], container.Image)
		containerName := container.Names[0][1:]
		if removeName == containerName {
			procLog.Info.Printf("[DOCKER] Get %s / %s-> Remove\n", removeName, containerName)
			if err = cli.ContainerStop(ctx, container.ID, containerType.StopOptions{}); err != nil {
				procLog.Error.Printf("[ERROR] Cannot stop container: %v\n", err)
				return err, http.StatusBadRequest
			}
			if err = cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{}); err != nil {
				procLog.Error.Printf("[ERROR] Cannot remove container: %v\n", err)
				return err, http.StatusBadRequest
			}

			sdtDeploy.DeleteAppInfo(appName, rootPath)
			return nil, http.StatusOK
		}
	}

	err = errors.New("Not found container.")

	return err, http.StatusNotFound
}

// The Deploy function deploys an application onto the device. Deploying as docker container
//
// Input:
//   - cli: The variable docker client.
//   - imageName: The name of container image.
//   - appName: The name of the application.
//   - appId: The ID of the application.
//
// Output:
//   - error: Error message in case of issues with the deploy command.
//   - int: Status code of the command execution.
func CreateContainer(cli *client.Client,
	imageName string,
	appName string,
	appId string,
	rootPath string,
	portData []interface{},
	envData map[string]interface{}) (error, int) {

	containerName := fmt.Sprintf("%s-%s", appName, appId)
	procLog.Info.Printf("[DEPLOY][DOCKER] Create Container: %s, Image: %s\n", containerName, imageName)

	ctx := context.Background()
	reader, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		procLog.Error.Printf("[DEPLOY][DOCKER] Found error in imagePull: %v\n", err)
		return err, http.StatusBadRequest
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)

	var newport nat.Port
	var portConfig []nat.PortBinding
	var hostConfig *containerType.HostConfig
	var containerConfig *containerType.Config
	var envConfig = make([]string, 0)

	// Set container Port
	if portData != nil {
		for _, portInterface := range portData {
			portmap := portInterface.(map[string]interface{})
			port := nat.PortBinding{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", int(portmap["hostPort"].(float64))),
			}
			newport, _ = nat.NewPort(portmap["protocol"].(string), fmt.Sprintf("%d", int(portmap["containerPort"].(float64))))
			portConfig = append(portConfig, port)
		}
		hostConfig = &containerType.HostConfig{
			PortBindings: nat.PortMap{
				newport: portConfig,
			},
		}
	}

	// Set container Env Variable
	if envData != nil {
		for keys, vals := range envData {
			envConfig = append(envConfig, fmt.Sprintf("%s=%s", keys, vals))
		}

		// Set container config
		containerConfig = &containerType.Config{
			Image: imageName,
			Env:   envConfig,
		}
	} else {
		containerConfig = &containerType.Config{
			Image: imageName,
		}
	}

	// Call docker client for docker create
	resp, err := cli.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		containerName,
	)
	if err != nil {
		procLog.Error.Printf("[DEPLOY][DOCKER] Found error in Container Create: %v\n", err)
		return err, http.StatusBadRequest
	}

	if err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		procLog.Error.Printf("[DEPLOY][DOCKER] Found error in Container Start: %v\n", err)
		return err, http.StatusBadRequest
	}

	sdtDeploy.SaveAppInfo(appName, appId, "", "dockerd", sdtType.NewInferenceInfo(), "", rootPath)

	return nil, http.StatusOK
}
