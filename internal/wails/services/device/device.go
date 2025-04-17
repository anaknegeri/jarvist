package device

import (
	"os"
	"runtime"
)

type DeviceInfo struct {
	Name         string
	OS           string
	Architecture string
}

type DeviceService struct{}

func New() *DeviceService {
	return &DeviceService{}
}

func (s *DeviceService) GetDeviceInfo() *DeviceInfo {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}

	osName := runtime.GOOS
	arch := runtime.GOARCH

	return &DeviceInfo{
		Name:         hostname,
		OS:           osName,
		Architecture: arch,
	}
}
