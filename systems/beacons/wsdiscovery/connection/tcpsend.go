package connection

import (
	"gorogs/logger"
	"net/http"
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
