package connection

import (
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/incoming"
	"io"
	"net"
	"net/http"
)

type HTTPResponsePayload struct {
	BodyBytes []byte
	Err       error
}

func HandleIncomingHTTPTransfer(w http.ResponseWriter, r *http.Request, outputChan chan<- incoming.WSMessage) {
	remoteAddrStr := r.RemoteAddr

	if r.Method != http.MethodPost {
		logger.DebugF(Name, "[HTTP Transfer] Rejecting non-POST HTTP request method (%s) from: %s", r.Method, remoteAddrStr)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorF(Name, "[HTTP Transfer] Failed to read inbound payload bytes from: %s", err, remoteAddrStr)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(bodyBytes) == 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	clientAddr, _ := net.ResolveTCPAddr("tcp", remoteAddrStr)

	var msg incoming.WSMessage

	if FastDecodingMode {
		if err := incoming.QuickDecode(bodyBytes, &msg); err != nil {
			logger.ErrorF(Name, "[Worker] Fast Decoding Failed: %s", err, remoteAddrStr)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	} else {
		if err := incoming.FullDecode(bodyBytes, &msg); err != nil {
			logger.ErrorF(Name, "[Worker] Dropping invalid/malformed payload block from remote host: %s", err, remoteAddrStr)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	}
	if err := incoming.FullDecode(bodyBytes, &msg); err != nil {
		logger.ErrorF(Name, "[HTTP Transfer] Dropping invalid/malformed SOAP frame from: %s", err, remoteAddrStr)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	responsePipe := make(chan any)

	msg.Sender = clientAddr
	msg.HTTPResponsePipe = responsePipe

	outputChan <- msg

	select {
	case untypedResult := <-responsePipe:
		result, ok := untypedResult.(HTTPResponsePayload)
		if !ok || result.Err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if err := StreamHTTPMetadataPayload(w, result.BodyBytes); err != nil {
			logger.ErrorF(Name, "[HTTP Transfer] Pipeline execution failed transmitting content blocks to: %s", err, remoteAddrStr)
		}

	case <-r.Context().Done():
	}
}
