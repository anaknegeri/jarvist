package hardware

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/denisbrodbeck/machineid"
)

// GetMachineID returns a unique identifier for the current machine
func GetMachineID() (string, error) {
	// Gunakan library machineid untuk mendapatkan ID yang unik per-OS
	id, err := machineid.ID()
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetHardwareID returns a comprehensive hardware identifier
func GetHardwareID() (string, error) {
	// Kumpulkan berbagai identifikasi perangkat keras
	var identifiers []string

	// 1. Machine ID
	machineID, err := GetMachineID()
	if err == nil {
		identifiers = append(identifiers, machineID)
	}

	// 2. MAC Address (alamat fisik dari kartu jaringan)
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if iface.HardwareAddr != nil && len(iface.HardwareAddr.String()) > 0 {
			// Hanya gunakan ethernet/wireless, abaikan virtual interfaces
			if !strings.Contains(iface.Name, "vEthernet") &&
				!strings.Contains(iface.Name, "VMware") &&
				!strings.Contains(iface.Name, "VirtualBox") {
				identifiers = append(identifiers, iface.HardwareAddr.String())
			}
		}
	}

	// 3. Hostname
	hostname, err := os.Hostname()
	if err == nil {
		identifiers = append(identifiers, hostname)
	}

	// 4. BIOS/Motherboard information
	biosInfo := GetBIOSInfo()
	identifiers = append(identifiers, biosInfo...)

	// 5. CPU information
	cpuInfo := GetCPUInfo()
	identifiers = append(identifiers, cpuInfo)

	// Gabungkan semua ID dan hash
	combinedID := strings.Join(identifiers, "|")
	hasher := sha256.New()
	hasher.Write([]byte(combinedID))
	hardwareID := hex.EncodeToString(hasher.Sum(nil))

	return hardwareID, nil
}

func GetBIOSInfo() []string {
	var info []string

	biosSerial := runCommand("wmic", "bios", "get", "serialnumber")
	if biosSerial != "" {
		info = append(info, "BIOS:"+biosSerial)
	}

	motherboardSerial := runCommand("wmic", "baseboard", "get", "serialnumber")
	if motherboardSerial != "" {
		info = append(info, "MB:"+motherboardSerial)
	}

	motherboardManufacturer := runCommand("wmic", "baseboard", "get", "manufacturer")
	if motherboardManufacturer != "" {
		info = append(info, "MBM:"+motherboardManufacturer)
	}

	motherboardProduct := runCommand("wmic", "baseboard", "get", "product")
	if motherboardProduct != "" {
		info = append(info, "MBP:"+motherboardProduct)
	}

	if len(info) == 0 {
		info = append(info, fmt.Sprintf("FallbackOS:%s-%s", runtime.GOOS, runtime.GOARCH))
	}

	return info
}

// GetCPUInfo gets CPU identifier information
func GetCPUInfo() string {
	cpuID := runCommand("wmic", "cpu", "get", "processorid")
	if cpuID != "" {
		return "CPU:" + cpuID
	}

	// Fallback
	return "CPUCores:" + fmt.Sprintf("%d", runtime.NumCPU())
}

func runCommand(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.Contains(line, "wmic") && !strings.Contains(line, "Serial Number") {
			return line
		}
	}

	return ""
}
