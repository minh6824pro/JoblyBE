package data

import (
	"JobblyBE/internal/biz"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// User struct for MongoDB
type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	FullName    string             `bson:"full_name"`
	Email       string             `bson:"email"`
	Password    string             `bson:"password"` // hashed password
	PhoneNumber string             `bson:"phone_number"`
	Role        string             `bson:"role"`
	Active      bool               `bson:"active"`
	Resume      []Resume           `bson:"resume"`
	LastLogin   *time.Time         `bson:"last_login,omitempty"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

type userRepo struct {
	data *Data
	log  *log.Helper
}

// NewUserRepo creates a new user repository
func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// CreateUser creates a new user in database
func (r *userRepo) CreateUser(ctx context.Context, user *biz.User) (*biz.User, error) {
	now := time.Now()
	dbUser := &User{
		FullName:    user.FullName,
		Email:       user.Email,
		Password:    user.Password,
		PhoneNumber: user.PhoneNumber,
		Role:        string(user.Role),
		Active:      user.Active,
		Resume:      []Resume{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	result, err := r.data.db.Collection(CollectionUser).InsertOne(ctx, dbUser)
	if err != nil {
		return nil, err
	}

	dbUser.ID = result.InsertedID.(primitive.ObjectID)
	return r.toBiz(dbUser), nil
}

// GetUserByEmail retrieves user by email
func (r *userRepo) GetUserByEmail(ctx context.Context, email string) (*biz.User, error) {
	var user User
	err := r.data.db.Collection(CollectionUser).FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // user not found
		}
		r.log.Errorf("failed to get user by email: %v", err)
		return nil, err
	}

	return r.toBiz(&user), nil
}

// GetUserByID retrieves user by ID
func (r *userRepo) GetUserByID(ctx context.Context, id string) (*biz.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var user User
	err = r.data.db.Collection(CollectionUser).FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // user not found
		}
		r.log.Errorf("failed to get user by id: %v", err)
		return nil, err
	}

	return r.toBiz(&user), nil
}

// UpdateLastLogin updates user's last login time
func (r *userRepo) UpdateLastLogin(ctx context.Context, userID string) error {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = r.data.db.Collection(CollectionUser).UpdateOne(
		ctx,
		bson.M{"_id": objID},
		bson.M{
			"$set": bson.M{
				"last_login": now,
				"updated_at": now,
			},
		},
	)
	if err != nil {
		r.log.Errorf("failed to update last login: %v", err)
		return err
	}

	return nil
}

// UpdateUser updates user information
func (r *userRepo) UpdateUser(ctx context.Context, user *biz.User) error {
	objID, err := primitive.ObjectIDFromHex(user.UserID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"full_name":    user.FullName,
			"password":     user.Password,
			"phone_number": user.PhoneNumber,
			"role":         string(user.Role),
			"active":       user.Active,
			"updated_at":   time.Now(),
		},
	}

	_, err = r.data.db.Collection(CollectionUser).UpdateOne(
		ctx,
		bson.M{"_id": objID},
		update,
	)
	if err != nil {
		r.log.Errorf("failed to update user: %v", err)
		return err
	}

	return nil
}

// toBiz converts data layer User to biz layer User
func (r *userRepo) toBiz(u *User) *biz.User {
	return &biz.User{
		UserID:      u.ID.Hex(),
		FullName:    u.FullName,
		Email:       u.Email,
		Password:    u.Password,
		PhoneNumber: u.PhoneNumber,
		Role:        biz.Role(u.Role),
		Active:      u.Active,
		LastLogin:   u.LastLogin,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
