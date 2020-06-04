package tools

import (
	"context"
	"fmt"
	"itextmine/constants"
	"itextmine/misc"
	"log"
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

func ExecuteEGard(workDir string, numParallelTasks int) error {
	dockerClient := misc.CreateDockerClient()
	ctx := context.Background()

	cleanupError := cleanUpEgard(ctx, dockerClient)
	if cleanupError != nil {
		return cleanupError
	}

	// create egard rlimsp mysql network
	log.Println(fmt.Sprintf("Creating %s network", constants.EGARD_RLIMSP_NETWORK_NAME))
	networkID, networkCreateError := createEgardRlimspNetwork(ctx, dockerClient)
	if networkCreateError != nil {
		return networkCreateError
	}

	// start the rlimsp mysql container
	log.Println(fmt.Sprintf("Creating %s container", constants.RLIMS_MYSQL_CONTAINER_NAME))
	rlimspMySQLContainerID, rlimspMysqlStartError := startEgardRLIMSPMySQLContainer(ctx, dockerClient)
	if rlimspMysqlStartError != nil {
		return rlimspMysqlStartError
	}

	// defer remove network when we are done
	defer dockerClient.NetworkRemove(ctx, networkID)

	// defer remove rlimsp mysql container when we are done
	defer dockerClient.ContainerRemove(ctx, rlimspMySQLContainerID, types.ContainerRemoveOptions{Force: true})

	// create mace2k network
	mace2kNetworkID, mace2kNetworkError := createMace2KNetwork(ctx, dockerClient)
	if mace2kNetworkError != nil {
		return mace2kNetworkError
	}

	// start the mace2k mysql container
	log.Println(fmt.Sprintf("Creating %s container", constants.MACE2K_MYSQL_CONTAINER))
	mace2kMysqlContainerID, mace2KmysqlError := startMace2KMySQLContainer(ctx, dockerClient)
	if mace2KmysqlError != nil {
		return mace2KmysqlError
	}

	// remove network when we are done
	defer dockerClient.NetworkRemove(ctx, mace2kNetworkID)

	// defer remove rlimsp mysql container when we are done
	defer dockerClient.ContainerRemove(ctx, mace2kMysqlContainerID, types.ContainerRemoveOptions{Force: true})

	// pull bionex docker image
	bionexPullError := misc.PullImage(ctx, dockerClient, constants.BIONEX_IMAGE)
	if bionexPullError != nil {
		return bionexPullError
	}

	// get a list of all the tasks
	rlimsWorkDirPath := path.Join(workDir, "egard")
	log.Println(fmt.Sprintf("Generating tasks from : %s ", rlimsWorkDirPath))
	tasks, tasksError := misc.GetSubDirNames(rlimsWorkDirPath)
	if tasksError != nil {
		return tasksError
	}

	// create a worker pool and start the execution
	wp := workerpool.New(numParallelTasks)

	// number of tasks
	num_tasks := len(*tasks) // multiple by three as we execute bionex, m2g and egard dockers
	log.Println(fmt.Sprintf("Generated %d tasks", num_tasks))

	// make a buffered channel to receive errors in go routine
	errorChan := make(chan error, num_tasks)

	// make a buffered channel to receive progress
	progressChan := make(chan bool, num_tasks)

	// make a done channel to signal work completion
	terminateChan := make(chan bool)

	// start a goroutine to handle the messages from worker pool
	go HandleProgress(progressChan, terminateChan, num_tasks)

	log.Println(fmt.Sprintf("Starting the pool with %d workers", numParallelTasks))

	for _, task := range *tasks {
		taskCopy := task
		wp.Submit(func() {

			// execute bionex
			bionexError := executeBionexDocker(ctx, dockerClient, taskCopy, workDir)
			if bionexError != nil {
				// send the message to error channel and die early
				errorChan <- bionexError
				progressChan <- true
				return
			}

			//progressChan <- true

			// execute m2g
			//progressChan <- true

			// execute egard
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

func executeBionexDocker(ctx context.Context, dockerClient *client.Client, taskName string, workDir string) error {

	taskPath := path.Join(workDir, "egard", taskName)

	// create bionex subfolder for given task
	bionexWorkDirPath := path.Join(taskPath, "bionex")
	bionexWorkCreateError := misc.CreateFolderIfNotExists(bionexWorkDirPath)
	if bionexWorkCreateError != nil {
		return bionexWorkCreateError
	}

	taskInputAbsolutePath, inputPathError := filepath.Abs(path.Join(taskPath, "input.json"))
	if inputPathError != nil {
		return inputPathError
	}

	taskOutputJsonAbsolutePath, jsonOutputPathError := filepath.Abs(path.Join(bionexWorkDirPath, "output.json"))
	if jsonOutputPathError != nil {
		return jsonOutputPathError
	}
	// create the output.json
	jsonFile, jsonCreateError := os.Create(taskOutputJsonAbsolutePath)
	if jsonCreateError != nil {
		return jsonCreateError
	}
	jsonFile.Close()

	taskOutputTxtAbsolutePath, txtOutputPathError := filepath.Abs(path.Join(bionexWorkDirPath, "output.txt"))
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
			fmt.Sprintf("%s:%s:ro", taskInputAbsolutePath, "/bionex_workdir/in.json"),
			fmt.Sprintf("%s:%s", taskOutputJsonAbsolutePath, "/bionex_workdir/out.json"),
			fmt.Sprintf("%s:%s", taskOutputTxtAbsolutePath, "/bionex_workdir/out.txt"),
		},
	}

	// network config
	rlimspNetworkConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			constants.EGARD_RLIMSP_NETWORK_NAME: {},
		},
	}

	// container config
	containerConfig := container.Config{
		Image: constants.BIONEX_IMAGE,
	}

	// create the container
	containerCreateResponse, containerCreateError := dockerClient.ContainerCreate(ctx,
		&containerConfig,
		&hostConfig,
		&rlimspNetworkConfig,
		fmt.Sprintf("%s-%s", constants.EGARD_BIONEX_QUALIFIER, taskName))

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

	return nil
}

func createEgardRlimspNetwork(ctx context.Context, dockerClient *client.Client) (string, error) {
	networkOptions := types.NetworkCreate{
		CheckDuplicate: false,
		Driver:         "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet: constants.EGARD_RLIMSP_SUBNET,
				},
			},
		},
	}
	netWorkCreateResponse, err := dockerClient.NetworkCreate(ctx, constants.EGARD_RLIMSP_NETWORK_NAME, networkOptions)

	if err != nil {
		return "", err
	} else {
		return netWorkCreateResponse.ID, nil
	}
}

func startEgardRLIMSPMySQLContainer(ctx context.Context, dockerClient *client.Client) (string, error) {

	// pull the image
	pullError := misc.PullImage(ctx, dockerClient, constants.RLIMSP_MYSQL_IMAGE)
	if pullError != nil {
		return "", pullError
	}

	// create the container
	rlimsMySQLNetworkConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			constants.EGARD_RLIMSP_NETWORK_NAME: {
				IPAddress: constants.EGARD_RLIMSP_MYSQL_IP_ADDRESS,
			},
		},
	}
	containerCreateResponse, containerCreateError := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: constants.RLIMSP_MYSQL_IMAGE,
	}, nil, &rlimsMySQLNetworkConfig, constants.EGARD_RLIMS_MYSQL_CONTAINER_NAME)

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

func createMace2KNetwork(ctx context.Context, dockerClient *client.Client) (string, error) {
	networkOptions := types.NetworkCreate{
		CheckDuplicate: false,
		Driver:         "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet: constants.MACE2K_NETWORK_SUBNET,
				},
			},
		},
	}
	netWorkCreateResponse, err := dockerClient.NetworkCreate(ctx, constants.MACE2K_NETWORK, networkOptions)

	if err != nil {
		return "", err
	} else {
		return netWorkCreateResponse.ID, nil
	}
}

func startMace2KMySQLContainer(ctx context.Context, dockerClient *client.Client) (string, error) {

	// pull the image
	pullError := misc.PullImage(ctx, dockerClient, constants.MAC2K_MYSQL_IMAGE)
	if pullError != nil {
		return "", pullError
	}

	// create the container
	rlimsMySQLNetworkConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			constants.MACE2K_NETWORK: {
				IPAddress: constants.MACE2K_MYSQL_IP_ADDRESS,
			},
		},
	}

	containerCreateResponse, containerCreateError := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: constants.RLIMSP_MYSQL_IMAGE,
	}, nil, &rlimsMySQLNetworkConfig, constants.MACE2K_MYSQL_CONTAINER)

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

func cleanUpEgard(ctx context.Context, dockerClient *client.Client) error {

	// remove dangling bionex containers
	danglingBionixRemoveError := misc.RemoveContainer(ctx, dockerClient, fmt.Sprintf("%s*", constants.EGARD_BIONEX_QUALIFIER))
	if danglingBionixRemoveError != nil {
		return danglingBionixRemoveError
	}

	// remove rlimsp mysql container
	rlimsMysqlContainerRemoveError := misc.RemoveContainer(ctx, dockerClient, constants.EGARD_RLIMS_MYSQL_CONTAINER_NAME)
	if rlimsMysqlContainerRemoveError != nil {
		return rlimsMysqlContainerRemoveError
	}

	// remove rlimsp network network
	networkRemoveError := misc.RemoveNetwork(ctx, dockerClient, constants.EGARD_RLIMSP_NETWORK_NAME)
	if networkRemoveError != nil {
		return networkRemoveError
	}

	// remove mace2k network network
	mac2kNetworkRemoveError := misc.RemoveNetwork(ctx, dockerClient, constants.MACE2K_NETWORK)
	if mac2kNetworkRemoveError != nil {
		return mac2kNetworkRemoveError
	}

	return nil

}
