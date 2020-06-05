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
	"github.com/gammazero/workerpool"
)

func ExecuteMirtex(workDir string, numParallelTasks int) error {
	dockerClient := misc.CreateDockerClient()

	ctx := context.Background()

	log.Println("Cleaning up docker env from previous run")
	// cleanup from previous run
	cleanupError := cleanUpMirtex(ctx, dockerClient)
	if cleanupError != nil {
		return cleanupError
	}

	// pull mirtex docker image
	mirtexPullError := misc.PullImage(ctx, dockerClient, "itextmine/mirtex")
	if mirtexPullError != nil {
		return mirtexPullError
	}

	// pull align docker image
	alignPullError := misc.PullImage(ctx, dockerClient, "itextmine/align")
	if alignPullError != nil {
		return alignPullError
	}

	// get a list of all the tasks
	mirtexWorkDirPath := path.Join(workDir, "mirtex")
	log.Println(fmt.Sprintf("Generating tasks from : %s ", mirtexWorkDirPath))
	tasks, tasksError := misc.GetSubDirNames(mirtexWorkDirPath)
	if tasksError != nil {
		return tasksError
	}

	// create a worker pool and start the execution
	wp := workerpool.New(numParallelTasks)

	// number of tasks
	num_tasks := len(*tasks)
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
			// execute mirtex container
			rlimsContainerError := executeMirtexContainer(ctx, dockerClient, taskCopy, workDir)
			if rlimsContainerError != nil {
				log.Println(fmt.Sprintf("ERROR: %s", rlimsContainerError.Error()))
				errorChan <- rlimsContainerError
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

func executeMirtexContainer(ctx context.Context, dockerClient *client.Client, taskName string, workdir string) error {

	taskInputAbsolutePath, inputPathError := filepath.Abs(path.Join(workdir, "mirtex", taskName, "input.json"))
	if inputPathError != nil {
		return inputPathError
	}

	taskOutputJsonAbsolutePath, jsonOutputPathError := filepath.Abs(path.Join(workdir, "mirtex", taskName, "output.json"))
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
			fmt.Sprintf("%s:%s:ro", taskInputAbsolutePath, "/mirtex_workdir/in.json"),
			fmt.Sprintf("%s:%s", taskOutputJsonAbsolutePath, "/mirtex_workdir/out.json"),
		},
	}

	// container config
	containerConfig := container.Config{
		Image: "itextmine/mirtex",
	}

	// create the container
	containerCreateResponse, containerCreateError := dockerClient.ContainerCreate(ctx,
		&containerConfig,
		&hostConfig,
		nil,
		fmt.Sprintf("mirtex-%s", taskName))

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
		alignOutputAbsolutePath, alignOutputAbsolutePathError := filepath.Abs(path.Join(workdir, "mirtex", taskName, "align.json"))
		if alignOutputAbsolutePathError != nil {
			return alignOutputAbsolutePathError
		}

		// run alignment
		alignError := ExecuteAlign(ctx, dockerClient, taskName, taskInputAbsolutePath, taskOutputJsonAbsolutePath, alignOutputAbsolutePath, "mirtex")
		if alignError != nil {
			return alignError
		}
	}
	return nil
}

func ReduceMirtex(toolWorkDir string, toolOutputDir string, collectionType string) error {

	// build output reduce json
	outputFilePath := fmt.Sprintf("%s/mirtex.%s.output.json", toolOutputDir, collectionType)
	reduceOutputCmdStr := fmt.Sprintf("cat %s/*/output.json > %s", toolWorkDir, outputFilePath)

	log.Println(fmt.Sprintf("Reducing Mirtex Output results to : %s", outputFilePath))

	// execute the command
	reduceOutputCmdErr, _, reduceOutputCmdErrOut := misc.Shellout(reduceOutputCmdStr)
	if reduceOutputCmdErr != nil {
		return errors.New(reduceOutputCmdErrOut)
	}

	// check reduce output
	outputReduceOutputCheckError := misc.CheckOutput(outputFilePath)
	if outputReduceOutputCheckError != nil {
		return outputReduceOutputCheckError
	}

	// build reduce align json
	alignOutputFilePath := fmt.Sprintf("%s/mirtex.%s.aligned.json", toolOutputDir, collectionType)
	reduceAlignCmdStr := fmt.Sprintf("cat %s/*/align.json > %s", toolWorkDir, alignOutputFilePath)

	log.Println(fmt.Sprintf("Reducing Mirtex align results to : %s", alignOutputFilePath))

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

func cleanUpMirtex(ctx context.Context, dockerClient *client.Client) error {
	// remove dangling mirtx containers
	danglingMirtexRemoveError := misc.RemoveContainer(ctx, dockerClient, "mirtex-task*")
	if danglingMirtexRemoveError != nil {
		return danglingMirtexRemoveError
	}

	// remove dangling mirtex align containers
	danglingMirtexAlignRemoveError := misc.RemoveContainer(ctx, dockerClient, "mirtex-align*")
	if danglingMirtexAlignRemoveError != nil {
		return danglingMirtexAlignRemoveError
	}

	return nil

}
