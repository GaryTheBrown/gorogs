package engine

import (
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/connection"
	"gorogs/systems/beacons/wsdiscovery/templates"
	"gorogs/systems/beacons/wsdiscovery/versions"
)

func (e *Engine) BroadcastHello() {
	for schemaVersion := range versions.SchemaList {
		payloadBytes, err := templates.GenerateXMLResponse(
			schemaVersion,
			versions.ToValueList[schemaVersion]["request"],
			versions.SchemaList[schemaVersion][versions.Discovery],
			versions.Hello.String(),
			"",
			"",
		)
		if err != nil {
			logger.ErrorF(Name, "XML transmission synthesis failed on Hello announcement serialization steps for version: %s", err, schemaVersion)
			continue
		}
		err = connection.SendMulticastBroadcast(payloadBytes)
		if err != nil {
			logger.ErrorF(Name, "Multicast transmission delivery failed for Hello startup packet frame version: %s", err, schemaVersion)
			continue
		}
	}
}

func (s *Engine) BroadcastBye() {
	defer close(s.FlushDone)

	for schemaVersion := range versions.SchemaList {
		payloadBytes, err := templates.GenerateXMLResponse(
			schemaVersion,
			versions.ToValueList[schemaVersion]["request"],
			versions.SchemaList[schemaVersion][versions.Discovery],
			versions.Bye.String(),
			"",
			"",
		)
		if err != nil {
			logger.ErrorF(Name, "XML transmission synthesis failed on Bye notice serialization steps for version: %s", err, schemaVersion)
			continue
		}

		err = connection.SendMulticastBroadcast(payloadBytes)
		if err != nil {
			logger.ErrorF(Name, "Multicast transmission delivery failed for Bye shutdown packet frame version: %s", err, schemaVersion)
			continue
		}
	}
	logger.Debug(Name, "All 'Bye' multicast payload byte streams successfully written to the system buffer.")
}
