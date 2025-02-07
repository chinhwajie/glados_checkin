package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type CheckinResponse struct {
	Message string `json:"message"`
}

type StatusResponse struct {
	Data struct {
		LeftDays string `json:"leftDays"`
	} `json:"data"`
}

func glados(ctx context.Context) ([]string, error) {
	cookie := os.Getenv("GLADOS")
	if cookie == "" {
		return nil, fmt.Errorf("GLADOS cookie not found")
	}

	headers := map[string]string{
		"cookie":       cookie,
		"referer":      "https://glados.rocks/console/checkin",
		"user-agent":   "Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 6.0)",
		"content-type": "application/json",
	}

	checkinReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://glados.rocks/api/user/checkin", strings.NewReader(`{"token":"glados.one"}`))
	if err != nil {
		return nil, fmt.Errorf("creating checkin request: %w", err)
	}
	for k, v := range headers {
		checkinReq.Header.Set(k, v)
	}

	checkinResp, err := http.DefaultClient.Do(checkinReq)
	if err != nil {
		return nil, fmt.Errorf("performing checkin request: %w", err)
	}
	defer checkinResp.Body.Close()

	var checkinData CheckinResponse
	if err := json.NewDecoder(checkinResp.Body).Decode(&checkinData); err != nil {
		return nil, fmt.Errorf("decoding checkin response: %w", err)
	}

	statusReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://glados.rocks/api/user/status", nil)
	if err != nil {
		return nil, fmt.Errorf("creating status request: %w", err)
	}
	for k, v := range headers {
		statusReq.Header.Set(k, v)
	}

	statusResp, err := http.DefaultClient.Do(statusReq)
	if err != nil {
		return nil, fmt.Errorf("performing status request: %w", err)
	}
	defer statusResp.Body.Close()

	var statusData StatusResponse
	if err := json.NewDecoder(statusResp.Body).Decode(&statusData); err != nil {
		return nil, fmt.Errorf("decoding status response: %w", err)
	}

	return []string{
		"Checking OK",
		checkinData.Message,
		fmt.Sprintf("Left Days %s", statusData.Data.LeftDays),
	}, nil
}

func notify(ctx context.Context, contents []string) error {
	clientName := os.Getenv("CLIENT_NAME")
	clientSecret := os.Getenv("CLIENT_SECRET")

	if clientName == "" || clientSecret == "" || contents == nil {
		return fmt.Errorf("missing client name, secret, or contents")
	}

	message := "**GLADOS Checkin**\\n"
	for i, content := range contents {
		message += content
		if i < len(contents)-1 {
			message += "\\n"
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://discordbot.lumisnap.im/send-message"), strings.NewReader(`{"content":"`+message+`"}`))
	if err != nil {
		return fmt.Errorf("creating notify request: %w", err)
	}
	req.Header.Set("ClientName", clientName)
	req.Header.Set("ClientSecret", clientSecret)

	_, err = http.DefaultClient.Do(req)
	return err
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Add a timeout
	defer cancel()

	contents, err := glados(ctx)
	if err != nil {
		fmt.Println("GlaDOS Error:", err)
		githubURL := fmt.Sprintf("<%s/%s>", os.Getenv("GITHUB_SERVER_URL"), os.Getenv("GITHUB_REPOSITORY"))
		contents = []string{"Checking Error", err.Error(), githubURL}
	}

	if err := notify(ctx, contents); err != nil {
		fmt.Println("Notify Error:", err)
	}
}
