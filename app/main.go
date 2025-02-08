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
	"sync"
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
	Service       string  `json:"service"`
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
		containerName := strings.TrimPrefix(container.Names[0], "/")
		containerList = append(containerList, struct {
			ID   string
			Name string
		}{
			ID:   containerID,
			Name: containerName,
		})
	}
	return containerList, nil
}

// пинг через доекровские cli команды
func pingService(containerID, serviceName string) PingResult {
	startTime := time.Now()
	thisID := getCurrentContainerID()

	cmd := exec.Command("docker", "exec", "-t", thisID, "ping", "-c", "1", serviceName)
	err := cmd.Run()
	duration := time.Since(startTime).Seconds()
	result := PingResult{
		ContainerID:  containerID,
		PingDuration: duration,
		Status:       "DOWN",
		Service:      serviceName,
	}
	if err != nil {
		log.Printf("Error pinging service %s from container %s: %v", serviceName, thisID, err)
		return result
	}
	result.Status = "UP"
	return result
}

func sendPingResult(apiURL string, result PingResult) {
	jsonData, _ := json.Marshal(result)
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to send ping result for %s: %v\n", result.Service, err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("Sent ping result for %s: %s\n %s\n", result.Service, result.Status, result.ContainerID)
}

func pingAllContainers(apiURL string, pingedContainers map[string]bool, mutex *sync.Mutex) {
	containers, err := discoverContainers()
	if err != nil {
		log.Printf("Failed to discover containers: %v\n", err)
		return
	}

	servicesToPing := []string{"api", "app", "postgres"}

	for _, container := range containers {
		for _, service := range servicesToPing {
			mutex.Lock()
			key := fmt.Sprintf("%s-%s", container.ID, service)
			if pingedContainers[key] {
				mutex.Unlock()
				continue
			}
			pingedContainers[key] = true
			mutex.Unlock()

			result := pingService(container.ID, service)
			result.ContainerName = container.Name
			sendPingResult(apiURL, result)
		}
	}
}

func main() {
	apiURL := "http://api:8080/ping_results"

	pingedContainers := make(map[string]bool)
	mutex := &sync.Mutex{}

	for {
		fmt.Println("Starting ping cycle...")
		pingAllContainers(apiURL, pingedContainers, mutex)

		mutex.Lock()
		for key := range pingedContainers {
			delete(pingedContainers, key)
		}
		mutex.Unlock()

		fmt.Println("Ping cycle completed. Waiting for 10 seconds before next cycle.")
		time.Sleep(10 * time.Second)
	}
}
