package misc

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
)

func RemoveNetwork(ctx context.Context, dockerClient *client.Client, networkName string) error {
	// check if network name is not empty
	if len(networkName) == 0 {
		return errors.New("Network name cannot be empty")
	}

	filterArgs, filterArgsError := filters.ParseFlag(fmt.Sprintf("name=%s", networkName), filters.NewArgs())
	if filterArgsError != nil {
		return filterArgsError
	}

	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{
		Filters: filterArgs,
	})

	if err != nil {
		return err
	}

	// loop over all these networks and remove them
	for networkIndex := range networks {
		network := networks[networkIndex]
		networkID := network.ID
		networkRemoveError := dockerClient.NetworkRemove(ctx, networkID)
		if networkRemoveError != nil {
			return networkRemoveError
		}
	}

	return nil
}

func RemoveContainer(ctx context.Context, dockerClient *client.Client, containerName string) error {
	// check if container name is not empty
	if len(containerName) == 0 {
		return errors.New("Container name cannot be empty")
	}

	filterArgs, filterArgsError := filters.ParseFlag(fmt.Sprintf("name=%s", containerName), filters.NewArgs())
	if filterArgsError != nil {
		return filterArgsError
	}

	containers, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})

	if err != nil {
		return err
	}

	// loop over all these networks and remove them
	for containerIndex := range containers {
		container := containers[containerIndex]

		// remove the container
		removeError := dockerClient.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
			Force: true,
		})

		if removeError != nil {
			return removeError
		}

	}

	return nil

}

func CheckIfNetworkExists(ctx context.Context, dockerClient *client.Client, networkName string) (bool, string, error) {
	// check if network name is not empty
	if len(networkName) == 0 {
		return false, "", errors.New("Network name cannot be empty")
	}

	filterArgs, filterArgsError := filters.ParseFlag(fmt.Sprintf("name=%s", networkName), filters.NewArgs())
	if filterArgsError != nil {
		return false, "", filterArgsError
	}

	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{
		Filters: filterArgs,
	})

	if err != nil {
		return false, "", err
	}

	// loop over all these networks and remove them
	if len(networks) > 0 {
		network := networks[0]
		networkID := network.ID
		return true, networkID, nil
	} else {
		return false, "", nil
	}
}

func PullImage(ctx context.Context, dockerClient *client.Client, imageName string) error {
	reader, pullError := dockerClient.ImagePull(ctx, fmt.Sprintf("docker.io/%s", imageName), types.ImagePullOptions{All: true})
	if pullError != nil {
		return pullError
	}

	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(reader, os.Stderr, termFd, isTerm, nil)

	return nil
}
