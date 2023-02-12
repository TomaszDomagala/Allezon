// package userProfilesGc is an unfortunate repercussion of not performing inserts to user profiles in worker.
// It cleans up user profiles that worker encounters.
// If cleaning was done in the same place as insertion we would be able to easily compare new length and remove only when necessary.
// However, cleaning up tags in API server would slow it down.

package userProfilesGc

import (
	"fmt"
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

type GC struct {
	l                   *zap.Logger
	eventChan           chan Event
	gcKeyInterval       time.Duration
	backoff             backoff.BackOff
	userProfilesCleaner UserProfilesCleaner
	actionGCs           map[types.Action]*actionGc
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

func New(deps *Dependencies) GC {
	return GC{
		limit:               deps.Limit,
		l:                   deps.Logger,
		gcKeyInterval:       deps.GcKeyInterval,
		eventChan:           deps.EventChan,
		actionGCs:           make(map[types.Action]*actionGc, 2),
		userProfilesCleaner: deps.UserProfilesCleaner,
		backoff:             deps.Backoff,
	}
}

func (gc *GC) Process(event Event) {
	agc, ok := gc.actionGCs[event.Action]
	if !ok {
		agc = gc.newActionGC(event.Action)
		gc.actionGCs[event.Action] = agc
		agc.start()
	}
	agc.process(event.Cookie)
}

func (gc *GC) Close() {
	for _, agc := range gc.actionGCs {
		agc.close()
	}
}

func (gc *GC) newActionGC(action types.Action) *actionGc {
	deps := actionGcDeps{
		logger:              gc.l,
		action:              action,
		backoff:             gc.backoff,
		limit:               gc.limit,
		userProfilesCleaner: gc.userProfilesCleaner,
		keyInterval:         gc.gcKeyInterval,
	}
	agc, err := newActionGc(&deps)
	if err != nil {
		gc.l.Fatal("error initializing action gc", zap.Error(err))
	}
	return agc
}

type cookieData struct {
	time    time.Time
	visited bool
}

type actionGc struct {
	ch   chan string
	pq   *orderedmap.OrderedMap[string, cookieData]
	pool *ants.PoolWithFunc

	l                   *zap.Logger
	action              types.Action
	backoff             backoff.BackOff
	limit               int
	userProfilesCleaner UserProfilesCleaner
	keyInterval         time.Duration
}

type actionGcDeps struct {
	logger              *zap.Logger
	action              types.Action
	backoff             backoff.BackOff
	limit               int
	userProfilesCleaner UserProfilesCleaner
	keyInterval         time.Duration
}

func newActionGc(deps *actionGcDeps) (agc *actionGc, err error) {
	agc = &actionGc{
		ch:                  make(chan string, cleanerChanSize),
		l:                   deps.logger,
		action:              deps.action,
		limit:               deps.limit,
		backoff:             deps.backoff,
		userProfilesCleaner: deps.userProfilesCleaner,
		keyInterval:         deps.keyInterval,
		pq:                  orderedmap.New[string, cookieData](),
	}

	const routineLimit = 10
	agc.pool, err = ants.NewPoolWithFunc(routineLimit, agc.clean)
	if err != nil {
		return nil, fmt.Errorf("error initializing goroutine pool, %w", err)
	}
	return agc, nil
}

func (agc *actionGc) start() {
	go func() {
		for cookie := range agc.ch {
			if err := agc.processQueue(); err != nil {
				agc.l.Error("error processing gc queue", zap.Error(err))
			}
			if data, ok := agc.pq.Get(cookie); ok {
				if !data.visited {
					agc.pq.Set(cookie, cookieData{time: data.time, visited: true})
				}
				continue
			}
			if err := agc.pool.Invoke(cookie); err != nil {
				agc.l.Error("error cleaning user profiles", zap.Any("cookie", cookie), zap.Error(err))
			} else {
				agc.pq.Set(cookie, cookieData{time: time.Now(), visited: false})
			}
		}
	}()
}

func (agc *actionGc) process(cookie string) {
	agc.ch <- cookie
}

func (agc *actionGc) close() {
	close(agc.ch)
	agc.pool.Release()
}

func (agc *actionGc) processQueue() error {
	pair := agc.pq.Oldest()
	if pair == nil || time.Since(pair.Value.time) > agc.keyInterval {
		return nil
	}
	if pair.Value.visited {
		if err := agc.pool.Invoke(pair.Key); err != nil {
			return fmt.Errorf("error cleaning user profiles, cookie %s, %w", pair.Key, err)
		}
	}
	agc.pq.Delete(pair.Key)
	return nil
}

func (agc *actionGc) clean(c any) {
	cookie := c.(string)
	err := backoff.Retry(func() error {
		if err := agc.userProfilesCleaner(cookie, agc.action, agc.limit); err != nil {
			agc.l.Debug("error cleaning user profiles", zap.String("cookie", cookie), zap.Stringer("action", agc.action), zap.Error(err))
			return err
		}
		return nil
	}, agc.backoff)
	if err != nil {
		agc.l.Error("timeout cleaning user profiles", zap.Any("cookie", cookie), zap.Stringer("action", agc.action), zap.Error(err))
	}
}
