package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type PingResult struct {
	Service            string    `json:"service"`
	IPAddress          string    `json:"ip_address"`
	PingDuration       float64   `json:"ping_duration"`
	Status             string    `json:"status"`
	LastSuccessfulPing time.Time `json:"last_successful_ping"`
}

func extractIPAddress(url string) string {
	parts := strings.Split(url, "//")
	if len(parts) > 1 {
		host := strings.Split(parts[1], ":")[0]
		return host
	}
	return "unknown"
}

func ping(url string, serviceName string) PingResult {
	startTime := time.Now()
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	duration := time.Since(startTime).Seconds()

	result := PingResult{
		Service:      serviceName,
		IPAddress:    extractIPAddress(url),
		PingDuration: duration,
		Status:       "DOWN",
	}

	if err != nil {
		log.Printf("Error pinging %s: %v", url, err)
		return result
	}
	defer resp.Body.Close()

	log.Printf("Response from %s: %d", url, resp.StatusCode)

	if resp.StatusCode == http.StatusOK {
		result.Status = "UP"
		result.LastSuccessfulPing = time.Now()
	} else {
		log.Printf("Unexpected status code from %s: %d", url, resp.StatusCode)
	}

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
	fmt.Printf("Sent ping result for %s: %s\n", result.Service, result.Status)
}

// Используем докерскую библиотеку для поиска контейнеров в нашей сети (от docker-compose)
func discoverContainers() ([]struct {
	URL         string
	ServiceName string
}, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var services []struct {
		URL         string
		ServiceName string
	}

	for _, container := range containers {
		containerName := strings.TrimPrefix(container.Names[0], "/")
		//vk_default хард-кодим, она автоматом создается, рутпроекта_дефолт*
		networks, err := cli.NetworkInspect(ctx, "vk_default", network.InspectOptions{})
		if err != nil {
			fmt.Printf("Failed to inspect network for container %s: %v\n", containerName, err)
			continue
		}
		found := false
		for _, network := range networks.Containers {
			if network.Name == containerName {
				ipAddress := network.IPv4Address
				if ipAddress == "" {
					fmt.Printf("No IP address found for container %s\n", containerName)
					continue
				}
				// Убираем сaбнет маску
				ipParts := strings.Split(ipAddress, "/")
				if len(ipParts) == 0 {
					fmt.Printf("Invalid IP address format for container %s: %s\n", containerName, ipAddress)
					continue
				}
				ipAddress = ipParts[0]
				// Раскидываем порты
				port := ""
				switch containerName {
				case "vk_api_1":
					port = "8080"
				case "vk_app_1":
					port = "8081"
				default:
					fmt.Printf("Unknown service: %s\n", containerName)
					continue
				}
				// URL
				url := fmt.Sprintf("http://%s:%s", ipAddress, port)

				// Для дебага
				fmt.Printf("Нашли: %s at:  %s\n", containerName, url)
				services = append(services, struct {
					URL         string
					ServiceName string
				}{
					URL:         url,
					ServiceName: containerName,
				})
				found = true
			}
		}
		if !found {
			fmt.Printf("Container %s not found in your network\n", containerName)
		}
	}

	return services, nil
}

func pingAllServices(apiURL string) {
	services, err := discoverContainers()
	if err != nil {
		fmt.Printf("Failed to discover containers: %v\n", err)
		return
	}

	for _, service := range services {
		result := ping(service.URL, service.ServiceName)

		sendPingResult(apiURL, result)
	}
}

func main() {
	apiURL := "http://api:8080/ping_results"
	// Цикл бесконечного пинга
	for {
		// result := ping("http://api:8080", "api")
		// fmt.Printf("Ping result: %+v\n", result)
		fmt.Println("Starting ping cycle...")
		pingAllServices(apiURL)
		fmt.Println("Ping cycle completed. Waiting for 5 seconds before next cycle.")
		time.Sleep(10 * time.Second)
	}
}
