package server

import (
	"JobblyBE/internal/biz"
	"JobblyBE/pkg/kafkax"
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
)

// EvaluateMessage represents the Kafka message for resume evaluation
type EvaluateMessage struct {
	ResumeID string `json:"resume_id"`
	UserID   string `json:"user_id"`
}

// EvaluateWorker handles resume evaluation jobs from Kafka
type EvaluateWorker struct {
	consumer       *kafkax.Consumer
	resumeUC       *biz.ResumeUseCase
	notificationUC *biz.NotificationUseCase
	hub            *Hub
	log            *log.Helper
}

// EvaluateWorkerConfig holds configuration for the evaluate worker
type EvaluateWorkerConfig struct {
	Brokers       []string
	Username      string
	Password      string
	EnableSASL    bool
	Timeout       time.Duration
	Topic         string
	GroupID       string
	NumPartitions int
	NumWorkers    int
}

// NewEvaluateWorker creates a new evaluate worker
func NewEvaluateWorker(
	config *EvaluateWorkerConfig,
	resumeUC *biz.ResumeUseCase,
	notificationUC *biz.NotificationUseCase,
	hub *Hub,
	logger log.Logger,
) *EvaluateWorker {
	consumerConfig := &kafkax.ConsumerConfig{
		Brokers:       config.Brokers,
		Username:      config.Username,
		Password:      config.Password,
		EnableSASL:    config.EnableSASL,
		Timeout:       config.Timeout,
		Topic:         config.Topic,
		GroupID:       config.GroupID,
		NumPartitions: config.NumPartitions,
		NumWorkers:    config.NumWorkers,
	}

	consumer := kafkax.NewConsumer(consumerConfig, logger)

	return &EvaluateWorker{
		consumer:       consumer,
		resumeUC:       resumeUC,
		notificationUC: notificationUC,
		hub:            hub,
		log:            log.NewHelper(logger),
	}
}

// Start starts the evaluate worker
func (w *EvaluateWorker) Start() {
	w.log.Info("Starting resume evaluation worker...")
	w.consumer.Start(w.handleMessage)
}

// Stop stops the evaluate worker
func (w *EvaluateWorker) Stop() error {
	w.log.Info("Stopping resume evaluation worker...")
	return w.consumer.Stop()
}

// handleMessage processes a Kafka message for resume evaluation
func (w *EvaluateWorker) handleMessage(ctx context.Context, msg kafka.Message) error {
	var evaluateMsg EvaluateMessage
	if err := json.Unmarshal(msg.Value, &evaluateMsg); err != nil {
		w.log.Errorf("Failed to unmarshal evaluate message: %v", err)
		return err
	}

	w.log.Infof("Processing resume evaluation: resume_id=%s, user_id=%s",
		evaluateMsg.ResumeID, evaluateMsg.UserID)

	// Call EvaluateAndSaveResume
	updatedResume, err := w.resumeUC.EvaluateAndSaveResume(ctx, evaluateMsg.ResumeID, evaluateMsg.UserID)
	if err != nil {
		w.log.Errorf("Failed to evaluate and save resume: resume_id=%s, error=%v",
			evaluateMsg.ResumeID, err)
		return err
	}

	// Log success with evaluation results
	if updatedResume != nil && updatedResume.ResumeDetail != nil &&
		len(updatedResume.ResumeDetail.Evaluations) > 0 {
		lastEval := updatedResume.ResumeDetail.Evaluations[len(updatedResume.ResumeDetail.Evaluations)-1]
		w.log.Infof("Resume evaluation completed: resume_id=%s, score=%.2f, grade=%s, jobs_analyzed=%d",
			evaluateMsg.ResumeID, lastEval.OverallScore, lastEval.Grade, lastEval.JobsAnalyzed)

		// Create notification
		notification := biz.CreateEvaluationCompleteNotification(
			evaluateMsg.UserID,
			evaluateMsg.ResumeID,
			updatedResume.ResumeDetail.Name,
			lastEval.OverallScore,
			lastEval.Grade,
		)

		savedNotification, err := w.notificationUC.CreateNotification(ctx, notification)
		if err != nil {
			w.log.Errorf("Failed to create notification: %v", err)
		} else {
			// Send WebSocket notification
			w.hub.SendNotification(evaluateMsg.UserID, &NotificationPayload{
				ID:        savedNotification.ID,
				Title:     savedNotification.Title,
				Content:   savedNotification.Content,
				Type:      savedNotification.Type,
				ObjectID:  savedNotification.ObjectID,
				CreatedAt: savedNotification.CreatedAt.Format(time.RFC3339),
			})
			w.log.Infof("Notification sent to user: %s", evaluateMsg.UserID)
		}
	} else {
		w.log.Infof("Resume evaluation completed: resume_id=%s", evaluateMsg.ResumeID)

		// Create basic notification even without detailed evaluation
		notification := &biz.Notification{
			UserID:   evaluateMsg.UserID,
			Title:    "CV đã được đánh giá",
			Content:  "CV của bạn đã được đánh giá. Vui lòng xem chi tiết để biết thêm thông tin.",
			Type:     biz.NotificationTypeEvaluate,
			ObjectID: evaluateMsg.ResumeID,
		}

		savedNotification, err := w.notificationUC.CreateNotification(ctx, notification)
		if err != nil {
			w.log.Errorf("Failed to create notification: %v", err)
		} else {
			// Send WebSocket notification
			w.hub.SendNotification(evaluateMsg.UserID, &NotificationPayload{
				ID:        savedNotification.ID,
				Title:     savedNotification.Title,
				Content:   savedNotification.Content,
				Type:      savedNotification.Type,
				ObjectID:  savedNotification.ObjectID,
				CreatedAt: savedNotification.CreatedAt.Format(time.RFC3339),
			})
			w.log.Infof("Notification sent to user: %s", evaluateMsg.UserID)
		}
	}

	return nil
}
