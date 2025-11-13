package data

import (
	"fmt"

	"github.com/moby/moby/client"
)

func getDockerClient(sock string) (*client.Client, error) {
	apiClient, err := client.New(client.WithHost(fmt.Sprintf("unix://%s", sock)), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return apiClient, nil
}
