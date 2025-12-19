package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/icegreg/chat-smpl/services/voice/internal/events"
	"github.com/icegreg/chat-smpl/services/voice/internal/model"
	"github.com/icegreg/chat-smpl/services/voice/internal/repository"
)

// Scheduler handles scheduled tasks like reminders and recurring event generation
type Scheduler struct {
	confRepo       repository.ConferenceRepository
	eventPublisher events.Publisher
	logger         *zap.Logger

	// Configuration
	reminderCheckInterval   time.Duration
	recurringCheckInterval  time.Duration
}

// NewScheduler creates a new scheduler instance
func NewScheduler(
	confRepo repository.ConferenceRepository,
	eventPublisher events.Publisher,
	logger *zap.Logger,
) *Scheduler {
	return &Scheduler{
		confRepo:                confRepo,
		eventPublisher:          eventPublisher,
		logger:                  logger,
		reminderCheckInterval:   1 * time.Minute,
		recurringCheckInterval:  1 * time.Hour,
	}
}

// Start begins the scheduler routines
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("starting scheduler")

	// Start reminder processor
	go s.runReminders(ctx)

	// Start recurring event generator
	go s.runRecurringGenerator(ctx)
}

// runReminders checks for pending reminders every minute
func (s *Scheduler) runReminders(ctx context.Context) {
	ticker := time.NewTicker(s.reminderCheckInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.processReminders(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("reminder processor stopped")
			return
		case <-ticker.C:
			s.processReminders(ctx)
		}
	}
}

// processReminders finds and sends pending reminders
func (s *Scheduler) processReminders(ctx context.Context) {
	now := time.Now()

	reminders, err := s.confRepo.GetPendingReminders(ctx, now)
	if err != nil {
		s.logger.Error("failed to get pending reminders", zap.Error(err))
		return
	}

	if len(reminders) == 0 {
		return
	}

	s.logger.Info("processing reminders", zap.Int("count", len(reminders)))

	for _, reminder := range reminders {
		if err := s.sendReminder(ctx, reminder); err != nil {
			s.logger.Error("failed to send reminder",
				zap.String("reminderId", reminder.ID.String()),
				zap.Error(err))
			continue
		}

		// Mark as sent
		if err := s.confRepo.MarkReminderSent(ctx, reminder.ID); err != nil {
			s.logger.Error("failed to mark reminder as sent",
				zap.String("reminderId", reminder.ID.String()),
				zap.Error(err))
		}
	}
}

// sendReminder publishes reminder event
func (s *Scheduler) sendReminder(ctx context.Context, reminder *model.ConferenceReminder) error {
	// Publish reminder event via RabbitMQ
	if err := s.eventPublisher.PublishConferenceReminder(ctx, reminder); err != nil {
		return err
	}

	s.logger.Info("reminder sent",
		zap.String("conferenceId", reminder.ConferenceID.String()),
		zap.String("userId", reminder.UserID.String()),
		zap.Int("minutesBefore", reminder.MinutesBefore))

	return nil
}

// runRecurringGenerator checks for recurring events and generates instances
func (s *Scheduler) runRecurringGenerator(ctx context.Context) {
	ticker := time.NewTicker(s.recurringCheckInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.generateRecurringInstances(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("recurring generator stopped")
			return
		case <-ticker.C:
			s.generateRecurringInstances(ctx)
		}
	}
}

// generateRecurringInstances creates future instances of recurring events
func (s *Scheduler) generateRecurringInstances(ctx context.Context) {
	// Get recurring conferences that need new instances generated
	// For now, generate instances up to 2 weeks in advance
	s.logger.Debug("checking for recurring events to generate")

	// TODO: Implement full recurring logic
	// 1. Find all recurring conferences with recurrence rules
	// 2. For each, calculate next occurrence based on frequency
	// 3. Create new conference instances if they don't exist
	// 4. Copy participants from original conference
}

// calculateNextOccurrence calculates the next occurrence based on recurrence rule
func (s *Scheduler) calculateNextOccurrence(rule *model.RecurrenceRule, fromTime time.Time) *time.Time {
	if rule == nil {
		return nil
	}

	var next time.Time

	switch rule.Frequency {
	case model.RecurrenceDaily:
		next = fromTime.AddDate(0, 0, 1)

	case model.RecurrenceWeekly:
		if len(rule.DaysOfWeek) == 0 {
			// Same day next week
			next = fromTime.AddDate(0, 0, 7)
		} else {
			// Find next scheduled day of week
			next = s.findNextDayOfWeek(fromTime, rule.DaysOfWeek)
		}

	case model.RecurrenceBiweekly:
		if len(rule.DaysOfWeek) == 0 {
			next = fromTime.AddDate(0, 0, 14)
		} else {
			// Find day of week, then add week if needed
			next = s.findNextDayOfWeek(fromTime, rule.DaysOfWeek)
			// Ensure it's at least a week away for biweekly
			if next.Sub(fromTime) < 7*24*time.Hour {
				next = next.AddDate(0, 0, 7)
			}
		}

	case model.RecurrenceMonthly:
		if rule.DayOfMonth != nil {
			// Specific day of month
			next = time.Date(
				fromTime.Year(), fromTime.Month()+1, *rule.DayOfMonth,
				fromTime.Hour(), fromTime.Minute(), fromTime.Second(), 0, fromTime.Location(),
			)
		} else {
			// Same day next month
			next = fromTime.AddDate(0, 1, 0)
		}
	}

	// Check if we've exceeded the until date or occurrence count
	if rule.UntilDate != nil && next.After(*rule.UntilDate) {
		return nil
	}

	return &next
}

// findNextDayOfWeek finds the next occurrence of specified days of week
func (s *Scheduler) findNextDayOfWeek(from time.Time, daysOfWeek []int) time.Time {
	currentDay := int(from.Weekday())

	// Find the next valid day
	for i := 1; i <= 7; i++ {
		nextDay := (currentDay + i) % 7
		for _, d := range daysOfWeek {
			if d == nextDay {
				return from.AddDate(0, 0, i)
			}
		}
	}

	// Fallback: next week same day
	return from.AddDate(0, 0, 7)
}

// CreateRemindersForConference creates default reminders for all participants
func (s *Scheduler) CreateRemindersForConference(ctx context.Context, conf *model.Conference) error {
	if conf.ScheduledAt == nil {
		return nil
	}

	// Get all participants
	confWithParticipants, err := s.confRepo.GetConferenceWithParticipants(ctx, conf.ID)
	if err != nil {
		return err
	}

	// Create 15-minute reminder for each participant
	for _, p := range confWithParticipants.Participants {
		reminder := &model.ConferenceReminder{
			ID:            uuid.New(),
			ConferenceID:  conf.ID,
			UserID:        p.UserID,
			RemindAt:      conf.ScheduledAt.Add(-15 * time.Minute),
			MinutesBefore: 15,
		}

		if err := s.confRepo.CreateReminder(ctx, reminder); err != nil {
			s.logger.Warn("failed to create reminder",
				zap.String("userId", p.UserID.String()),
				zap.Error(err))
		}
	}

	return nil
}
