package buildinfo

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
)

//go:embed info.json
var infoFile embed.FS

// BuildInfoService provides version information by directly reading info.json
type BuildInfoService struct {
	rawInfo *RawInfoJSON
}

// RawInfoJSON represents the structure of the info.json file
type RawInfoJSON struct {
	Fixed struct {
		FileVersion string `json:"file_version"`
	} `json:"fixed"`
	Info map[string]struct {
		ProductVersion  string `json:"ProductVersion"`
		CompanyName     string `json:"CompanyName"`
		FileDescription string `json:"FileDescription"`
		LegalCopyright  string `json:"LegalCopyright"`
		ProductName     string `json:"ProductName"`
		Comments        string `json:"Comments"`
	} `json:"info"`
}

type BuildInfo struct {
	FileVersion     string `json:"file_version"`
	ProductVersion  string `json:"product_version"`
	CompanyName     string `json:"company_name"`
	FileDescription string `json:"file_description"`
	LegalCopyright  string `json:"legal_copyright"`
	ProductName     string `json:"product_name"`
	Comments        string `json:"comments"`
}

// NewBuildInfoService creates a new BuildInfoService
func NewBuildInfoService() *BuildInfoService {
	service := &BuildInfoService{}
	err := service.loadInfoJSON()
	if err != nil {
		log.Printf("Error loading info.json: %v", err)
	}
	return service
}

func (s *BuildInfoService) LoadBuildInfo() BuildInfo {
	buildInfo := BuildInfo{
		FileVersion: "Unknown",
	}

	// Add fixed information
	if s.rawInfo != nil {
		buildInfo.FileVersion = s.rawInfo.Fixed.FileVersion
	}

	// Add info details
	if s.rawInfo != nil && len(s.rawInfo.Info) > 0 {
		for _, info := range s.rawInfo.Info {
			buildInfo.ProductVersion = info.ProductVersion
			buildInfo.CompanyName = info.CompanyName
			buildInfo.FileDescription = info.FileDescription
			buildInfo.LegalCopyright = info.LegalCopyright
			buildInfo.ProductName = info.ProductName
			buildInfo.Comments = info.Comments
			break // Assuming only one set of info is present
		}
	}

	return buildInfo
}

// loadInfoJSON reads and parses the info.json file
func (s *BuildInfoService) loadInfoJSON() error {
	// Use //go:embed to read the file
	data, err := infoFile.ReadFile("info.json")
	if err != nil {
		return fmt.Errorf("failed to read info.json: %w", err)
	}

	var rawInfo RawInfoJSON
	err = json.Unmarshal(data, &rawInfo)
	if err != nil {
		return fmt.Errorf("failed to parse info.json: %w", err)
	}

	s.rawInfo = &rawInfo
	return nil
}

// GetProductVersion returns the product version from info.json
func (s *BuildInfoService) GetProductVersion() string {
	if s.rawInfo != nil && len(s.rawInfo.Info) > 0 {
		for _, info := range s.rawInfo.Info {
			return info.ProductVersion
		}
	}
	return "Unknown"
}

// GetProductName returns the product name from info.json
func (s *BuildInfoService) GetProductName() string {
	if s.rawInfo != nil && len(s.rawInfo.Info) > 0 {
		for _, info := range s.rawInfo.Info {
			return info.ProductName
		}
	}
	return "Unknown"
}

// GetCompanyName returns the company name from info.json
func (s *BuildInfoService) GetCompanyName() string {
	if s.rawInfo != nil && len(s.rawInfo.Info) > 0 {
		for _, info := range s.rawInfo.Info {
			return info.CompanyName
		}
	}
	return "Unknown"
}

// GetFileVersion returns the file version from info.json
func (s *BuildInfoService) GetFileVersion() string {
	if s.rawInfo != nil {
		return s.rawInfo.Fixed.FileVersion
	}
	return "Unknown"
}

// GetCopyright returns the copyright from info.json
func (s *BuildInfoService) GetCopyright() string {
	if s.rawInfo != nil && len(s.rawInfo.Info) > 0 {
		for _, info := range s.rawInfo.Info {
			return info.LegalCopyright
		}
	}
	return "Unknown"
}

// GetFullInfoJSON returns the entire info.json content as a JSON string
func (s *BuildInfoService) GetFullInfoJSON() (string, error) {
	if s.rawInfo == nil {
		return "", fmt.Errorf("info.json not loaded")
	}

	data, err := json.Marshal(s.rawInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal info JSON: %w", err)
	}

	return string(data), nil
}

// Reload allows manual reloading of the info.json file
func (s *BuildInfoService) Reload() error {
	return s.loadInfoJSON()
}
