package misc

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func ExecuteAlign(ctx context.Context,
	dockerClient *client.Client,
	taskName string,
	originalJsonPath string,
	toolOutputJsonPath string,
	alignedJsonPath string) error {

	// check original json exists
	originalJsonPathExists, originalJsonPathError := PathExists(originalJsonPath)
	if originalJsonPathExists == false {
		return originalJsonPathError
	}

	// check if tool output json exists
	toolOuputJsonPathExists, toolOuputJsonPathError := PathExists(toolOutputJsonPath)
	if toolOuputJsonPathExists == false {
		return toolOuputJsonPathError
	}

	// touch the alignJson File
	touchError := TouchFile(alignedJsonPath)
	if touchError != nil {
		return touchError
	}

	// create bind mounts
	hostConfig := container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", originalJsonPath, "/align_workdir/origin_file.json"),
			fmt.Sprintf("%s:%s", toolOutputJsonPath, "/align_workdir/result_file.json"),
			fmt.Sprintf("%s:%s", alignedJsonPath, "/align_workdir/output_file.json"),
		},
	}

	// container config
	containerConfig := container.Config{
		Image: "itextmine/align",
	}

	// create the container
	containerCreateResponse, containerCreateError := dockerClient.ContainerCreate(ctx,
		&containerConfig,
		&hostConfig,
		nil,
		fmt.Sprintf("rlimsp-align-%s", taskName),
	)

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
	checkoutputErr := CheckOutput(alignedJsonPath)
	if checkoutputErr != nil {
		return checkoutputErr
	}

	return nil
}
