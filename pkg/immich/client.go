package immich

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type Client struct {
	client   *http.Client
	endPoint string
	apiKey   string
	deviceID string
}

type PingResponse struct {
	Res string `json:"res"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type AssetUploadResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ServerResponse struct {
	Message []string `json:"message"`
}

func NewClient(endpoint string, apiKey string) (*Client, error) {
	var err error
	deviceID, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &Client{
		endPoint: endpoint + "/api",
		apiKey:   apiKey,
		deviceID: deviceID,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}, nil
}

// Ping server
func (c *Client) PingServer(ctx context.Context) (*PingResponse, error) {
	req, err := http.NewRequest("GET", c.endPoint+"/server/ping", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	pr := PingResponse{}
	if err := parseResponse(resp, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func (c *Client) GetCurrentUser(ctx context.Context) (*UserResponse, error) {
	req, err := http.NewRequest("GET", c.endPoint+"/users/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	ur := UserResponse{}
	if err := parseResponse(resp, &ur); err != nil {
		return nil, err
	}

	return &ur, nil
}

// Upload Asset
func (c *Client) UploadAsset(ctx context.Context, filename string, createdAt time.Time, modifiedAt time.Time) (*AssetUploadResponse, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Get file info
	fileinfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Create a buffer to hold the multipart data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add the file to the multipart writer
	part, err := writer.CreateFormFile("assetData", fileinfo.Name())
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	err = writer.WriteField("deviceAssetId", fmt.Sprintf("%s-%d", fileinfo.Name(), fileinfo.Size()))
	if err != nil {
		return nil, err
	}

	err = writer.WriteField("deviceId", c.deviceID)
	if err != nil {
		return nil, err
	}

	err = writer.WriteField("fileCreatedAt", createdAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	err = writer.WriteField("fileModifiedAt", modifiedAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	// Close the multipart writer to finalize the content
	writer.Close()

	req, err := http.NewRequest("POST", c.endPoint+"/assets", &body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	aur := AssetUploadResponse{}
	if err := parseResponse(resp, &aur); err != nil {
		return nil, err
	}

	return &aur, nil
}
