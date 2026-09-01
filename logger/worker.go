package logger

import (
	"fmt"
	"os"
)

type LogModeType int

const (
	typeStandard LogModeType = iota
	typeStart
	typeAppend
	typeEnd
	typeFatal
)

type logMessage struct {
	kind      LogModeType
	text      string
	subSystem string
}

var (
	logChan    chan logMessage = make(chan logMessage, 1000)
	fatalChan  chan string     = make(chan string)
	fatalAck   chan struct{}   = make(chan struct{})
	workerDone chan struct{}   = make(chan struct{})
)

func init() {
	go logWorker()
}

func Close() {
	close(logChan)
	<-workerDone
}

func fatalScript(lineIsOpen bool, fatalText string) {
	if lineIsOpen {
		fmt.Fprintln(os.Stdout)
	}

	fmt.Fprint(os.Stdout, fatalText)
	_ = os.Stdout.Sync()

	close(fatalAck)
}

func logWorker() {
	defer close(workerDone)
	lineIsOpen := false
	var overflowBuffer []logMessage

	for {
		select {
		case fatalText := <-fatalChan:
			fatalScript(lineIsOpen, fatalText)
			return
		case msg, ok := <-logChan:
			if !ok {
				return
			}

			switch msg.kind {
			case typeStandard:
				fmt.Fprint(os.Stdout, msg.text)

			case typeStart:
				fmt.Fprint(os.Stdout, msg.text)
				_ = os.Stdout.Sync()
				lineIsOpen = true

			Lockdown:
				for {
					select {
					case fatalText := <-fatalChan:
						fatalScript(lineIsOpen, fatalText)
						return

					case innerMsg, ok := <-logChan:
						if !ok {
							lineIsOpen = false
							break Lockdown
						}

						if innerMsg.subSystem == msg.subSystem {
							if innerMsg.kind == typeAppend {
								fmt.Fprint(os.Stdout, innerMsg.text)
								_ = os.Stdout.Sync()
							} else if innerMsg.kind == typeEnd {
								fmt.Fprint(os.Stdout, innerMsg.text)
								lineIsOpen = false
								break Lockdown
							}
						} else {
							overflowBuffer = append(overflowBuffer, innerMsg)
						}
					}
				}

				if len(overflowBuffer) > 0 {
					for _, bufferedMsg := range overflowBuffer {
						if bufferedMsg.kind == typeStandard || bufferedMsg.kind == typeStart {
							fmt.Fprint(os.Stdout, bufferedMsg.text)
						}
					}
					overflowBuffer = overflowBuffer[:0]
				}
			}
		}
	}
}
