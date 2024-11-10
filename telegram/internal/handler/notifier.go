package handler

import (
	"context"
	"fmt"

	log "github.com/catalystgo/logger/cli"
	"golang.org/x/sync/errgroup"
)

type notifier struct {
	h *Handler
}

func (n notifier) PrayerSoon(ctx context.Context, chatIDs []int, prayer string, time string) {
	errG, _ := errgroup.WithContext(ctx)
	for _, chatID := range chatIDs {
		errG.Go(func() error {
			err := n.praySoon(chatID, prayer, time)
			if err != nil {
				log.Errorf("Notifier.PrayerSoon: [%d] => %v", chatID, err)
			}
			return nil
		})
	}
}

func (n notifier) praySoon(chatID int, prayer string, time string) error {
	if err := n.h.setScript(chatID); err != nil {
		return err
	}

	script := n.h.getChatScript(chatID)

	message := fmt.Sprintf(script.PrayerSoon, script.GetPrayerByName(prayer), time)
	r, err := n.h.bot.SendMessage(chatID, message, "HTML", 0, false, false)
	if err != nil {
		return err
	}

	go n.h.replace(chatID, r.Result.MessageId)
	return nil
}

func (n notifier) PrayerNow(ctx context.Context, chatIDs []int, prayer string) {
	errG, _ := errgroup.WithContext(ctx)
	for _, chatID := range chatIDs {
		errG.Go(func() error {
			err := n.prayNow(chatID, prayer)
			if err != nil {
				log.Errorf("Notifier.PrayerNow: [%d] => %v", chatID, err)
			}
			return nil
		})
	}
}

func (n notifier) prayNow(chatID int, prayer string) error {
	if err := n.h.setScript(chatID); err != nil {
		return err
	}

	message := fmt.Sprintf(n.h.getChatScript(chatID).PrayerArrived, n.h.getChatScript(chatID).GetPrayerByName(prayer))
	r, err := n.h.bot.SendMessage(chatID, message, "HTML", 0, false, false)
	if err != nil {
		return err
	}

	go n.h.replace(chatID, r.Result.MessageId)
	return nil
}

func (n notifier) PrayerJummah(ctx context.Context, chatIDs []int, time string) {
	errG, _ := errgroup.WithContext(ctx)
	for _, chatID := range chatIDs {
		errG.Go(func() error {
			err := n.prayJummah(chatID, time)
			if err != nil {
				log.Errorf("Notifier.PrayerJummah: [%d] => %v", chatID, err)
			}
			return nil
		})
	}
}

func (n notifier) prayJummah(chatID int, time string) error {
	if err := n.h.setScript(chatID); err != nil {
		return err
	}

	message := fmt.Sprintf(n.h.getChatScript(chatID).GomaaDay, time)
	r, err := n.h.bot.SendMessage(chatID, message, "HTML", 0, false, false)
	if err != nil {
		return err
	}

	go n.h.replace(chatID, r.Result.MessageId)
	return nil
}
