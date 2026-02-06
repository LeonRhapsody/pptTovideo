package tts

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/LeonRhapsody/pptTovideo/internal/config"
	"github.com/gorilla/websocket"
)

type XunfeiProvider struct {
	Config *config.Config
}

func NewXunfeiProvider(cfg *config.Config) *XunfeiProvider {
	return &XunfeiProvider{Config: cfg}
}

func (x *XunfeiProvider) Synthesize(text string, outputPath string, voiceName string) error {
	if x.Config.XunfeiAppID == "" || x.Config.XunfeiAPIKey == "" || x.Config.XunfeiAPISecret == "" {
		return fmt.Errorf("Xunfei credentials not configured")
	}

	if voiceName == "" {
		voiceName = "xiaoyan"
	}

	hostUrl := "wss://tts-api.xfyun.cn/v2/tts"
	d := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	urlStr := x.assembleAuthUrl(hostUrl, x.Config.XunfeiAPIKey, x.Config.XunfeiAPISecret)
	conn, _, err := d.Dial(urlStr, nil)
	if err != nil {
		return fmt.Errorf("dialing xunfei: %v", err)
	}
	defer conn.Close()

	frameData := map[string]interface{}{
		"common": map[string]interface{}{
			"app_id": x.Config.XunfeiAppID,
		},
		"business": map[string]interface{}{
			"aue":    "lame", // mp3
			"sfl":    1,
			"vcn":    voiceName,
			"speed":  50,
			"volume": 50,
			"pitch":  50,
			"bgs":    0,
			"tte":    "UTF8",
		},
		"data": map[string]interface{}{
			"status": 2,
			"text":   base64.StdEncoding.EncodeToString([]byte(text)),
		},
	}

	if err := conn.WriteJSON(frameData); err != nil {
		return fmt.Errorf("sending data: %v", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %v", err)
	}
	defer outFile.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read message: %v", err)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(msg, &resp); err != nil {
			return err
		}

		if code, ok := resp["code"].(float64); ok && code != 0 {
			return fmt.Errorf("xunfei api error code: %v, message: %v", code, resp["message"])
		}

		if data, ok := resp["data"].(map[string]interface{}); ok {
			if audio, ok := data["audio"].(string); ok {
				decoded, err := base64.StdEncoding.DecodeString(audio)
				if err != nil {
					return err
				}
				outFile.Write(decoded)
			}

			if status, ok := data["status"].(float64); ok && status == 2 {
				// Last frame
				break
			}
		}
	}

	return nil
}

func (x *XunfeiProvider) assembleAuthUrl(hosturl string, apiKey, apiSecret string) string {
	ul, err := url.Parse(hosturl)
	if err != nil {
		return hosturl
	}
	date := time.Now().UTC().Format(time.RFC1123)
	signString := []string{"host: " + ul.Host, "date: " + date, "GET " + ul.Path + " HTTP/1.1"}
	sgin := strings.Join(signString, "\n")
	sha := hmacWithShaTobase64("hmac-sha256", sgin, apiSecret)
	authUrl := fmt.Sprintf("hmac username=\"%s\", algorithm=\"%s\", headers=\"%s\", signature=\"%s\"", apiKey, "hmac-sha256", "host date request-line", sha)
	authorization := base64.StdEncoding.EncodeToString([]byte(authUrl))
	v := url.Values{}
	v.Add("host", ul.Host)
	v.Add("date", date)
	v.Add("authorization", authorization)
	return hosturl + "?" + v.Encode()
}

func hmacWithShaTobase64(algorithm, data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	encodeData := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(encodeData)
}
