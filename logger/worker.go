package logger

import (
	"fmt"
	"os"
)

type LogType int

const (
	typeStandard LogType = iota
	typeStart
	typeAppend
	typeEnd
	typeFatal
)

type logMessage struct {
	kind      LogType
	text      string
	subSystem string
}

var (
	logChan    = make(chan logMessage, 1000)
	fatalChan  = make(chan string)
	fatalAck   = make(chan struct{})
	workerDone = make(chan struct{})
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

					case innerMsg := <-logChan:
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
							logChan <- innerMsg
						}
					}
				}
			}
		}
	}
}
