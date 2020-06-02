package tools

import (
	"context"
	"errors"
	"fmt"
	"itextmine/misc"
	"os"
	"path"
	"path/filepath"

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
	originalJsonPathExists, originalJsonPathError := misc.PathExists(originalJsonPath)
	if originalJsonPathExists == false {
		return originalJsonPathError
	}

	// check if tool output json exists
	toolOuputJsonPathExists, toolOuputJsonPathError := misc.PathExists(toolOutputJsonPath)
	if toolOuputJsonPathExists == false {
		return toolOuputJsonPathError
	}

	// touch the alignJson File
	touchError := misc.TouchFile(alignedJsonPath)
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
	checkoutputErr := misc.CheckOutput(alignedJsonPath)
	if checkoutputErr != nil {
		return checkoutputErr
	}

	return nil
}

func Reduce(workDir string, outputDir string, toolName string, collectionType string) error {
	// build path to final workdir
	toolWorkDir, toolWorkDirErr := filepath.Abs(path.Join(workDir, toolName))
	if toolWorkDirErr != nil {
		return toolWorkDirErr
	}

	// check if this path already exist
	toolWorkDirExists, toolWorkDirExistsError := misc.PathExists(toolWorkDir)
	if toolWorkDirExists == false {
		return toolWorkDirExistsError
	}

	// build path to final output dir
	toolOutputDir, toolOutputDirErr := filepath.Abs(path.Join(outputDir, toolName))
	if toolOutputDirErr != nil {
		return toolOutputDirErr
	}

	// check if tool output dir exists
	outputDirExists, _ := misc.PathExists(toolOutputDir)

	// if does not exist, create it
	if outputDirExists == false {
		mkDirErr := os.MkdirAll(toolOutputDir, os.FileMode(0777))
		if mkDirErr != nil {
			return mkDirErr
		}
	}

	if toolName == "rlimsp" {
		// reduce rlimsp
		rlimsPpReduceError := ReduceRlimsp(toolWorkDir, toolOutputDir, collectionType)
		if rlimsPpReduceError != nil {
			return rlimsPpReduceError
		}

		// reduce efip
		efipReduceError := ReduceEfip(toolWorkDir, toolOutputDir, collectionType)
		if efipReduceError != nil {
			return efipReduceError
		}

	} else {
		return errors.New(fmt.Sprintf("Unknown tool %s", toolName))
	}

	return nil
}
