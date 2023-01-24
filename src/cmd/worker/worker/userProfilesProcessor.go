package worker

import (
	"errors"
	"fmt"
	"sort"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"go.uber.org/zap"
)

const maxLen = 200

type userProfilesProcessor struct {
	userProfiles db.UserProfileClient
	base         baseProcessor
}

func newUserProfilesProcessor(userProfiles db.UserProfileClient, logger *zap.Logger) userProfilesProcessor {
	u := userProfilesProcessor{
		userProfiles: userProfiles,
	}
	u.base = newBaseProcessor(u.processTagOnce, logger)
	return u
}

func (p userProfilesProcessor) run(tagsChan <-chan types.UserTag) {
	p.base.run(tagsChan)
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
