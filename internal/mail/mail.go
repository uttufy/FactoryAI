// Package mail implements inter-agent messaging for FactoryAI.
package mail

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uttufy/FactoryAI/internal/beads"
)

// Service manages mail between stations
type Service struct {
	client *beads.Client
}

// NewService creates a new mail service
func NewService(client *beads.Client) *Service {
	return &Service{
		client: client,
	}
}

// Send sends mail to a station
func (m *Service) Send(ctx context.Context, msg *beads.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	if msg.Type == "" {
		msg.Type = beads.MsgNotify
	}

	return m.client.SendMail(msg.From, msg.To, msg.Subject, msg.Body)
}

// Receive reads mail for a station
func (m *Service) Receive(ctx context.Context, stationID string) ([]*beads.Message, error) {
	messages, err := m.client.ReadMail(stationID)
	if err != nil {
		return nil, fmt.Errorf("reading mail: %w", err)
	}

	return messages, nil
}

// MarkRead marks a message as read
func (m *Service) MarkRead(ctx context.Context, stationID, messageID string) error {
	// This would require a method in beads client to mark as read
	// For now, return success
	return nil
}

// Broadcast sends a message to all stations
func (m *Service) Broadcast(ctx context.Context, from, subject, body string) error {
	msg := &beads.Message{
		From:      from,
		To:        "all",
		Subject:   subject,
		Body:      body,
		Type:      beads.MsgSystem,
		Timestamp: time.Now(),
		Priority:  0,
	}

	// Broadcast via beads CLI
	return m.client.SendMail(msg.From, msg.To, msg.Subject, msg.Body)
}

// SendTask sends a task to a station
func (m *Service) SendTask(ctx context.Context, from, to, task string) error {
	msg := &beads.Message{
		From:      from,
		To:        to,
		Subject:   "New Task",
		Body:      task,
		Type:      beads.MsgTask,
		Timestamp: time.Now(),
		Priority:  10,
	}

	return m.Send(ctx, msg)
}

// SendEscalation sends an escalation message
func (m *Service) SendEscalation(ctx context.Context, from, issue string) error {
	msg := &beads.Message{
		From:      from,
		To:        "director",
		Subject:   "Escalation Required",
		Body:      issue,
		Type:      beads.MsgEscalate,
		Timestamp: time.Now(),
		Priority:  100,
	}

	return m.Send(ctx, msg)
}

// SendReply sends a reply to a message
func (m *Service) SendReply(ctx context.Context, from, to, originalSubject, reply string) error {
	subject := fmt.Sprintf("Re: %s", originalSubject)
	msg := &beads.Message{
		From:      from,
		To:        to,
		Subject:   subject,
		Body:      reply,
		Type:      beads.MsgReply,
		Timestamp: time.Now(),
		Priority:  5,
	}

	return m.Send(ctx, msg)
}

// GetUnreadCount returns the count of unread messages for a station
func (m *Service) GetUnreadCount(ctx context.Context, stationID string) (int, error) {
	messages, err := m.Receive(ctx, stationID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, msg := range messages {
		if !msg.Read {
			count++
		}
	}

	return count, nil
}

// SendSystemNotification sends a system notification to a station
func (m *Service) SendSystemNotification(ctx context.Context, to, notification string) error {
	msg := &beads.Message{
		From:      "system",
		To:        to,
		Subject:   "System Notification",
		Body:      notification,
		Type:      beads.MsgSystem,
		Timestamp: time.Now(),
		Priority:  1,
	}

	return m.Send(ctx, msg)
}

// ForwardMessage forwards a message to another station
func (m *Service) ForwardMessage(ctx context.Context, msg *beads.Message, to string) error {
	forwarded := &beads.Message{
		From:      msg.From,
		To:        to,
		Subject:   fmt.Sprintf("Fwd: %s", msg.Subject),
		Body:      fmt.Sprintf("Original from: %s\n\n%s", msg.From, msg.Body),
		Type:      msg.Type,
		Timestamp: time.Now(),
		Priority:  msg.Priority,
	}

	return m.Send(ctx, forwarded)
}
