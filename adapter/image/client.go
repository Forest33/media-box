package image

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"media-box-ng/pkg/logger"
)

type Config struct {
	OutputPath string
}

type Client struct {
	cfg *Config
	log *logger.Zerolog
}

func NewImageClient(cfg *Config, log *logger.Zerolog) *Client {
	return &Client{
		cfg: cfg,
		log: log,
	}
}

func (c *Client) Get(url string) (string, error) {
	f := fmt.Sprintf("%x", md5.Sum([]byte(url)))
	imgPath := filepath.Join(c.cfg.OutputPath, f)
	if _, err := os.Stat(imgPath); err == nil {
		return imgPath, nil
	}

	return imgPath, c.load(url, imgPath)
}

func (c *Client) load(url, output string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error().Msgf("error close body: %v", err)
		}
	}()

	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			c.log.Error().Msgf("error close file: %v", err)
		}
	}()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
