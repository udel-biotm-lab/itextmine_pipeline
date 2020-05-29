package rlimsp

import (
	"context"
	"fmt"
	"itextmine/misc"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/gammazero/workerpool"
)

func Execute(workDir string, numParallelTasks int) error {
	dockerClient := misc.CreateDockerClient()

	ctx := context.Background()

	// remove rlimsp network before starting
	networkRemoveError := misc.RemoveNetwork(ctx, dockerClient, "rlimsp")
	if networkRemoveError != nil {
		return networkRemoveError
	}

	// create network
	networkID, networkCreateError := createRlimspNetwork(ctx, dockerClient)
	if networkCreateError != nil {
		return networkCreateError
	}

	// remove network when we are done
	defer dockerClient.NetworkRemove(ctx, networkID)

	// start the rlimsp mysql container
	rlimsMySQLContainerID, rlimspMysqlStartError := startRLIMSPMySQLContainer(ctx, dockerClient)
	if rlimspMysqlStartError != nil {
		return rlimspMysqlStartError
	}

	// remove this container when we are done
	defer dockerClient.ContainerRemove(ctx, rlimsMySQLContainerID, types.ContainerRemoveOptions{Force: true})

	// pull rlimsp docker image
	pullError := misc.PullImage(ctx, dockerClient, "itextmine/rlimsp")
	if pullError != nil {
		return pullError
	}

	// get a list of all the tasks
	rlimsWorkDirPath := path.Join(workDir, "rlimsp")
	tasks, tasksError := misc.GetSubDirNames(rlimsWorkDirPath)
	if tasksError != nil {
		return tasksError
	}

	// create a worker pool and start the execution
	wp := workerpool.New(numParallelTasks)

	// make a buffered channel to receive errors
	errorChan := make(chan error, len(*tasks))

	// make a buffered channel to receive progress
	progressChan := make(chan bool, len(*tasks))

	// make a done channel to signal work completion
	terminateChan := make(chan bool)

	// start a goroutine to handle the messages from worker pool
	go handleMessage(errorChan, progressChan, terminateChan, len(*tasks))

	for _, task := range *tasks {
		taskCopy := task
		println("Sending " + taskCopy)
		wp.Submit(func() {
			println("Executing " + taskCopy)
			rlimsContainerError := executeRLIMSPContainer(ctx, dockerClient, taskCopy, workDir)
			if rlimsContainerError != nil {
				errorChan <- rlimsContainerError
			}
			progressChan <- true
		})
	}

	wp.StopWait()

	// close the error channel
	close(errorChan)
	close(progressChan)

	// send a message on done channel to quit the goroutine
	terminateChan <- true

	return nil
}

func handleMessage(errorChan chan error, progressChan chan bool, terminateChan chan bool, taskCount int) {
	// create and start new bar
	// bar := pb.StartNew(taskCount)

	for {
		select {
		case err := <-errorChan:
			if err != nil {
				println(err.Error())
			}
		case isTaskDone := <-progressChan:
			if isTaskDone {
				//bar.Increment()
				println("Task done")
			}
		case isTerminate := <-terminateChan:
			if isTerminate {
				//bar.Finish()
				return
			}
		}
	}

}

func executeRLIMSPContainer(ctx context.Context, dockerClient *client.Client, taskName string, workdir string) error {

	// network config
	rlimspNetworkConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"rlimsp": &network.EndpointSettings{
				IPAddress: "10.0.0.2",
			},
		},
	}

	taskInputAbsolutePath, inputPathError := filepath.Abs(path.Join(workdir, "rlimsp", taskName, "input.json"))
	if inputPathError != nil {
		return inputPathError
	}

	taskOutputJsonAbsolutePath, jsonOutputPathError := filepath.Abs(path.Join(workdir, "rlimsp", taskName, "output.json"))
	if jsonOutputPathError != nil {
		return jsonOutputPathError
	}
	// create the output.json
	jsonFile, jsonCreateError := os.Create(taskOutputJsonAbsolutePath)
	if jsonCreateError != nil {
		return jsonCreateError
	}
	jsonFile.Close()

	taskOutputTxtAbsolutePath, txtOutputPathError := filepath.Abs(path.Join(workdir, "rlimsp", taskName, "output.txt"))
	if txtOutputPathError != nil {
		return txtOutputPathError
	}
	// create the output.txt
	txtFile, txtCreateError := os.Create(taskOutputTxtAbsolutePath)
	if txtCreateError != nil {
		return txtCreateError
	}
	txtFile.Close()

	// host config
	hostConfig := container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s:ro", taskInputAbsolutePath, "/rlims_workdir/in.json"),
			fmt.Sprintf("%s:%s", taskOutputJsonAbsolutePath, "/rlims_workdir/out.json"),
			fmt.Sprintf("%s:%s", taskOutputTxtAbsolutePath, "/rlims_workdir/out.txt"),
		},
	}

	// container config
	containerConfig := container.Config{
		Image: "itextmine/rlimsp",
	}

	// create the container
	containerCreateResponse, containerCreateError := dockerClient.ContainerCreate(ctx,
		&containerConfig,
		&hostConfig,
		&rlimspNetworkConfig,
		fmt.Sprintf("rlimsp-%s", taskName))

	if containerCreateError != nil {
		return containerCreateError
	}

	// start this container
	containerStartError := dockerClient.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
	if containerStartError != nil {
		return containerStartError
	}

	// wait for container to be done running
	status, waitErr := dockerClient.ContainerWait(ctx, containerCreateResponse.ID)
	println(status)
	if waitErr != nil {
		return waitErr
	}
	// remove the container when we are done
	defer dockerClient.ContainerRemove(ctx, containerCreateResponse.ID, types.ContainerRemoveOptions{Force: true})

	// check the output
	checkoutputErr := misc.CheckOutput(taskOutputJsonAbsolutePath)
	if checkoutputErr != nil {
		return checkoutputErr
	}

	return nil
}

func createRlimspNetwork(ctx context.Context, dockerClient *client.Client) (string, error) {
	networkOptions := types.NetworkCreate{
		CheckDuplicate: false,
		Driver:         "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				network.IPAMConfig{
					Subnet: "10.0.0.0/16",
				},
			},
		},
	}
	netWorkCreateResponse, err := dockerClient.NetworkCreate(ctx, "rlimsp", networkOptions)

	if err != nil {
		return "", err
	} else {
		return netWorkCreateResponse.ID, err
	}
}

func startRLIMSPMySQLContainer(ctx context.Context, dockerClient *client.Client) (string, error) {

	// pull the image
	pullError := misc.PullImage(ctx, dockerClient, "itextmine/rlimsp-mysql")
	if pullError != nil {
		return "", pullError
	}

	// create the container
	rlimsMySQLNetworkConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"rlimsp": &network.EndpointSettings{
				IPAddress: "10.0.0.2",
			},
		},
	}
	containerCreateResponse, containerCreateError := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: "itextmine/rlimsp-mysql",
	}, nil, &rlimsMySQLNetworkConfig, "rlimsp-mysql")

	if containerCreateError != nil {
		return "", containerCreateError
	}

	// start this container
	containerStartError := dockerClient.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
	if containerStartError != nil {
		return "", containerStartError
	}

	// sleep for 5 seconds for the db to init
	time.Sleep(5 * time.Second)

	return containerCreateResponse.ID, nil

}
