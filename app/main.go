package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type PingResult struct {
	ContainerID   string  `json:"container_id"`
	ContainerName string  `json:"container_name"`
	PingDuration  float64 `json:"ping_duration"`
	Status        string  `json:"status"`
}

// ID этого контейнера
func getCurrentContainerID() string {
	return os.Getenv("HOSTNAME")
}

// ID всех остальных контейнеров в vk_default сетке.
func discoverContainers() ([]struct {
	ID   string
	Name string
}, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// фильтр по сетке
	filter := filters.NewArgs()
	filter.Add("network", "vk_default")

	// контейнеры по фильтру
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containerList []struct {
		ID   string
		Name string
	}
	for _, container := range containers {
		containerID := container.ID
		// ТОЛЬКО ПЕРВЫЕ 12 символов айди контейнра позволяют отправить пингу
		shortID := containerID[:12]
		containerName := strings.TrimPrefix(container.Names[0], "/")
		containerList = append(containerList, struct {
			ID   string
			Name string
		}{
			ID:   shortID,
			Name: containerName,
		})
	}
	return containerList, nil
}

// пинг через доекровские cli команды
func pingService(containerID string) PingResult {
	startTime := time.Now()
	thisID := getCurrentContainerID()

	cmd := exec.Command("docker", "exec", thisID, "ping", "-c", "1", containerID)
	err := cmd.Run()
	duration := time.Since(startTime).Seconds()
	result := PingResult{
		ContainerID:  containerID,
		PingDuration: duration,
		Status:       "DOWN",
	}
	if err != nil {
		log.Printf("Error pinging container %s: %v", containerID, err)
		return result
	}
	result.Status = "UP"
	return result
}

func sendPingResult(apiURL string, result PingResult) {
	jsonData, _ := json.Marshal(result)
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to send ping result for %s: %v\n", result.ContainerID, err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("Sent ping result for %s: ID(%s)\n %s\n", result.ContainerName, result.ContainerID, result.Status)
}

func pingAllContainers(apiURL string) {
	var result PingResult
	containers, err := discoverContainers()
	if err != nil {
		log.Printf("Failed to discover containers: %v\n", err)
		return
	}

	for _, container := range containers {
		result = pingService(container.ID)
		sendPingResult(apiURL, result)
	}
}

func main() {
	apiURL := "http://api:8080/ping_results"

	for {
		fmt.Println("Начинаю пинг...")
		pingAllContainers(apiURL)
		time.Sleep(10 * time.Second)
	}
}
