package tools

import (
	"context"
	"errors"
	"fmt"
	"itextmine/constants"
	"itextmine/misc"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/gammazero/workerpool"
)

func ExecuteRlimsp(workDir string, numParallelTasks int) error {
	dockerClient := misc.CreateDockerClient()

	ctx := context.Background()

	log.Println("Cleaning up docker env from previous run")
	// cleanup from previous run
	cleanupError := cleanUpRlimsp(ctx, dockerClient)
	if cleanupError != nil {
		return cleanupError
	}

	// create rlimsp network
	log.Println(fmt.Sprintf("Creating %s network", constants.RLIMS_NETWORK_NAME))
	networkID, networkCreateError := createRlimspNetwork(ctx, dockerClient)
	if networkCreateError != nil {
		return networkCreateError
	}

	// start the rlimsp mysql container if does not exists
	log.Println(fmt.Sprintf("Creating %s container", constants.RLIMS_MYSQL_CONTAINER_NAME))
	rlimsMySQLContainerID, rlimspMysqlStartError := startRLIMSPMySQLContainer(ctx, dockerClient)
	if rlimspMysqlStartError != nil {
		return rlimspMysqlStartError
	}

	// remove network when we are done
	defer dockerClient.NetworkRemove(ctx, networkID)

	// remove this container when we are done
	defer dockerClient.ContainerRemove(ctx, rlimsMySQLContainerID, types.ContainerRemoveOptions{Force: true})

	// pull rlimsp docker image
	rlimsPullError := misc.PullImage(ctx, dockerClient, "itextmine/rlimsp")
	if rlimsPullError != nil {
		return rlimsPullError
	}

	// pull efip docker image
	efipPullError := misc.PullImage(ctx, dockerClient, "leebird/efip")
	if efipPullError != nil {
		return efipPullError
	}

	// pull align docker image
	alignPullError := misc.PullImage(ctx, dockerClient, "itextmine/align")
	if alignPullError != nil {
		return alignPullError
	}

	// get a list of all the tasks
	rlimsWorkDirPath := path.Join(workDir, "rlimsp")
	log.Println(fmt.Sprintf("Generating tasks from : %s ", rlimsWorkDirPath))
	tasks, tasksError := misc.GetSubDirNames(rlimsWorkDirPath)
	if tasksError != nil {
		return tasksError
	}

	// create a worker pool and start the execution
	wp := workerpool.New(numParallelTasks)

	// number of tasks
	num_tasks := len(*tasks) * 2 // multiple by two as we execute both rlimsp and efip together
	log.Println(fmt.Sprintf("Generated %d tasks", num_tasks))

	// make a buffered channel to receive errors in go routine
	errorChan := make(chan error, num_tasks)

	// make a buffered channel to receive progress
	progressChan := make(chan bool, num_tasks)

	// make a done channel to signal work completion
	terminateChan := make(chan bool)

	// start a goroutine to handle the messages from worker pool
	go handleProgress(progressChan, terminateChan, num_tasks)

	log.Println(fmt.Sprintf("Starting the pool with %d workers", numParallelTasks))

	for _, task := range *tasks {
		taskCopy := task
		wp.Submit(func() {
			// execute rlimsp container
			rlimsContainerError := executeRLIMSPContainer(ctx, dockerClient, taskCopy, workDir)
			if rlimsContainerError != nil {
				log.Println(fmt.Sprintf("ERROR: %s", rlimsContainerError.Error()))
				errorChan <- rlimsContainerError
			}

			progressChan <- true
			// execute efip container
			efipContainerError := ExecuteEfipContainer(ctx, dockerClient, taskCopy, workDir)
			if efipContainerError != nil {
				log.Println(fmt.Sprintf("ERROR: %s", efipContainerError.Error()))
				errorChan <- efipContainerError
			}

			progressChan <- true
		})
	}

	wp.StopWait()

	// close the channels channel
	close(errorChan)
	close(progressChan)

	// get the errors from error channel
	errors := make([]error, 0)
	for err := range errorChan {
		errors = append(errors, err)
	}

	// send a message on done channel to quit the goroutine
	terminateChan <- true

	// check if we had any errors
	if len(errors) > 0 {
		error_value := errors[0]
		return error_value
	}

	return nil
}

func handleProgress(progressChan chan bool, terminateChan chan bool, taskCount int) {
	// create and start new bar
	bar := pb.Full.Start(taskCount)

	for {
		select {
		case isTaskDone := <-progressChan:
			if isTaskDone {
				bar.Increment()
			}
		case isTerminate := <-terminateChan:
			if isTerminate {
				bar.Finish()
				return
			}
		}
	}

}

func executeRLIMSPContainer(ctx context.Context, dockerClient *client.Client, taskName string, workdir string) error {

	// network config
	rlimspNetworkConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"rlimsp": {
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
	_, waitErr := dockerClient.ContainerWait(ctx, containerCreateResponse.ID)
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

	// create the absolute path for align output
	alignOutputAbsolutePath, alignOutputAbsolutePathError := filepath.Abs(path.Join(workdir, "rlimsp", taskName, "align.json"))
	if alignOutputAbsolutePathError != nil {
		return alignOutputAbsolutePathError
	}

	// run alignment
	alignError := ExecuteAlign(ctx, dockerClient, taskName, taskInputAbsolutePath, taskOutputJsonAbsolutePath, alignOutputAbsolutePath, "rlimsp")
	if alignError != nil {
		return alignError
	}

	return nil
}

func createRlimspNetwork(ctx context.Context, dockerClient *client.Client) (string, error) {
	networkOptions := types.NetworkCreate{
		CheckDuplicate: false,
		Driver:         "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet: "10.0.0.0/16",
				},
			},
		},
	}
	netWorkCreateResponse, err := dockerClient.NetworkCreate(ctx, constants.RLIMS_NETWORK_NAME, networkOptions)

	if err != nil {
		return "", err
	} else {
		return netWorkCreateResponse.ID, nil
	}
}

func startRLIMSPMySQLContainer(ctx context.Context, dockerClient *client.Client) (string, error) {

	// check if rlimsp mysql container exists

	// pull the image
	pullError := misc.PullImage(ctx, dockerClient, "itextmine/rlimsp-mysql")
	if pullError != nil {
		return "", pullError
	}

	// create the container
	rlimsMySQLNetworkConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			constants.RLIMS_NETWORK_NAME: {
				IPAddress: "10.0.0.2",
			},
		},
	}
	containerCreateResponse, containerCreateError := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: "itextmine/rlimsp-mysql",
	}, nil, &rlimsMySQLNetworkConfig, constants.RLIMS_MYSQL_CONTAINER_NAME)

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

func ReduceRlimsp(toolWorkDir string, toolOutputDir string, collectionType string) error {

	// build reduce align json
	alignOutputFilePath := fmt.Sprintf("%s/rlimsp.%s.align.json", toolOutputDir, collectionType)
	reduceAlignCmdStr := fmt.Sprintf("cat %s/*/align.json > %s", toolWorkDir, alignOutputFilePath)

	log.Println(fmt.Sprintf("Reducing RLIMSP results to : %s", alignOutputFilePath))

	// execute the command
	reduceAlignCmdErr, _, reduceAlignCmdErrOut := misc.Shellout(reduceAlignCmdStr)
	if reduceAlignCmdErr != nil {
		return errors.New(reduceAlignCmdErrOut)
	}

	// check reduce output
	reduceOutputCheckError := misc.CheckOutput(alignOutputFilePath)
	if reduceOutputCheckError != nil {
		return reduceOutputCheckError
	}

	return nil

}

func cleanUpRlimsp(ctx context.Context, dockerClient *client.Client) error {
	// remove dangling rlimsp containers
	danglingRlimsRemoveError := misc.RemoveContainer(ctx, dockerClient, "rlimsp-task*")
	if danglingRlimsRemoveError != nil {
		return danglingRlimsRemoveError
	}

	// remove dangling align rlimsp containers
	danglingRlimsAlignRemoveError := misc.RemoveContainer(ctx, dockerClient, "rlimsp-align*")
	if danglingRlimsAlignRemoveError != nil {
		return danglingRlimsAlignRemoveError
	}

	// remove dangling efip containers
	danglingEfipRemoveError := misc.RemoveContainer(ctx, dockerClient, "rlimsp-efip-task*")
	if danglingEfipRemoveError != nil {
		return danglingEfipRemoveError
	}

	// remove dangling align rlimsp containers
	danglingEfipAlignRemoveError := misc.RemoveContainer(ctx, dockerClient, "efip-align*")
	if danglingEfipAlignRemoveError != nil {
		return danglingEfipAlignRemoveError
	}

	// remove rlimsp mysql container
	rlimsMysqlContainerRemoveError := misc.RemoveContainer(ctx, dockerClient, constants.RLIMS_MYSQL_CONTAINER_NAME)
	if rlimsMysqlContainerRemoveError != nil {
		return rlimsMysqlContainerRemoveError
	}

	// remove rlimsp network network
	networkRemoveError := misc.RemoveNetwork(ctx, dockerClient, constants.RLIMS_NETWORK_NAME)
	if networkRemoveError != nil {
		return networkRemoveError
	}

	return nil

}
