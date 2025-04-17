package location

import (
	"jarvist/internal/common/models"

	"gorm.io/gorm"
)

type LocationService struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *LocationService {
	return &LocationService{
		DB: db,
	}
}

func (s *LocationService) CreateLocation(input models.LocationInput) (*models.Location, error) {
	location := &models.Location{
		Name:        input.Name,
		Description: input.Description,
	}

	if err := s.DB.Create(location).Error; err != nil {
		return nil, err
	}

	return location, nil
}

func (s *LocationService) ListLocations() ([]models.Location, error) {
	var locations []models.Location
	result := s.DB.Find(&locations)
	if result.Error != nil {
		return nil, result.Error
	}
	return locations, nil
}

func (s *LocationService) UpdateLocation(id string, input models.LocationInput) (*models.Location, error) {
	var location models.Location
	result := s.DB.First(&location, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}

	location.Name = input.Name
	location.Description = input.Description

	result = s.DB.Save(&location)
	if result.Error != nil {
		return nil, result.Error
	}

	return &location, nil
}

func (s *LocationService) DeleteLocation(id string) error {
	result := s.DB.Delete(&models.Location{}, "id = ?", id)
	return result.Error
}
