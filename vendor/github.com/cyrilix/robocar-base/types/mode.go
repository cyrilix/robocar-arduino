package types

import (
	"log"
)

type DriveMode int

const (
	DriveModeInvalid = -1
	DriveModeUser    = iota
	DriveModePilot
)

func ToString(mode DriveMode) string {
	switch mode {
	case DriveModeUser:
		return "user"
	case DriveModePilot:
		return "pilot"
	default:
		return ""
	}
}

func ParseString(val string) DriveMode {
	switch val {
	case "user":
		return DriveModeUser
	case "pilot":
		return DriveModePilot
	default:
		log.Printf("invalid DriveMode: %v", val)
		return DriveModeInvalid
	}
}
