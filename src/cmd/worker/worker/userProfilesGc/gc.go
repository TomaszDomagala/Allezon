// package userProfilesGc is an unfortunate repercussion of not performing inserts to user profiles in worker.
// It cleans up user profiles that worker encounters.
// If cleaning was done in the same place as insertion we would be able to easily compare new length and remove only when necessary.
// However, cleaning up tags in API server would slow it down.

package userProfilesGc

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/panjf2000/ants/v2"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

const cleanerChanSize = 1024

type UserProfilesCleaner = func(cookie string, action types.Action, limit int) error

type Event struct {
	Cookie string
	Action types.Action
}

type UserProfileGC struct {
	l                   *zap.Logger
	eventChan           chan Event
	gcKeyInterval       time.Duration
	backoff             backoff.BackOff
	userProfilesCleaner UserProfilesCleaner
	cleaners            map[types.Action]chan<- Event
	limit               int
}

type Dependencies struct {
	Logger              *zap.Logger
	EventChan           chan Event
	GcKeyInterval       time.Duration
	Backoff             backoff.BackOff
	UserProfilesCleaner UserProfilesCleaner
	Limit               int
}

func New(deps Dependencies) UserProfileGC {
	return UserProfileGC{
		limit:               deps.Limit,
		l:                   deps.Logger,
		gcKeyInterval:       deps.GcKeyInterval,
		eventChan:           deps.EventChan,
		cleaners:            make(map[types.Action]chan<- Event, 2),
		userProfilesCleaner: deps.UserProfilesCleaner,
		backoff:             deps.Backoff,
	}
}

func (gc *UserProfileGC) Run() {
	for event := range gc.eventChan {
		cleaner, ok := gc.cleaners[event.Action]
		if !ok {
			cl := make(chan Event, cleanerChanSize)
			go gc.cleaner(cl)
			cleaner = cl
			gc.cleaners[event.Action] = cleaner
		}
		cleaner <- event
	}
	for _, ch := range gc.cleaners {
		close(ch)
	}
}

func (gc *UserProfileGC) cleaner(ch <-chan Event) {
	pq := orderedmap.New[string, time.Time]()
	const routineLimit = 32

	defer ants.Release()
	p, err := ants.NewPoolWithFunc(routineLimit, gc.clean)
	if err != nil {
		gc.l.Fatal("error initializing goroutine pool", zap.Error(err))
	}

	for event := range ch {
		pair := pq.Oldest()
		if pair != nil && time.Since(pair.Value) < gc.gcKeyInterval {
			if err := p.Invoke(Event{Cookie: pair.Key, Action: event.Action}); err != nil {
				gc.l.Error("error cleaning user profiles", zap.Any("cookie", pair.Key), zap.Error(err))
			} else {
				pq.Delete(pair.Key)
			}
		}
		if _, ok := pq.Get(event.Cookie); ok {
			continue
		}
		if err := p.Invoke(event); err != nil {
			gc.l.Error("error cleaning user profiles", zap.Any("event", event), zap.Error(err))
		} else {
			pq.Set(event.Cookie, time.Now())
		}
	}
}

func (gc *UserProfileGC) clean(ev any) {
	event := ev.(Event)
	err := backoff.Retry(func() error {
		if err := gc.userProfilesCleaner(event.Cookie, event.Action, gc.limit); err != nil {
			gc.l.Debug("error cleaning user profiles for cookie", zap.String("cookie", event.Cookie), zap.Error(err))
			return err
		}
		return nil
	}, gc.backoff)
	if err != nil {
		gc.l.Error("timeout cleaning user profiles for cookie", zap.Any("event", event), zap.Error(err))
	}
}
