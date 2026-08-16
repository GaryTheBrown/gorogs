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
	logger.DebugF("WSDiscovery", "[HTTP Transfer] Received active metadata POST stream from client: %s", remoteAddrStr)

	if r.Method != http.MethodPost {
		logger.DebugF("WSDiscovery", "[HTTP Transfer] Rejecting non-POST HTTP request method (%s) from: %s", r.Method, remoteAddrStr)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorF("WSDiscovery", "[HTTP Transfer] Failed to read inbound payload bytes from: %s", err, remoteAddrStr)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(bodyBytes) == 0 {
		logger.DebugF("WSDiscovery", "[HTTP Transfer] Dropping empty HTTP request payload block from: %s", remoteAddrStr)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	logger.DebugF("WSDiscovery", "[HTTP Transfer] Extracted %d bytes of raw XML metadata payload from client: %s. Handing over to decoding loop.", len(bodyBytes), remoteAddrStr)

	clientAddr, _ := net.ResolveTCPAddr("tcp", remoteAddrStr)

	var msg incoming.WSMessage

	if FastDecodingMode {
		if err := incoming.QuickDecode(bodyBytes, &msg); err != nil {
			logger.ErrorF("WSDiscovery", "[Worker] Fast Decoding Failed: %s", err, remoteAddrStr)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	} else {
		if err := incoming.FullDecode(bodyBytes, &msg); err != nil {
			logger.ErrorF("WSDiscovery", "[Worker] Dropping invalid/malformed payload block from remote host: %s", err, remoteAddrStr)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	}
	if err := incoming.FullDecode(bodyBytes, &msg); err != nil {
		logger.ErrorF("WSDiscovery", "[HTTP Transfer] Dropping invalid/malformed SOAP frame from: %s", err, remoteAddrStr)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	responsePipe := make(chan interface{})

	msg.Sender = clientAddr
	msg.HTTPResponsePipe = responsePipe

	logger.DebugF("WSDiscovery", "[HTTP Transfer] Forwarding verified WS-Transfer message packet up to the engine dispatcher queue from sender: %s", remoteAddrStr)

	outputChan <- msg

	select {
	case untypedResult := <-responsePipe:
		result, ok := untypedResult.(HTTPResponsePayload)
		if !ok || result.Err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if err := StreamHTTPMetadataPayload(w, result.BodyBytes); err != nil {
			logger.ErrorF("WSDiscovery", "[HTTP Transfer] Pipeline execution failed transmitting content blocks to: %s", err, remoteAddrStr)
		}

	case <-r.Context().Done():
		logger.DebugF("WSDiscovery", "[HTTP Transfer] Client connection abandoned or connection pipeline timed out prematurely: %s", remoteAddrStr)
	}
}
