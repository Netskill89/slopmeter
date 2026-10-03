package main

import (
	"encoding/json"
	"net"
	"os"
)

// Network adapters can be listed without capture privileges or a password prompt.
func listNetworkInterfaces() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	rows := []map[string]string{{"name": "", "label": "Automatic (capture default)"}, {"name": "any", "label": "All interfaces (any)"}}
	for _, device := range interfaces {
		label := device.Name
		if device.Flags&net.FlagUp == 0 {
			label += " (down)"
		}
		if device.Flags&net.FlagLoopback != 0 {
			label += " (loopback)"
		}
		rows = append(rows, map[string]string{"name": device.Name, "label": label})
	}
	return json.NewEncoder(os.Stdout).Encode(rows)
}
