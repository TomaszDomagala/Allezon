package worker

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

const maxLen = 200

type userProfilesProcessor struct {
	userProfiles db.UserProfileClient
	logger       *zap.Logger
	config       userProfilesProcessorCfg
}

type userProfilesProcessorCfg struct {
	backoff backoff.ExponentialBackOff
}

func newUserProfilesProcessor(userProfiles db.UserProfileClient, logger *zap.Logger) userProfilesProcessor {
	return userProfilesProcessor{
		userProfiles: userProfiles,
		logger:       logger,
		config: userProfilesProcessorCfg{
			backoff: backoff.ExponentialBackOff{
				InitialInterval:     10 * time.Millisecond,
				RandomizationFactor: backoff.DefaultRandomizationFactor,
				Multiplier:          backoff.DefaultMultiplier,
				MaxInterval:         500 * time.Second,
				MaxElapsedTime:      10 * time.Second,
				Stop:                backoff.Stop,
				Clock:               backoff.SystemClock,
			}},
	}
}

func (p userProfilesProcessor) run(tagsChan <-chan types.UserTag) {
	for tag := range tagsChan {
		if err := p.processTag(tag); err != nil {
			p.logger.Error("error processing user tag", zap.Error(err))
		}
	}
}

func (p userProfilesProcessor) processTag(tag types.UserTag) error {
	bo := p.config.backoff
	bo.Reset()

	err := backoff.Retry(func() error {
		err := p.processTagOnce(tag)
		if err != nil {
			p.logger.Error("error processing user tag", zap.Error(err))
			if !errors.Is(err, db.GenerationMismatch) {
				return fmt.Errorf("error while processing user tag with cookie %s, %w", tag.Cookie, err)
			}
		}
		return nil
	}, &bo)
	if err != nil {
		return fmt.Errorf("error while processing with backoff user tag with cookie %s, %w", tag.Cookie, err)
	}
	return nil
}

func (p userProfilesProcessor) processTagOnce(tag types.UserTag) error {
	up, err := p.userProfiles.Get(tag.Cookie)
	if err != nil && !errors.Is(err, db.KeyNotFoundError) {
		return fmt.Errorf("error getting tag, %w", err)
	}

	var arrPtr *[]types.UserTag
	switch tag.Action {
	case types.Buy:
		arrPtr = &up.Result.Buys
	case types.View:
		arrPtr = &up.Result.Views
	default:
		return fmt.Errorf("unknown action, %d", tag.Action)
	}
	newArr := append(*arrPtr, tag)
	sort.Slice(newArr, func(i, j int) bool { return newArr[i].Time.Before(newArr[j].Time) })
	if len(newArr) > maxLen {
		newArr = newArr[len(newArr)-maxLen:]
	}
	*arrPtr = newArr

	if err := p.userProfiles.Update(tag.Cookie, up.Result, up.Generation); err != nil {
		return fmt.Errorf("error updating tag, %w", err)
	}
	return nil
}
