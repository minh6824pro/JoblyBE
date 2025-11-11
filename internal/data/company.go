package data

import (
	"JobblyBE/internal/biz"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Company struct for MongoDB
type Company struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	Description string             `bson:"description"`
	Website     string             `bson:"website"`
	LogoURL     string             `bson:"logo_url"`
	Industry    string             `bson:"industry"`
	CompanySize string             `bson:"company_size"`
	Location    string             `bson:"location"`
	FoundedYear string             `bson:"founded_year"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

type companyRepo struct {
	data *Data
	log  *log.Helper
}

// NewCompanyRepo creates a new company repository
func NewCompanyRepo(data *Data, logger log.Logger) biz.CompanyRepo {
	return &companyRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// CreateCompany creates a new company
func (r *companyRepo) CreateCompany(ctx context.Context, company *biz.Company) (*biz.Company, error) {
	now := time.Now()
	dbCompany := &Company{
		Name:        company.Name,
		Description: company.Description,
		Website:     company.Website,
		LogoURL:     company.LogoURL,
		Industry:    company.Industry,
		CompanySize: company.CompanySize,
		Location:    company.Location,
		FoundedYear: company.FoundedYear,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	result, err := r.data.db.Collection(CollectionCompany).InsertOne(ctx, dbCompany)
	if err != nil {
		r.log.Errorf("failed to create company: %v", err)
		return nil, err
	}

	dbCompany.ID = result.InsertedID.(primitive.ObjectID)
	return r.toBiz(dbCompany), nil
}

// UpdateCompany updates an existing company
func (r *companyRepo) UpdateCompany(ctx context.Context, company *biz.Company) error {
	objID, err := primitive.ObjectIDFromHex(company.ID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"name":         company.Name,
			"description":  company.Description,
			"website":      company.Website,
			"logo_url":     company.LogoURL,
			"industry":     company.Industry,
			"company_size": company.CompanySize,
			"location":     company.Location,
			"founded_year": company.FoundedYear,
			"updated_at":   time.Now(),
		},
	}

	_, err = r.data.db.Collection(CollectionCompany).UpdateOne(
		ctx,
		bson.M{"_id": objID},
		update,
	)
	if err != nil {
		r.log.Errorf("failed to update company: %v", err)
		return err
	}

	return nil
}

// DeleteCompany deletes a company
func (r *companyRepo) DeleteCompany(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.data.db.Collection(CollectionCompany).DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		r.log.Errorf("failed to delete company: %v", err)
		return err
	}

	return nil
}

// GetCompany retrieves a company by ID
func (r *companyRepo) GetCompany(ctx context.Context, id string) (*biz.Company, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var company Company
	err = r.data.db.Collection(CollectionCompany).FindOne(ctx, bson.M{"_id": objID}).Decode(&company)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Company not found
		}
		r.log.Errorf("failed to get company: %v", err)
		return nil, err
	}

	return r.toBiz(&company), nil
}

// GetCompanyByName retrieves a company by name
func (r *companyRepo) GetCompanyByName(ctx context.Context, name string) (*biz.Company, error) {
	var company Company
	err := r.data.db.Collection(CollectionCompany).FindOne(ctx, bson.M{"name": name}).Decode(&company)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Company not found
		}
		r.log.Errorf("failed to get company by name: %v", err)
		return nil, err
	}

	return r.toBiz(&company), nil
}

// ListCompanies lists companies with filters and pagination
func (r *companyRepo) ListCompanies(ctx context.Context, filter *biz.CompanyFilter, page, pageSize int32) ([]*biz.Company, int32, error) {
	// Build filter query
	query := bson.M{}

	if filter != nil {
		if filter.Industry != "" {
			// Case-insensitive match for industry
			query["industry"] = bson.M{"$regex": filter.Industry, "$options": "i"}
		}
		if filter.Location != "" {
			query["location"] = bson.M{"$regex": filter.Location, "$options": "i"}
		}
		if filter.Keyword != "" {
			query["$or"] = []bson.M{
				{"name": bson.M{"$regex": filter.Keyword, "$options": "i"}},
				{"description": bson.M{"$regex": filter.Keyword, "$options": "i"}},
			}
		}
	}

	// Count total
	total, err := r.data.db.Collection(CollectionCompany).CountDocuments(ctx, query)
	if err != nil {
		r.log.Errorf("failed to count companies: %v", err)
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * pageSize

	// Find with pagination
	opts := options.Find().
		SetSort(bson.M{"created_at": -1}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := r.data.db.Collection(CollectionCompany).Find(ctx, query, opts)
	if err != nil {
		r.log.Errorf("failed to list companies: %v", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var companies []*biz.Company
	for cursor.Next(ctx) {
		var company Company
		if err := cursor.Decode(&company); err != nil {
			continue
		}
		companies = append(companies, r.toBiz(&company))
	}

	return companies, int32(total), nil
}

// toBiz converts data layer Company to biz layer Company
func (r *companyRepo) toBiz(c *Company) *biz.Company {
	return &biz.Company{
		ID:          c.ID.Hex(),
		Name:        c.Name,
		Description: c.Description,
		Website:     c.Website,
		LogoURL:     c.LogoURL,
		Industry:    c.Industry,
		CompanySize: c.CompanySize,
		Location:    c.Location,
		FoundedYear: c.FoundedYear,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
