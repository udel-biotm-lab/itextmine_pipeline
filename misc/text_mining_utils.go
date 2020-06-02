package misc

import (
	"context"
	"errors"
	"fmt"
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

func Reduce(workDir string, outputDir string, toolName string, collectionType string) error {
	// build path to final workdir
	toolWorkDir, toolWorkDirErr := filepath.Abs(path.Join(workDir, toolName))
	if toolWorkDirErr != nil {
		return toolWorkDirErr
	}

	// check if this path already exist
	toolWorkDirExists, toolWorkDirExistsError := PathExists(toolWorkDir)
	if toolWorkDirExists == false {
		return toolWorkDirExistsError
	}

	// build path to final output dir
	toolOutputDir, toolOutputDirErr := filepath.Abs(path.Join(outputDir, toolName))
	if toolOutputDirErr != nil {
		return toolOutputDirErr
	}

	// check if tool output dir exists
	outputDirExists, _ := PathExists(toolOutputDir)

	// if does not exist, create it
	if outputDirExists == false {
		mkDirErr := os.MkdirAll(toolOutputDir, os.FileMode(0777))
		if mkDirErr != nil {
			return mkDirErr
		}
	}

	if toolName == "rlimsp" {
		return reduceRlimsp(toolWorkDir, toolOutputDir, collectionType)
	} else {
		return errors.New(fmt.Sprintf("Unknown tool %s", toolName))
	}

	return nil
}

func reduceRlimsp(toolWorkDir string, toolOutputDir string, collectionType string) error {
	// build reduce align json
	alignOutputFilePath := fmt.Sprintf("%s/rlimsp.%s.align.json", toolOutputDir, collectionType)
	reduceAlignCmdStr := fmt.Sprintf("cat %s/*/align.json > %s", toolWorkDir, alignOutputFilePath)

	// execute the command
	reduceAlignCmdErr, _, reduceAlignCmdErrOut := Shellout(reduceAlignCmdStr)
	if reduceAlignCmdErr != nil {
		return errors.New(reduceAlignCmdErrOut)
	}

	// check reduce output
	reduceOutputCheckError := CheckOutput(alignOutputFilePath)
	if reduceOutputCheckError != nil {
		return reduceOutputCheckError
	}

	// create a folder to store efip inputs
	eFipInputFolderName := fmt.Sprintf("%s/efip_%s_input", toolOutputDir, collectionType)
	eFipFolderCreateError := CreateFolderIfNotExists(eFipInputFolderName)
	if eFipFolderCreateError != nil {
		return eFipFolderCreateError
	}

	// get a list of all subfolders in tool workdir
	subDirNames, subDirNamesErr := GetSubDirNames(toolWorkDir)
	if subDirNamesErr != nil {
		return subDirNamesErr
	}

	// loop and copy all the output.txt to efip folder
	for index := range *subDirNames {
		subDir := (*subDirNames)[index]
		originalOutputTxtPath, originalOutputTxtPathErr := filepath.Abs(path.Join(toolWorkDir, subDir, "output.txt"))
		if originalOutputTxtPathErr != nil {
			return originalOutputTxtPathErr
		}

		destFolderPath, destFolderPathErr := filepath.Abs(path.Join(eFipInputFolderName, subDir))
		if destFolderPathErr != nil {
			return destFolderPathErr
		}

		// create the destination folder
		destFolderCreateErr := CreateFolderIfNotExists(destFolderPath)
		if destFolderCreateErr != nil {
			return destFolderCreateErr
		}

		// location to destination output.txt
		destOutputTxtPath, destOutputTxtPathErr := filepath.Abs(path.Join(destFolderPath, "output.txt"))
		if destOutputTxtPathErr != nil {
			return destOutputTxtPathErr
		}

		// move to new location
		moveErr := os.Rename(originalOutputTxtPath, destOutputTxtPath)
		if moveErr != nil {
			return moveErr
		}

	}

	return nil

}
