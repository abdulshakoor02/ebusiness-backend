package services

import (
	"context"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CountryService struct {
	countryRepo ports.CountryRepository
}

func NewCountryService(countryRepo ports.CountryRepository) *CountryService {
	return &CountryService{
		countryRepo: countryRepo,
	}
}

func (s *CountryService) CreateCountry(ctx context.Context, req ports.CreateCountryRequest) (*domain.Country, error) {
	country := domain.NewCountry(
		req.Name,
		req.ISO2,
		req.ISO3,
		req.PhoneCode,
		req.Currency,
		req.CurrencyName,
	)

	if err := s.countryRepo.Create(ctx, country); err != nil {
		return nil, err
	}

	return country, nil
}

func (s *CountryService) GetCountry(ctx context.Context, id primitive.ObjectID) (*domain.Country, error) {
	return s.countryRepo.GetByID(ctx, id)
}

func (s *CountryService) UpdateCountry(ctx context.Context, id primitive.ObjectID, req ports.UpdateCountryRequest) (*domain.Country, error) {
	country, err := s.countryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		country.Name = req.Name
	}
	if req.ISO2 != "" {
		country.ISO2 = req.ISO2
	}
	if req.ISO3 != "" {
		country.ISO3 = req.ISO3
	}
	if req.PhoneCode != "" {
		country.PhoneCode = req.PhoneCode
	}
	if req.Currency != "" {
		country.Currency = req.Currency
	}
	if req.CurrencyName != "" {
		country.CurrencyName = req.CurrencyName
	}
	if req.IsActive != nil {
		country.IsActive = *req.IsActive
	}

	if err := s.countryRepo.Update(ctx, country); err != nil {
		return nil, err
	}

	return country, nil
}

func (s *CountryService) DeleteCountry(ctx context.Context, id primitive.ObjectID) error {
	return s.countryRepo.Delete(ctx, id)
}

func (s *CountryService) ListCountries(ctx context.Context, req ports.FilterRequest) ([]*domain.Country, int64, error) {
	filters := req.Filters
	if filters == nil {
		filters = make(map[string]interface{})
	}

	if _, exists := filters["is_active"]; !exists {
		filters["is_active"] = true
	}

	return s.countryRepo.List(ctx, filters, req.Offset, req.Limit)
}
