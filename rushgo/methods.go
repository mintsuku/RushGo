package rushgo

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/quic-go/quic-go/http3"
)

// Config struct for RushGo client settings
type Config struct {
	EnableHTTP2 bool
	EnableHTTP3 bool
	Timeout     time.Duration
}

// RushGo struct to encapsulate the http client and default headers
type RushGo struct {
	client         *http.Client
	defaultHeaders map[string]string
	userAgent      string // User-Agent header
}

// New initializes a new RushGo instance with optional configuration
func New(cfg *Config) *RushGo {
	if cfg == nil {
		cfg = &Config{
			EnableHTTP2: true,
			EnableHTTP3: false, // Default to false for backward compatibility
			Timeout:     30 * time.Second,
		}
	}

	var transport http.RoundTripper
	if cfg.EnableHTTP3 {
		// Use http3.RoundTripper for HTTP/3 support
		transport = &http3.RoundTripper{}
	} else {
		// Fallback to using http.Transport
		transport = &http.Transport{
			ForceAttemptHTTP2: cfg.EnableHTTP2,
		}
	}

	return &RushGo{
		client: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		defaultHeaders: make(map[string]string), // Initialize the map here
	}
}

func (rg *RushGo) WithTimeout(timeout time.Duration) *RushGo {
    rg.client.Timeout = timeout
    return rg
}

// WithHeaders sets default headers for the RushGo client
func (rg *RushGo) WithHeaders(headers map[string]string) *RushGo {
    for key, value := range headers {
        rg.defaultHeaders[key] = value
    }
    return rg
}

// WithCookies sets cookies for the RushGo client's default headers.
func (rg *RushGo) WithCookies(cookies map[string]string) *RushGo {
    cookieStrings := []string{}
    for name, value := range cookies {
        cookieStrings = append(cookieStrings, fmt.Sprintf("%s=%s", name, value))
    }
    rg.defaultHeaders["Cookie"] = strings.Join(cookieStrings, "; ")
    return rg
}


// Get makes a GET request using the RushGo client
func (rg *RushGo) Get(url string) (*http.Response, error) {
    return rg.sendRequest("GET", url, nil)
}

// Post makes a POST request using the RushGo client
func (rg *RushGo) Post(url string, body []byte) (*http.Response, error) {
    return rg.sendRequest("POST", url, body)
}

// Put makes a PUT request using the RushGo client
func (rg *RushGo) Put(url string, body []byte) (*http.Response, error) {
    return rg.sendRequest("PUT", url, body)
}


// Patch makes a PATCH request using the RushGo client
func (rg *RushGo) Patch(url string, body []byte) (*http.Response, error) {
    return rg.sendRequest("PATCH", url, body)
}

// Delete makes a DELETE request using the RushGo client
func (rg *RushGo) Delete(url string) (*http.Response, error) {
    return rg.sendRequest("DELETE", url, nil)
}

func (rg *RushGo) Head(url string) (*http.Response, error) {
    return rg.sendRequest("HEAD", url, nil)
}

func (rg *RushGo) Options(url string) (*http.Response, error) {
    return rg.sendRequest("OPTIONS", url, nil)
}

func (rg *RushGo) WithBasicAuth(username, password string) *RushGo {
    rg.defaultHeaders["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
    return rg
}

func (rg *RushGo) WithBearerToken(token string) *RushGo {
    rg.defaultHeaders["Authorization"] = "Bearer " + token
    return rg
}

func (rg *RushGo) FollowRedirects() *RushGo {
    rg.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
        return nil
    }
    return rg
}

func (rg *RushGo) WithProxy(proxyURL string) *RushGo {
    if url, err := url.Parse(proxyURL); err == nil {
        rg.client.Transport = &http.Transport{
            Proxy: http.ProxyURL(url),
        }
    }
    return rg
}

// sendRequest is a helper method to make HTTP requests
func (rg *RushGo) sendRequest(method, url string, body []byte) (*http.Response, error) {
    req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }

    // Apply default headers to the request
    for key, value := range rg.defaultHeaders {
        req.Header.Set(key, value)
    }

    // Set User-Agent header if it's provided
    if rg.userAgent != "" {
        req.Header.Set("User-Agent", rg.userAgent)
    }

    return rg.client.Do(req)
}

func (rg *RushGo) SetHeaders(headers map[string]string) *RushGo {
    for key, value := range headers {
        rg.defaultHeaders[key] = value
    }
    return rg
}

// SetCookies sets cookies for the RushGo client without replacing the existing ones.
func (rg *RushGo) SetCookies(cookies map[string]string) *RushGo {
    // Merge the new cookies with the existing ones
    for name, value := range cookies {
        existingValue, exists := rg.defaultHeaders["Cookie"]
        if exists {
            // Append the new cookie to the existing ones
            rg.defaultHeaders["Cookie"] = existingValue + "; " + fmt.Sprintf("%s=%s", name, value)
        } else {
            // Set the new cookie as the first one
            rg.defaultHeaders["Cookie"] = fmt.Sprintf("%s=%s", name, value)
        }
    }
    return rg
}

func (rg *RushGo) WithUserAgent(userAgent string) *RushGo {
    if userAgent == "random" {
        // Generate and set a random User-Agent
        rg.userAgent = RandUserAgent().String()
    } else {
        rg.userAgent = userAgent
    }
    return rg
}


func (rg *RushGo) WebSocketConnect(urlStr string) (*websocket.Conn, *http.Response, error) {
    // You can customize the Dialer if you need to set timeouts or other settings
    dialer := websocket.DefaultDialer

    // Pass headers if needed
    headers := http.Header{}
    for key, value := range rg.defaultHeaders {
        headers.Add(key, value)
    }

    // Connect to the WebSocket server
    conn, resp, err := dialer.Dial(urlStr, headers)
    if err != nil {
        return nil, nil, err
    }

    return conn, resp, err
}




// DownloadImage downloads an image from the given URL and saves it to the specified path.
// If savePath is nil, the image is saved in the current working directory with its original filename.
// It returns the http.Response and an error, if any.
func (rg *RushGo) DownloadImage(url string, savePath *string) (*http.Response, error) {
    // Make a GET request to the image URL
    resp, err := rg.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Check if the response status code is 200 (OK)
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
    }

    // Determine the save path
    var finalPath string
    if savePath == nil {
        // Extract filename from the URL
        _, fileName := path.Split(url)
        // Determine the file extension from the Content-Type header
        contentType := resp.Header.Get("Content-Type")
        ext := ".jpg" // Default extension if Content-Type is not available or not recognized
        if contentType != "" {
            ext = "." + strings.Split(contentType, "/")[1]
        }
        finalPath = filepath.Join(".", fileName+ext)
    } else {
        // Use the provided path
        finalPath = *savePath
    }

    // Create a file to save the image
    file, err := os.Create(finalPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    // Copy the image data from the response to the file
    _, err = io.Copy(file, resp.Body)
    if err != nil {
        return nil, err
    }

    return resp, nil
}
