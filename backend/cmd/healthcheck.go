package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const healthcheckTimeout = 4 * time.Second

func checkServerAlive(baseURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), healthcheckTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/system/version", nil)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	_, _ = io.Copy(io.Discard, response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	return nil
}

func runHealthcheckCommand() {
	err := checkServerAlive("http://localhost" + serverAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("healthcheck ok")
	os.Exit(0)
}
