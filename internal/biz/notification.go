package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// Notification types
const (
	NotificationTypeHR       = "hr"       // HR notification
	NotificationTypeEvaluate = "evaluate" // CV Evaluation notification
)

// Notification represents a user notification
type Notification struct {
	ID        string
	UserID    string
	Title     string
	Content   string
	Type      string
	ObjectID  string // Reference to related object (e.g., resume_id)
	IsRead    bool
	CreatedAt time.Time
}

// NotificationRepo is the interface for notification repository
type NotificationRepo interface {
	CreateNotification(ctx context.Context, notification *Notification) (*Notification, error)
	GetNotificationsByUserID(ctx context.Context, userID string, page, pageSize int32) ([]*Notification, int32, error)
	MarkAsRead(ctx context.Context, notificationID, userID string) error
	MarkAllAsRead(ctx context.Context, userID string) error
	GetUnreadCount(ctx context.Context, userID string) (int32, error)
}

// NotificationUseCase handles notification business logic
type NotificationUseCase struct {
	repo NotificationRepo
	log  *log.Helper
}

// NewNotificationUseCase creates a new notification use case
func NewNotificationUseCase(repo NotificationRepo, logger log.Logger) *NotificationUseCase {
	return &NotificationUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// CreateNotification creates a new notification
func (uc *NotificationUseCase) CreateNotification(ctx context.Context, notification *Notification) (*Notification, error) {
	notification.CreatedAt = time.Now()
	notification.IsRead = false
	return uc.repo.CreateNotification(ctx, notification)
}

// GetUserNotifications gets notifications for a user
func (uc *NotificationUseCase) GetUserNotifications(ctx context.Context, userID string, page, pageSize int32) ([]*Notification, int32, error) {
	return uc.repo.GetNotificationsByUserID(ctx, userID, page, pageSize)
}

// MarkAsRead marks a notification as read
func (uc *NotificationUseCase) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	return uc.repo.MarkAsRead(ctx, notificationID, userID)
}

// MarkAllAsRead marks all notifications as read for a user
func (uc *NotificationUseCase) MarkAllAsRead(ctx context.Context, userID string) error {
	return uc.repo.MarkAllAsRead(ctx, userID)
}

// GetUnreadCount gets the count of unread notifications
func (uc *NotificationUseCase) GetUnreadCount(ctx context.Context, userID string) (int32, error) {
	return uc.repo.GetUnreadCount(ctx, userID)
}

// Notification templates
func CreateEvaluationCompleteNotification(userID, resumeID, cvName string, score float64, grade string) *Notification {
	title := "🎉 CV đã được đánh giá!"
	content := "CV của bạn đã được đánh giá thành công. "
	if cvName != "" {
		content += "CV: " + cvName + ". "
	}
	content += "Điểm đánh giá: " + grade + ". "
	content += "Vui lòng xem chi tiết để biết thêm thông tin và các đề xuất cải thiện."

	return &Notification{
		UserID:   userID,
		Title:    title,
		Content:  content,
		Type:     NotificationTypeEvaluate,
		ObjectID: resumeID,
	}
}
