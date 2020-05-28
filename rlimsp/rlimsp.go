package rlimsp

import (
	"context"
	"itextmine/misc"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func Execute(workDir string, numParallelTasks int) error {
	dockerClient := misc.CreateDockerClient()

	ctx := context.Background()

	// remove rlimsp network
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
