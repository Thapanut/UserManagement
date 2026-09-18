package mongodb

import (
	"context"
	"errors"
	"time"

	"backend-challenge/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	usersCollectionName = "users"
)

// MongoUserRepository คือ Implementation ของ domain.UserRepository ที่ต่อกับ MongoDB
type MongoUserRepository struct {
	collection *mongo.Collection
}

// NewMongoUserRepository สร้าง Instance ใหม่ของ MongoUserRepository พร้อมทั้งสร้าง Unique Index บน email
func NewMongoUserRepository(db *mongo.Database) (*MongoUserRepository, error) {
	col := db.Collection(usersCollectionName)
	repo := &MongoUserRepository{
		collection: col,
	}

	// สร้าง Unique Index สำหรับฟิลด์ email ใน MongoDB
	// ช่วยป้องกันข้อมูลอีเมลซ้ำกันที่ระดับ Database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := col.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// Create บันทึกผู้ใช้ใหม่ลง collection
func (r *MongoUserRepository) Create(ctx context.Context, user *domain.User) error {
	// หากยังไม่มี ID ให้สร้าง ObjectID อัตโนมัติ
	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}

	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		// ตรวจสอบกรณี MongoDB Duplicate Key Error (Code 11000) สำหรับ email
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrEmailAlreadyExists
		}
		return err
	}

	return nil
}

// FindByID ค้นหาผู้ใช้จาก ObjectID
func (r *MongoUserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error) {
	var user domain.User
	filter := bson.M{"_id": id}

	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// FindByEmail ค้นหาผู้ใช้จากอีเมล
func (r *MongoUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	filter := bson.M{"email": email}

	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// FindAll ดึงผู้ใช้ทั้งหมดออกมาจากฐานข้อมูล
func (r *MongoUserRepository) FindAll(ctx context.Context) ([]*domain.User, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*domain.User
	for cursor.Next(ctx) {
		var u domain.User
		if err := cursor.Decode(&u); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	// คืน slice ว่างแทน nil เพื่อความสม่ำเสมอของ JSON response
	if users == nil {
		users = []*domain.User{}
	}

	return users, nil
}

// Update อัปเดตข้อมูล Name และ Email ของผู้ใช้
func (r *MongoUserRepository) Update(ctx context.Context, id primitive.ObjectID, name string, email string) (*domain.User, error) {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"name":  name,
			"email": email,
		},
	}

	// ใช้ FindOneAndUpdate พร้อม ReturnDocument = After เพื่อดึงเอกสารล่าสุดกลับมาทันที
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updatedUser domain.User

	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedUser)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrUserNotFound
		}
		if mongo.IsDuplicateKeyError(err) {
			return nil, domain.ErrEmailAlreadyExists
		}
		return nil, err
	}

	return &updatedUser, nil
}

// Delete ลบเอกสารผู้ใช้ออกจากฐานข้อมูล
func (r *MongoUserRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// Count คืนจำนวนเอกสารทั้งหมดใน users collection
func (r *MongoUserRepository) Count(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}
