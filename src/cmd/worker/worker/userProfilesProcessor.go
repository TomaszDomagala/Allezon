package worker

import (
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

const maxLen = 200

// userProfileBackoff is a backoff strategy used to update aggregates.
var userProfileBackoff = backoff.ExponentialBackOff{
	InitialInterval:     10 * time.Millisecond,
	RandomizationFactor: backoff.DefaultRandomizationFactor,
	Multiplier:          backoff.DefaultMultiplier,
	MaxInterval:         500 * time.Second,
	MaxElapsedTime:      10 * time.Second,
	Stop:                backoff.Stop,
	Clock:               backoff.SystemClock,
}

func runUpdateUserProfileProcessor(tagsChan <-chan types.UserTag, userProfiles db.UserProfileClient, logger *zap.Logger) {
	for tag := range tagsChan {
		logger.Debug("[UP] processing tag", zap.Any("tag", tag))
		if err := updateUserProfileBackoff(tag, userProfiles, userProfileBackoff, logger); err != nil {
			logger.Error("error updating user profile", zap.Error(err))
		}
		logger.Debug("[UP] processed tag", zap.Any("tag", tag))
	}
}

func updateUserProfileBackoff(tag types.UserTag, userProfiles db.UserProfileClient, bo backoff.ExponentialBackOff, logger *zap.Logger) error {
	err := backoff.Retry(func() error {
		if err := updateUserProfile(tag, userProfiles); err != nil {
			logger.Debug("[UP] error processing tag", zap.Any("tag", tag), zap.Error(err))
			return err
		}
		return nil
	}, &bo)
	if err != nil {
		return fmt.Errorf("error backoff updating user profile, %w", err)
	}
	return nil
}

func updateUserProfile(tag types.UserTag, userProfiles db.UserProfileClient) error {
	up, err := userProfiles.Get(tag.Cookie)
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
	var newArr []types.UserTag
	for i, t := range *arrPtr {
		if t.Time.Before(tag.Time) {
			newArr = slices.Insert(*arrPtr, i, tag)
			break
		}
	}
	if newArr == nil {
		newArr = append(*arrPtr, tag)
	}
	if len(newArr) > maxLen {
		newArr = newArr[:maxLen]
	}
	*arrPtr = newArr

	if err := userProfiles.Update(tag.Cookie, up.Result, up.Generation); err != nil {
		return fmt.Errorf("error updating tag, %w", err)
	}
	return nil
}
