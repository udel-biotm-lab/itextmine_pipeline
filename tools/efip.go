package tools

import (
	"context"
	"errors"
	"fmt"
	"itextmine/misc"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func ExecuteEfipContainer(ctx context.Context, dockerClient *client.Client, taskName string, workdir string) error {

	rlimspTaskInputAbsolutePath, rlismpInputPathError := filepath.Abs(path.Join(workdir, "rlimsp", taskName, "input.json"))
	if rlismpInputPathError != nil {
		return rlismpInputPathError
	}

	taskInputAbsolutePath, inputPathError := filepath.Abs(path.Join(workdir, "rlimsp", taskName, "output.txt"))
	if inputPathError != nil {
		return inputPathError
	}

	taskOutputJsonAbsolutePath, jsonOutputPathError := filepath.Abs(path.Join(workdir, "rlimsp", taskName, "efip_output.json"))
	if jsonOutputPathError != nil {
		return jsonOutputPathError
	}

	// create the output.json
	jsonFile, jsonCreateError := os.Create(taskOutputJsonAbsolutePath)
	if jsonCreateError != nil {
		return jsonCreateError
	}
	jsonFile.Close()

	// host config
	hostConfig := container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s:ro", taskInputAbsolutePath, "/efip_workdir/docs.rlims.txt"),
			fmt.Sprintf("%s:%s", taskOutputJsonAbsolutePath, "/efip_workdir/docs.json"),
		},
	}

	// container config
	containerConfig := container.Config{
		Image: "leebird/efip",
	}

	// create the container
	containerCreateResponse, containerCreateError := dockerClient.ContainerCreate(ctx,
		&containerConfig,
		&hostConfig,
		nil,
		fmt.Sprintf("rlimsp-efip-%s", taskName))

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
		// No output being present is not an an error. The tool might not find anything in this set of docs
		log.Println(fmt.Sprintf("WARN: %s", checkoutputErr.Error()))
	} else {
		// create the absolute path for align output
		alignOutputAbsolutePath, alignOutputAbsolutePathError := filepath.Abs(path.Join(workdir, "rlimsp", taskName, "efip_align.json"))
		if alignOutputAbsolutePathError != nil {
			return alignOutputAbsolutePathError
		}

		// run alignment
		alignError := ExecuteAlign(ctx, dockerClient, taskName, rlimspTaskInputAbsolutePath, taskOutputJsonAbsolutePath, alignOutputAbsolutePath, "efip")
		if alignError != nil {
			return alignError
		}
	}

	return nil
}

func ReduceEfip(toolWorkDir string, toolOutputDir string, collectionType string) error {

	// build output reduce json file paths
	outputFilePath := fmt.Sprintf("%s/efip.%s.output.json", toolOutputDir, collectionType)
	reduceOutputCmdStr := fmt.Sprintf("cat %s/*/efip_output.json > %s", toolWorkDir, outputFilePath)

	log.Println(fmt.Sprintf("Reducing EFIP output results to : %s", outputFilePath))

	// execute the command
	reduceOutputCmdErr, _, reduceOuputCmdErrOut := misc.Shellout(reduceOutputCmdStr)
	if reduceOutputCmdErr != nil {
		return errors.New(reduceOuputCmdErrOut)
	}

	// check reduce output
	outputReduceOutputCheckError := misc.CheckOutput(outputFilePath)
	if outputReduceOutputCheckError != nil {
		return outputReduceOutputCheckError
	}

	// build align reduce json
	alignOutputFilePath := fmt.Sprintf("%s/efip.%s.aligned.json", toolOutputDir, collectionType)
	reduceAlignCmdStr := fmt.Sprintf("cat %s/*/efip_align.json > %s", toolWorkDir, alignOutputFilePath)

	log.Println(fmt.Sprintf("Reducing EFIP Align results to : %s", alignOutputFilePath))

	// execute the command
	reduceAlignCmdErr, _, reduceAlignCmdErrOut := misc.Shellout(reduceAlignCmdStr)
	if reduceAlignCmdErr != nil {
		return errors.New(reduceAlignCmdErrOut)
	}

	// check reduce output
	alignReduceOutputCheckError := misc.CheckOutput(alignOutputFilePath)
	if alignReduceOutputCheckError != nil {
		return alignReduceOutputCheckError
	}

	return nil

}
