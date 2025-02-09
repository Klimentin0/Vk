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

// ID всех контейнеров в vk_default сетке.
func discoverContainers() ([]struct {
	ID string
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
		ID string
	}
	for _, container := range containers {
		containerID := container.ID
		// ТОЛЬКО ПЕРВЫЕ 12 символов айди контейнра позволяют отправить пингу
		shortID := containerID[:12]
		containerList = append(containerList, struct {
			ID string
		}{
			ID: shortID,
		})
	}
	return containerList, nil
}

// пинг через доекровские cli команды
func pingService(containerID string) PingResult {
	thisID := getCurrentContainerID()
	result := PingResult{
		ContainerName: "unknown",
		ContainerID:   containerID,
		PingDuration:  0,
		Status:        "DOWN",
	}

	// Получаем имя контейнера
	cmd2 := exec.Command("docker", "inspect", "--format", "{{.Name}}", containerID)
	output, err := cmd2.Output()
	if err != nil {
		log.Printf("Failed to inspect container %s: %v", containerID, err)
	} else {
		// Причёсываем имя
		containerName := strings.TrimSpace(string(output))
		containerName = strings.TrimPrefix(containerName, "/") // Убираем слэши
		result.ContainerName = containerName
	}

	//Пингуем
	startTime := time.Now()
	cmd := exec.Command("docker", "exec", thisID, "ping", "-c", "1", containerID)
	err = cmd.Run()
	duration := time.Since(startTime).Seconds()
	result.PingDuration = duration
	if err != nil {
		log.Printf("Error pinging container %s: %v", containerID, err)
		return result
	}
	result.Status = "UP"
	return result
}

func sendPingResult(apiURL string, result PingResult) {
	//  JSON
	jsonData, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("Failed to serialize ping result for %s: %v\n", result.ContainerID, err)
		return
	}

	//  HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// создаём HTTP POST req
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to create HTTP request for %s: %v\n", result.ContainerID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправка запроса
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send ping result for %s: %v\n", result.ContainerID, err)
		return
	}
	defer resp.Body.Close()

	// Проверяем код
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("API returned non-2xx status code: %d for container %s\n", resp.StatusCode, result.ContainerID)
		return
	}

	// Логиирование для проверки
	fmt.Printf("Sent ping result for %s: ID(%s)\n Status: %s\n with duration: %f\n", result.ContainerName, result.ContainerID, result.Status, result.PingDuration)
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
	apiURL := "http://api:8080/ping-results"

	for {
		fmt.Println("Начинаю пинг...")
		time.Sleep(5 * time.Second)
		pingAllContainers(apiURL)
		time.Sleep(5 * time.Second)
	}
}
