package service

import (
	"context"

	pb "JobblyBE/api/notification/v1"
	"JobblyBE/internal/biz"
	"JobblyBE/pkg/middleware/auth"
)

type NotificationService struct {
	pb.UnimplementedNotificationServer

	uc *biz.NotificationUseCase
}

func NewNotificationService(uc *biz.NotificationUseCase) *NotificationService {
	return &NotificationService{uc: uc}
}

func (s *NotificationService) GetNotifications(ctx context.Context, req *pb.GetNotificationsRequest) (*pb.GetNotificationsReply, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, pb.ErrorNotificationNotFound("User not authenticated")
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	notifications, total, err := s.uc.GetUserNotifications(ctx, claims.UserID, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]*pb.NotificationItem, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, &pb.NotificationItem{
			Id:        n.ID,
			UserId:    n.UserID,
			Title:     n.Title,
			Content:   n.Content,
			Type:      n.Type,
			ObjectId:  n.ObjectID,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return &pb.GetNotificationsReply{
		Notifications: items,
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

func (s *NotificationService) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountRequest) (*pb.GetUnreadCountReply, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, pb.ErrorNotificationNotFound("User not authenticated")
	}

	count, err := s.uc.GetUnreadCount(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	return &pb.GetUnreadCountReply{
		Count: count,
	}, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.MarkAsReadReply, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, pb.ErrorNotificationNotFound("User not authenticated")
	}

	err = s.uc.MarkAsRead(ctx, req.NotificationId, claims.UserID)
	if err != nil {
		return nil, err
	}

	return &pb.MarkAsReadReply{
		Success: true,
	}, nil
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, req *pb.MarkAllAsReadRequest) (*pb.MarkAllAsReadReply, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, pb.ErrorNotificationNotFound("User not authenticated")
	}

	err = s.uc.MarkAllAsRead(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	return &pb.MarkAllAsReadReply{
		Success: true,
	}, nil
}
