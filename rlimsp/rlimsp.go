package rlimsp

import (
	"context"
	"itextmine/misc"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func Execute(workDir string, numParallelTasks int) error {
	dockerClient := misc.CreateDockerClient()

	ctx := context.Background()

	// remove rlimsp network before starting
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
	rlimsMySQLContainerID, rlimspMysqlStartError := startRLIMSPMySQLContainer(ctx, dockerClient)
	if rlimspMysqlStartError != nil {
		return rlimspMysqlStartError
	}

	// remove this container when we are done
	defer dockerClient.ContainerRemove(ctx, rlimsMySQLContainerID, types.ContainerRemoveOptions{Force: true})

	// start rlimsp dockers

	return nil
}

func startRLIMSPContainer(ctx context.Context, dockerClient *client.Client) error {
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

func startRLIMSPMySQLContainer(ctx context.Context, dockerClient *client.Client) (string, error) {

	// pull the image
	pullError := misc.PullImage(ctx, dockerClient, "itextmine/rlimsp-mysql")
	if pullError != nil {
		return "", pullError
	}

	// create the container
	rlimsMySQLNetworkConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"rlimsp": &network.EndpointSettings{
				IPAddress: "10.0.0.2",
			},
		},
	}
	containerCreateResponse, containerCreateError := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: "itextmine/rlimsp-mysql",
	}, nil, &rlimsMySQLNetworkConfig, "rlimsp-mysql")

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
