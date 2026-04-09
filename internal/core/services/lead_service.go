package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"strconv"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/abdulshakoor02/goCrmBackend/pkg/ai"
	"github.com/abdulshakoor02/goCrmBackend/pkg/excel"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadService struct {
	leadRepo          ports.LeadRepository
	categoryRepo      ports.LeadCategoryRepository
	sourceRepo        ports.LeadSourceRepository
	qualificationRepo ports.QualificationRepository
	countryRepo       ports.CountryRepository
	commentRepo       ports.LeadCommentRepository
	aiClient          *ai.Client
	maxFileSize       int64
	maxRows           int64
}

func NewLeadService(leadRepo ports.LeadRepository, categoryRepo ports.LeadCategoryRepository, sourceRepo ports.LeadSourceRepository, qualificationRepo ports.QualificationRepository, countryRepo ports.CountryRepository, commentRepo ports.LeadCommentRepository, aiClient *ai.Client, maxFileSize, maxRows int64) *LeadService {
	return &LeadService{
		leadRepo:          leadRepo,
		categoryRepo:      categoryRepo,
		sourceRepo:        sourceRepo,
		qualificationRepo: qualificationRepo,
		countryRepo:       countryRepo,
		commentRepo:       commentRepo,
		aiClient:          aiClient,
		maxFileSize:       maxFileSize,
		maxRows:           maxRows,
	}
}

func (s *LeadService) CreateLead(ctx context.Context, req ports.CreateLeadRequest) (*domain.Lead, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to create lead")
	}

	lead := domain.NewLead(
		tenantID,
		req.FirstName,
		req.LastName,
		req.Designation,
		req.Email,
		req.Phone,
	)

	if req.AssignedTo != "" {
		assignedToID, err := primitive.ObjectIDFromHex(req.AssignedTo)
		if err != nil {
			return nil, errors.New("invalid assigned_to user id format")
		}
		lead.AssignedTo = assignedToID
	}

	if req.CategoryID != "" {
		categoryID, err := primitive.ObjectIDFromHex(req.CategoryID)
		if err != nil {
			return nil, errors.New("invalid category_id form")
		}
		lead.CategoryID = categoryID
	}

	if req.SourceID != "" {
		sourceID, err := primitive.ObjectIDFromHex(req.SourceID)
		if err != nil {
			return nil, errors.New("invalid source_id format")
		}
		lead.SourceID = sourceID
	}

	if req.CountryID != "" {
		countryID, err := primitive.ObjectIDFromHex(req.CountryID)
		if err != nil {
			return nil, errors.New("invalid country_id format")
		}
		lead.CountryID = countryID
	}

	if req.QualificationID != "" {
		qualificationID, err := primitive.ObjectIDFromHex(req.QualificationID)
		if err != nil {
			return nil, errors.New("invalid qualification_id format")
		}
		lead.QualificationID = qualificationID
	}

	if req.Address.Street != "" || req.Address.City != "" || req.Address.State != "" || req.Address.Country != "" || req.Address.ZipCode != "" || req.Address.AddressLine != "" {
		lead.Address = req.Address
	}

	lead.BuildSearchText()

	if err := s.leadRepo.Create(ctx, lead); err != nil {
		return nil, err
	}

	return lead, nil
}

func (s *LeadService) GetLead(ctx context.Context, id primitive.ObjectID) (*domain.Lead, error) {
	return s.leadRepo.GetByID(ctx, id)
}

func (s *LeadService) UpdateLead(ctx context.Context, id primitive.ObjectID, req ports.UpdateLeadRequest) (*domain.Lead, error) {
	lead, err := s.leadRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.FirstName != "" {
		lead.FirstName = req.FirstName
	}
	if req.LastName != "" {
		lead.LastName = req.LastName
	}
	if req.Designation != "" {
		lead.Designation = req.Designation
	}
	if req.Email != "" {
		lead.Email = req.Email
	}
	if req.Phone != "" {
		lead.Phone = req.Phone
	}

	if req.AssignedTo != "" {
		assignedToID, err := primitive.ObjectIDFromHex(req.AssignedTo)
		if err != nil {
			return nil, errors.New("invalid assigned_to user id format")
		}
		lead.AssignedTo = assignedToID
	}

	if req.CategoryID != "" {
		categoryID, err := primitive.ObjectIDFromHex(req.CategoryID)
		if err != nil {
			return nil, errors.New("invalid category_id format")
		}
		lead.CategoryID = categoryID
	}

	if req.SourceID != "" {
		sourceID, err := primitive.ObjectIDFromHex(req.SourceID)
		if err != nil {
			return nil, errors.New("invalid source_id format")
		}
		lead.SourceID = sourceID
	}

	if req.CountryID != "" {
		countryID, err := primitive.ObjectIDFromHex(req.CountryID)
		if err != nil {
			return nil, errors.New("invalid country_id format")
		}
		lead.CountryID = countryID
	}

	if req.QualificationID != "" {
		qualificationID, err := primitive.ObjectIDFromHex(req.QualificationID)
		if err != nil {
			return nil, errors.New("invalid qualification_id format")
		}
		lead.QualificationID = qualificationID
	}

	if req.Address.Street != "" || req.Address.City != "" || req.Address.State != "" || req.Address.Country != "" || req.Address.ZipCode != "" || req.Address.AddressLine != "" {
		lead.Address = req.Address
	}

	lead.BuildSearchText()

	if err := s.leadRepo.Update(ctx, lead); err != nil {
		return nil, err
	}

	return lead, nil
}

func (s *LeadService) ListLeads(ctx context.Context, req ports.FilterRequest) ([]*ports.LeadListItem, int64, error) {
	return s.leadRepo.List(ctx, req.Filters, req.Search, req.Offset, req.Limit)
}

func (s *LeadService) UpdateLeadStatus(ctx context.Context, id primitive.ObjectID, req ports.UpdateLeadStatusRequest) (*domain.Lead, error) {
	lead, err := s.leadRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Status != domain.LeadStatusActive && req.Status != domain.LeadStatusInactive {
		return nil, errors.New("status must be 'active' or 'inactive'")
	}

	if lead.Status == domain.LeadStatusLead {
		return nil, errors.New("lead has not been converted to a client yet")
	}

	lead.Status = req.Status
	lead.UpdatedAt = time.Now()

	if err := s.leadRepo.Update(ctx, lead); err != nil {
		return nil, err
	}

	return lead, nil
}

func (s *LeadService) ImportLeads(ctx context.Context, data []byte, ext string, assignedTo string) (*ports.ImportResult, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to import leads")
	}

	if int64(len(data)) > s.maxFileSize {
		return nil, errors.New("file exceeds maximum size limit")
	}

	headers, rows, err := excel.ParseFile(data, ext)
	if err != nil {
		return nil, err
	}

	result := &ports.ImportResult{
		TotalRows: len(rows),
	}

	if len(rows) == 0 {
		return result, nil
	}

	if int64(len(rows)) > s.maxRows {
		return nil, errors.New("file exceeds maximum row limit: " + strconv.Itoa(len(rows)) + " rows found, max allowed: " + strconv.Itoa(int(s.maxRows)))
	}

	sampleRows := make([][]string, 0, 10)
	for i := 0; i < len(rows) && i < 10; i++ {
		sampleRows = append(sampleRows, rows[i])
	}

	colResult, err := excel.MapColumns(ctx, s.aiClient, headers, sampleRows)
	if err != nil {
		slog.Warn("Column mapping failed, using heuristic", "error", err)
	}

	existingCategories, _, _ := s.categoryRepo.List(ctx, nil, 0, 1000)
	existingSources, _, _ := s.sourceRepo.List(ctx, nil, 0, 1000)
	existingQualifications, _, _ := s.qualificationRepo.List(ctx, nil, 0, 1000)

	catOpts := toRefOptsCategories(existingCategories)
	srcOpts := toRefOptsSources(existingSources)
	qualOpts := toRefOptsQualifications(existingQualifications)

	catMap := make(map[string]excel.ReferenceOption)
	for _, o := range catOpts {
		catMap[strings.ToLower(o.Name)] = o
	}
	srcMap := make(map[string]excel.ReferenceOption)
	for _, o := range srcOpts {
		srcMap[strings.ToLower(o.Name)] = o
	}
	qualMap := make(map[string]excel.ReferenceOption)
	for _, o := range qualOpts {
		qualMap[strings.ToLower(o.Name)] = o
	}

	var defaultCountryID primitive.ObjectID
	defaultCountry, err := s.countryRepo.FindByName(ctx, "United Arab Emirates")
	if err == nil && defaultCountry != nil {
		defaultCountryID = defaultCountry.ID
	}

	var assignedToID primitive.ObjectID
	if assignedTo != "" {
		assignedToID, _ = primitive.ObjectIDFromHex(assignedTo)
	}

	leads := make([]*domain.Lead, 0, len(rows))
	createdCategories := make([]string, 0)
	createdSources := make([]string, 0)

	for i, row := range rows {
		lead := domain.NewLead(tenantID, "", "", "", "", "")

		for _, m := range colResult.Mappings {
			idx := m.ColumnIndex
			if idx < 0 || idx >= len(row) {
				continue
			}
			val := strings.TrimSpace(row[idx])
			if val == "" {
				continue
			}

			switch m.TargetField {
			case "first_name":
				lead.FirstName = val
			case "last_name":
				lead.LastName = val
			case "full_name":
				parts := strings.SplitN(val, " ", 2)
				lead.FirstName = parts[0]
				if len(parts) > 1 {
					lead.LastName = parts[1]
				}
			case "email":
				lead.Email = val
			case "phone":
				lead.Phone = val
			case "designation":
				lead.Designation = val
			case "comments":
				lead.Comments = val
			case "category_name":
				if opt, ok := catMap[strings.ToLower(val)]; ok {
					catID, _ := primitive.ObjectIDFromHex(opt.ID)
					lead.CategoryID = catID
				} else {
					newCat := domain.NewLeadCategory(tenantID, val, "Auto-created during import")
					if err := s.categoryRepo.Create(ctx, newCat); err == nil {
						lead.CategoryID = newCat.ID
						catMap[strings.ToLower(val)] = excel.ReferenceOption{ID: newCat.ID.Hex(), Name: newCat.Name}
						createdCategories = append(createdCategories, newCat.Name)
					}
				}
			case "source_name":
				if opt, ok := srcMap[strings.ToLower(val)]; ok {
					srcID, _ := primitive.ObjectIDFromHex(opt.ID)
					lead.SourceID = srcID
				} else {
					newSrc := domain.NewLeadSource(tenantID, val, "Auto-created during import")
					if err := s.sourceRepo.Create(ctx, newSrc); err == nil {
						lead.SourceID = newSrc.ID
						srcMap[strings.ToLower(val)] = excel.ReferenceOption{ID: newSrc.ID.Hex(), Name: newSrc.Name}
						createdSources = append(createdSources, newSrc.Name)
					}
				}
			case "qualification_name":
				if opt, ok := qualMap[strings.ToLower(val)]; ok {
					qualID, _ := primitive.ObjectIDFromHex(opt.ID)
					lead.QualificationID = qualID
				} else {
					result.Errors = append(result.Errors, ports.ImportError{
						Row:    i + 2,
						Field:  "qualification",
						Value:  val,
						Reason: "qualification not found in system",
					})
				}
			case "country_name":
				country, err := s.countryRepo.FindByName(ctx, val)
				if err == nil && country != nil {
					lead.CountryID = country.ID
				} else {
					lead.CountryID = defaultCountryID
				}
			}
		}

		if lead.CountryID.IsZero() {
			lead.CountryID = defaultCountryID
		}

		if !assignedToID.IsZero() {
			lead.AssignedTo = assignedToID
		}

		if lead.FirstName == "" && lead.LastName == "" {
			result.Errors = append(result.Errors, ports.ImportError{
				Row:    i + 2,
				Field:  "name",
				Reason: "first name or last name is required",
			})
			result.Skipped++
			continue
		}

		if lead.Email == "" && lead.Phone == "" {
			result.Errors = append(result.Errors, ports.ImportError{
				Row:    i + 2,
				Field:  "contact",
				Reason: "email or phone is required",
			})
			result.Skipped++
			continue
		}

		lead.BuildSearchText()

		existingLead, err := s.leadRepo.FindByEmailOrPhone(ctx, tenantID, lead.Email, lead.Phone)
		if err != nil {
			slog.Warn("Error checking for duplicate lead", "error", err)
		}

		if existingLead != nil {
			commentDate := time.Now().Format("2006-01-02 15:04")
			comment := domain.NewLeadComment(tenantID, existingLead.ID, primitive.NilObjectID, "Contacted again - "+commentDate)
			if err := s.commentRepo.Create(ctx, comment); err != nil {
				slog.Warn("Failed to add duplicate comment", "error", err)
			}
			result.Skipped++
			continue
		}

		leads = append(leads, lead)
	}

	if len(leads) > 0 {
		inserted, err := s.leadRepo.BulkInsert(ctx, leads)
		if err != nil {
			slog.Error("Bulk insert failed", "error", err)
			return result, err
		}
		result.Inserted = inserted
	}

	result.CreatedCategories = createdCategories
	result.CreatedSources = createdSources

	return result, nil
}

func toRefOptsCategories(cats []*domain.LeadCategory) []excel.ReferenceOption {
	opts := make([]excel.ReferenceOption, len(cats))
	for i, c := range cats {
		opts[i] = excel.ReferenceOption{ID: c.ID.Hex(), Name: c.Name}
	}
	return opts
}

func toRefOptsSources(srcs []*domain.LeadSource) []excel.ReferenceOption {
	opts := make([]excel.ReferenceOption, len(srcs))
	for i, s := range srcs {
		opts[i] = excel.ReferenceOption{ID: s.ID.Hex(), Name: s.Name}
	}
	return opts
}

func toRefOptsQualifications(quals []*domain.Qualification) []excel.ReferenceOption {
	opts := make([]excel.ReferenceOption, len(quals))
	for i, q := range quals {
		opts[i] = excel.ReferenceOption{ID: q.ID.Hex(), Name: q.Name}
	}
	return opts
}
