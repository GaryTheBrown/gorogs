package connection

import (
	"bytes"
	"gorogs/logger"
	"net/http"
	"time"
)

func SendSimpleHTTPResponse(w http.ResponseWriter, statusCode int, statusText string) {
	http.Error(w, statusText, statusCode)
}

func StreamHTTPMetadataPayload(w http.ResponseWriter, xmlPayload []byte) error {
	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(xmlPayload); err != nil {
		logger.Error("WSDiscovery", "Socket drop encountered streaming raw XML payload body data through HTTP response writer", err)
		return err
	}

	return nil
}

func SendTCPUnicastResponse(payload []byte, targetURL string) error {
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Post(targetURL, "application/soap+xml", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}
