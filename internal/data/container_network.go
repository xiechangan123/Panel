package data

import (
	"context"
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/types"
)

type containerNetworkRepo struct{}

func NewContainerNetworkRepo() biz.ContainerNetworkRepo {
	return &containerNetworkRepo{}
}

// List 列出网络
func (r *containerNetworkRepo) List() ([]types.ContainerNetwork, error) {
	apiClient, err := getDockerClient("/var/run/docker.sock")
	if err != nil {
		return nil, err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	resp, err := apiClient.NetworkList(context.Background(), client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	var networks []types.ContainerNetwork
	for _, item := range resp.Items {
		ipamConfigs := make([]types.ContainerNetworkIPAMConfig, 0)
		for _, ipam := range item.IPAM.Config {
			ipamConfigs = append(ipamConfigs, types.ContainerNetworkIPAMConfig{
				Subnet:     ipam.Subnet,
				IPRange:    ipam.IPRange,
				Gateway:    ipam.Gateway,
				AuxAddress: ipam.AuxAddress,
			})
		}
		networks = append(networks, types.ContainerNetwork{
			ID:         item.ID,
			Name:       item.Name,
			Driver:     item.Driver,
			IPv6:       item.EnableIPv6,
			Internal:   item.Internal,
			Attachable: item.Attachable,
			Ingress:    item.Ingress,
			Scope:      item.Scope,
			CreatedAt:  item.Created,
			IPAM: types.ContainerNetworkIPAM{
				Driver:  item.IPAM.Driver,
				Options: types.MapToKV(item.IPAM.Options),
				Config:  ipamConfigs,
			},
			Options: types.MapToKV(item.Options),
			Labels:  types.MapToKV(item.Labels),
		})
	}

	slices.SortFunc(networks, func(a types.ContainerNetwork, b types.ContainerNetwork) int {
		return strings.Compare(a.Name, b.Name)
	})

	return networks, nil
}

// Create 创建网络
func (r *containerNetworkRepo) Create(req *request.ContainerNetworkCreate) (string, error) {
	apiClient, err := getDockerClient("/var/run/docker.sock")
	if err != nil {
		return "", err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	var ipamConfigs []network.IPAMConfig
	if req.Ipv4.Enabled {
		v4Subnet, err := netip.ParsePrefix(req.Ipv4.Subnet)
		if err != nil {
			return "", fmt.Errorf("invalid ipv4 subnet: %w", err)
		}
		v4Gateway, err := netip.ParseAddr(req.Ipv4.Gateway)
		if err != nil {
			return "", fmt.Errorf("invalid ipv4 gateway: %w", err)
		}
		v4IPRange, err := netip.ParsePrefix(req.Ipv4.IPRange)
		if err != nil {
			return "", fmt.Errorf("invalid ipv4 ip range: %w", err)
		}
		ipamConfigs = append(ipamConfigs, network.IPAMConfig{
			Subnet:  v4Subnet,
			Gateway: v4Gateway,
			IPRange: v4IPRange,
		})
	}
	if req.Ipv6.Enabled {
		v6Subnet, err := netip.ParsePrefix(req.Ipv6.Subnet)
		if err != nil {
			return "", fmt.Errorf("invalid ipv6 subnet: %w", err)
		}
		v6Gateway, err := netip.ParseAddr(req.Ipv6.Gateway)
		if err != nil {
			return "", fmt.Errorf("invalid ipv6 gateway: %w", err)
		}
		v6IPRange, err := netip.ParsePrefix(req.Ipv6.IPRange)
		if err != nil {
			return "", fmt.Errorf("invalid ipv6 ip range: %w", err)
		}
		ipamConfigs = append(ipamConfigs, network.IPAMConfig{
			Subnet:  v6Subnet,
			Gateway: v6Gateway,
			IPRange: v6IPRange,
		})
	}

	options := client.NetworkCreateOptions{
		EnableIPv4: &req.Ipv4.Enabled,
		EnableIPv6: &req.Ipv6.Enabled,
		Driver:     req.Driver,
		Options:    types.KVToMap(req.Options),
		Labels:     types.KVToMap(req.Labels),
	}
	if len(ipamConfigs) > 0 {
		options.IPAM = &network.IPAM{
			Config: ipamConfigs,
		}
	}

	resp, err := apiClient.NetworkCreate(context.Background(), req.Name, options)
	if err != nil {
		return "", err
	}

	return resp.ID, err
}

// Remove 删除网络
func (r *containerNetworkRepo) Remove(id string) error {
	apiClient, err := getDockerClient("/var/run/docker.sock")
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.NetworkRemove(context.Background(), id, client.NetworkRemoveOptions{})
	return err
}

// Prune 清理未使用的网络
func (r *containerNetworkRepo) Prune() error {
	apiClient, err := getDockerClient("/var/run/docker.sock")
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.NetworkPrune(context.Background(), client.NetworkPruneOptions{})
	return err
}
