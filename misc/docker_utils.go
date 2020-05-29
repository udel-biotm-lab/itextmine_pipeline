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

func PullImage(ctx context.Context, dockerClient *client.Client, imageName string) error {
	reader, pullError := dockerClient.ImagePull(ctx, fmt.Sprintf("docker.io/%s", imageName), types.ImagePullOptions{All: true})
	if pullError != nil {
		return pullError
	}

	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(reader, os.Stderr, termFd, isTerm, nil)

	return nil
}

func StopAndRemoveContainers(ctx context.Context, dockerClient *client.Client, _ string) error {

	return nil
}
