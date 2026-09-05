package topology

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	containertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type dockerAPI interface {
	ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error)
	NetworkList(context.Context, client.NetworkListOptions) (client.NetworkListResult, error)
}

type Handler struct {
	docker   dockerAPI
	hostName string
	logger   *slog.Logger
	now      func() time.Time
}

type Snapshot struct {
	GeneratedAt time.Time       `json:"generatedAt"`
	Host        HostNode        `json:"host"`
	Networks    []NetworkNode   `json:"networks"`
	Containers  []ContainerNode `json:"containers"`
}

type HostNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NetworkNode struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Driver       string   `json:"driver"`
	Scope        string   `json:"scope"`
	Internal     bool     `json:"internal"`
	ContainerIDs []string `json:"containerIds"`
}

type ContainerNode struct {
	ID         string          `json:"id"`
	ShortID    string          `json:"shortId"`
	Name       string          `json:"name"`
	Image      string          `json:"image"`
	State      string          `json:"state"`
	NetworkIDs []string        `json:"networkIds"`
	Ports      []PublishedPort `json:"ports"`
}

type PublishedPort struct {
	PrivatePort int    `json:"privatePort"`
	PublicPort  *int   `json:"publicPort"`
	Protocol    string `json:"protocol"`
}

func NewHandler(docker dockerAPI, hostName string, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	hostName = strings.TrimSpace(hostName)
	if hostName == "" {
		hostName = "homelab"
	}
	return &Handler{docker: docker, hostName: hostName, logger: logger, now: time.Now}
}

func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet || (request.URL.Path != "" && request.URL.Path != "/") {
		writer.Header().Set("Allow", http.MethodGet)
		writeProblem(writer, http.StatusMethodNotAllowed, "Method not allowed.", "This topology endpoint is read-only.")
		return
	}

	networkResult, err := handler.docker.NetworkList(request.Context(), client.NetworkListOptions{})
	if err != nil {
		handler.unavailable(writer, err)
		return
	}
	containerResult, err := handler.docker.ContainerList(request.Context(), client.ContainerListOptions{All: true})
	if err != nil {
		handler.unavailable(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, buildSnapshot(
		handler.hostName,
		handler.now().UTC(),
		networkResult,
		containerResult,
	))
}

func buildSnapshot(hostName string, generatedAt time.Time, networkResult client.NetworkListResult, containerResult client.ContainerListResult) Snapshot {
	networks := make(map[string]*NetworkNode, len(networkResult.Items))
	for _, item := range networkResult.Items {
		id := item.ID
		if id == "" {
			id = item.Name
		}
		networks[id] = &NetworkNode{
			ID: id, Name: item.Name, Driver: item.Driver, Scope: item.Scope,
			Internal: item.Internal, ContainerIDs: []string{},
		}
	}

	containers := make([]ContainerNode, 0, len(containerResult.Items))
	for _, item := range containerResult.Items {
		container := ContainerNode{
			ID: item.ID, ShortID: shortID(item.ID), Name: containerName(item),
			Image: item.Image, State: normalizedState(item.State), NetworkIDs: []string{},
			Ports: publishedPorts(item.Ports),
		}
		if item.NetworkSettings != nil {
			for networkName, settings := range item.NetworkSettings.Networks {
				id := ""
				if settings != nil {
					id = settings.NetworkID
				}
				if id == "" {
					id = findNetworkID(networks, networkName)
				}
				if id == "" {
					id = networkName
				}
				network, exists := networks[id]
				if !exists {
					network = &NetworkNode{ID: id, Name: networkName, ContainerIDs: []string{}}
					networks[id] = network
				}
				network.ContainerIDs = append(network.ContainerIDs, item.ID)
				container.NetworkIDs = append(container.NetworkIDs, id)
			}
		}
		sort.Strings(container.NetworkIDs)
		containers = append(containers, container)
	}

	networkList := make([]NetworkNode, 0, len(networks))
	for _, network := range networks {
		sort.Strings(network.ContainerIDs)
		networkList = append(networkList, *network)
	}
	sort.Slice(networkList, func(left, right int) bool {
		if strings.EqualFold(networkList[left].Name, networkList[right].Name) {
			return networkList[left].ID < networkList[right].ID
		}
		return strings.ToLower(networkList[left].Name) < strings.ToLower(networkList[right].Name)
	})
	sort.Slice(containers, func(left, right int) bool {
		if strings.EqualFold(containers[left].Name, containers[right].Name) {
			return containers[left].ID < containers[right].ID
		}
		return strings.ToLower(containers[left].Name) < strings.ToLower(containers[right].Name)
	})

	return Snapshot{
		GeneratedAt: generatedAt,
		Host:        HostNode{ID: "host", Name: hostName},
		Networks:    networkList,
		Containers:  containers,
	}
}

func findNetworkID(networks map[string]*NetworkNode, name string) string {
	for id, network := range networks {
		if network.Name == name {
			return id
		}
	}
	return ""
}

func containerName(item containertypes.Summary) string {
	if len(item.Names) > 0 {
		if name := strings.TrimLeft(item.Names[0], "/"); name != "" {
			return name
		}
	}
	return shortID(item.ID)
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func normalizedState(state containertypes.ContainerState) string {
	value := strings.TrimSpace(string(state))
	if value == "" {
		return "Unknown"
	}
	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}

func publishedPorts(items []containertypes.PortSummary) []PublishedPort {
	result := make([]PublishedPort, 0, len(items))
	for _, item := range items {
		var publicPort *int
		if item.PublicPort > 0 {
			value := int(item.PublicPort)
			publicPort = &value
		}
		result = append(result, PublishedPort{
			PrivatePort: int(item.PrivatePort), PublicPort: publicPort,
			Protocol: strings.ToLower(item.Type),
		})
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].PrivatePort != result[right].PrivatePort {
			return result[left].PrivatePort < result[right].PrivatePort
		}
		return result[left].Protocol < result[right].Protocol
	})
	return result
}

func (handler *Handler) unavailable(writer http.ResponseWriter, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	handler.logger.Warn("Docker topology request failed")
	writeProblem(writer, http.StatusServiceUnavailable, "Infrastructure unavailable.", "Docker topology is unavailable.")
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeProblem(writer http.ResponseWriter, status int, title, detail string) {
	writer.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"type": "about:blank", "title": title, "status": status, "detail": detail,
	})
}
