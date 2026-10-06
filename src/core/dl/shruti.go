package dl

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"ashokshau/tgmusic/config"
)

func shrutiRetryDelay(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds >= 0 {
				if seconds > 30 {
					seconds = 30
				}
				return time.Duration(seconds) * time.Second
			}
		}
	}

	delay := time.Duration(attempt) * 2 * time.Second
	if delay > 10*time.Second {
		delay = 10 * time.Second
	}
	return delay
}

func newShrutiHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			ForceAttemptHTTP2: false,
		},
	}
}

func downloadWithShruti(videoID string, isVideo bool) (string, error) {
	shrutiDownloadMu.Lock()
	defer shrutiDownloadMu.Unlock()

	apiURL := strings.TrimRight(os.Getenv("SHRUTI_API_URL"), "/")
	apiKey := os.Getenv("SHRUTI_API_KEY")

	if apiURL == "" {
		return "", fmt.Errorf("SHRUTI_API_URL is not configured")
	}
	if apiKey == "" {
		return "", fmt.Errorf("SHRUTI_API_KEY is not configured")
	}

	mediaType := "audio"
	ext := ".mp3"
	timeout := 5 * time.Minute

	if isVideo {
		mediaType = "video"
		ext = ".mp4"
		timeout = 10 * time.Minute
	}

	u, err := url.Parse(apiURL + "/download")
	if err != nil {
		return "", fmt.Errorf("invalid Shruti API URL: %w", err)
	}

	q := u.Query()
	q.Set("url", videoID)
	q.Set("type", mediaType)
	q.Set("api_key", apiKey)
	u.RawQuery = q.Encode()

	fileName := videoID + ext
	filePath := filepath.Join(config.DownloadsDir, fileName)
	tempPath := filePath + ".part"

	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}

	if err := os.MkdirAll(config.DownloadsDir, 0755); err != nil {
		return "", fmt.Errorf("create downloads directory: %w", err)
	}

	client := newShrutiHTTPClient(timeout)

	const maxAttempts = 8

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		offset := int64(0)

		if info, err := os.Stat(tempPath); err == nil {
			offset = info.Size()
		}

		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return "", fmt.Errorf("create Shruti request: %w", err)
		}

		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}

		resp, err := client.Do(req)
		if err != nil {
			if attempt == maxAttempts {
				return "", fmt.Errorf("Shruti request failed after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if offset > 0 && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()

			if attempt == maxAttempts {
				return "", fmt.Errorf(
					"Shruti resume failed: expected HTTP 206, got HTTP %d",
					resp.StatusCode,
				)
			}

			time.Sleep(500 * time.Millisecond)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			resp.Body.Close()

			if attempt == maxAttempts {
				return "", fmt.Errorf(
					"Shruti API returned HTTP 429 after %d attempts: %s",
					maxAttempts,
					strings.TrimSpace(string(body)),
				)
			}

			time.Sleep(shrutiRetryDelay(resp, attempt))
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			resp.Body.Close()
			return "", fmt.Errorf(
				"Shruti API returned HTTP %d: %s",
				resp.StatusCode,
				strings.TrimSpace(string(body)),
			)
		}

		flags := os.O_CREATE | os.O_WRONLY
		if offset > 0 {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}

		f, err := os.OpenFile(tempPath, flags, 0644)
		if err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("open download file: %w", err)
		}

		_, copyErr := io.Copy(f, resp.Body)
		closeErr := f.Close()
		resp.Body.Close()

		if copyErr == nil && closeErr == nil {
			if err := os.Rename(tempPath, filePath); err != nil {
				os.Remove(tempPath)
				return "", fmt.Errorf("finalize Shruti download: %w", err)
			}
			return filePath, nil
		}

		if closeErr != nil {
			copyErr = closeErr
		}

		if attempt == maxAttempts {
			os.Remove(tempPath)
			return "", fmt.Errorf(
				"save Shruti download failed after %d attempts: %w",
				maxAttempts,
				copyErr,
			)
		}

		// Keep the partial file and resume from its current size.
		time.Sleep(300 * time.Millisecond)
	}

	return "", fmt.Errorf("Shruti download failed")
}

// Keep strconv referenced for compatibility with builds that inspect
// Content-Range parsing in this downloader.
var shrutiDownloadMu sync.Mutex

var _ = strconv.IntSize
