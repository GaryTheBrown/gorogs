package connection

import (
	"fmt"
	"gorogs/beacons/wsdd/incoming"
	"gorogs/logger"
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
	logger.Debug("wsdd", fmt.Sprintf("[HTTP Transfer] Received active metadata POST stream from client: %s", remoteAddrStr))

	if r.Method != http.MethodPost {
		logger.Debug("wsdd", fmt.Sprintf("[HTTP Transfer] Rejecting non-POST HTTP request method (%s) from: %s", r.Method, remoteAddrStr))
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("wsdd", fmt.Sprintf("[HTTP Transfer] Failed to read inbound payload bytes from: %s", remoteAddrStr), err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(bodyBytes) == 0 {
		logger.Debug("wsdd", fmt.Sprintf("[HTTP Transfer] Dropping empty HTTP request payload block from: %s", remoteAddrStr))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	logger.Debug("wsdd", fmt.Sprintf("[HTTP Transfer] Extracted %d bytes of raw XML metadata payload from client: %s. Handing over to decoding loop.", len(bodyBytes), remoteAddrStr))

	clientAddr, _ := net.ResolveTCPAddr("tcp", remoteAddrStr)

	var msg incoming.WSMessage
	if err := incoming.Decode(bodyBytes, &msg); err != nil {
		logger.Error("wsdd", fmt.Sprintf("[HTTP Transfer] Dropping invalid/malformed SOAP frame from: %s", remoteAddrStr), err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	responsePipe := make(chan interface{})

	msg.Sender = clientAddr
	msg.HTTPResponsePipe = responsePipe

	logger.Debug("wsdd", fmt.Sprintf("[HTTP Transfer] Forwarding verified WS-Transfer message packet up to the engine dispatcher queue from sender: %s", remoteAddrStr))

	outputChan <- msg

	select {
	case untypedResult := <-responsePipe:
		// Recover our strong tracking types from the empty interface channel container safely
		result, ok := untypedResult.(HTTPResponsePayload)
		if !ok || result.Err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if err := StreamHTTPMetadataPayload(w, result.BodyBytes); err != nil {
			logger.Error("wsdd", fmt.Sprintf("[HTTP Transfer] Pipeline execution failed transmitting content blocks to: %s", remoteAddrStr), err)
		}

	case <-r.Context().Done():
		logger.Debug("wsdd", fmt.Sprintf("[HTTP Transfer] Client connection abandoned or connection pipeline timed out prematurely: %s", remoteAddrStr))
	}
}
