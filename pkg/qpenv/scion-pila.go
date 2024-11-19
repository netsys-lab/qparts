package qpenv

import "os"

type SCIONPilaSettings struct {
	TrcFolder  string
	ServerAddr string
	Enabled    bool
}

var PilaSettings = SCIONPilaSettings{
	TrcFolder:  "/etc/scion/certs",
	ServerAddr: "http://localhost:8843",
	Enabled:    false,
}

func init() {
	trcFolder := os.Getenv("SCION_PILA_TRC_FOLDER")
	if trcFolder != "" {
		PilaSettings.TrcFolder = trcFolder
	}

	serverAddr := os.Getenv("SCION_PILA_SERVER_ADDR")
	if serverAddr != "" {
		PilaSettings.ServerAddr = serverAddr
	}

	enabled := os.Getenv("SCION_PILA_ENABLED")
	if enabled != "" && enabled == "true" {
		PilaSettings.Enabled = true
	}
}
