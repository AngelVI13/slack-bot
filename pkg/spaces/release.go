package spaces

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
)

type ReleaseInfo struct {
	ReleaserId string
	OwnerId    string
	OwnerName  string
	Space      *Space
	StartDate  *time.Time
	EndDate    *time.Time
	Submitted  bool
	UniqueId   int
	Active     bool

	// These are only used while the user is choosing date range to refer
	// between space selected and release range selected (i.e. between booking modal
	// and corresponding release modal)
	RootViewId string
	ViewId     string
}

func (i *ReleaseInfo) MarkSubmitted() {
	slog.Info("ReleaseInfo Submitted", "info", i)
	i.Submitted = true

	// Need to reset view IDs as they are no longer needed.
	// If we don't reset them and user tries to release another
	// space without closing the parent model -> GetByViewId can return
	// incorrect data.
	i.RootViewId = ""
	i.ViewId = ""
}

func (i *ReleaseInfo) MarkActive() {
	slog.Info("ReleaseInfo Active", "info", i)
	i.Active = true
}

func (i *ReleaseInfo) DataPresent() bool {
	return (i.ReleaserId != "" &&
		i.OwnerId != "" &&
		i.OwnerName != "" &&
		i.Space != nil &&
		i.StartDate != nil &&
		i.EndDate != nil)
}

func (i *ReleaseInfo) Check() string {
	if !i.DataPresent() {
		return fmt.Sprintf(
			"Missing date information for temporary release of space (%s)",
			i.Space.Key(),
		)
	}

	return common.CheckDateRange(*i.StartDate, *i.EndDate)
}

func (i ReleaseInfo) String() string {
	startDateStr := "nil"
	if i.StartDate != nil {
		startDateStr = i.StartDate.Format("2006-01-02")
	}

	endDateStr := "nil"
	if i.EndDate != nil {
		endDateStr = i.EndDate.Format("2006-01-02")
	}

	return fmt.Sprintf(
		"ReleaseInfo(space=%s, userName=%s, start=%s, end=%s)",
		i.Space.Key(),
		i.OwnerName,
		startDateStr,
		endDateStr,
	)
}

// TODO: Check for memory leaks afterward whole implementation is done
// ... a lot of dangling pointers around..
// TODO: update logging to reflect new structure
type ReleaseMap map[SpaceKey]*ReleasePool

func (q ReleaseMap) GetAll(spaceKey SpaceKey) []*ReleaseInfo {
	releasePool, ok := q[spaceKey]
	if !ok {
		return nil
	}
	return releasePool.All()
}

func (q ReleaseMap) Get(spaceKey SpaceKey, id int) *ReleaseInfo {
	releasePool, ok := q[spaceKey]
	if !ok {
		return nil
	}
	return releasePool.ByIdx(id)
}

func (q ReleaseMap) GetActive(spaceKey SpaceKey) *ReleaseInfo {
	releasePool, ok := q[spaceKey]
	if !ok {
		return nil
	}
	return releasePool.Active()
}

func (q ReleaseMap) GetByRootViewId(rootId string) *ReleaseInfo {
	for _, pool := range q {
		release := pool.ByRootViewId(rootId)
		if release != nil {
			return release
		}
	}
	return nil
}

func (q ReleaseMap) GetByViewId(viewId string) *ReleaseInfo {
	for _, pool := range q {
		release := pool.ByViewId(viewId)
		if release != nil {
			return release
		}
	}
	return nil
}

func (q ReleaseMap) RemoveRelease(spaceKey SpaceKey, id int) error {
	pool, ok := q[spaceKey]
	if !ok {
		return fmt.Errorf("spaceKey not in release map: %q", spaceKey)
	}

	slog.Info("Removing release from release map", "space", spaceKey, "release", id)
	err := pool.Remove(id)
	return err
}

func (q ReleaseMap) RemoveAllReleases(spaceKey SpaceKey) bool {
	_, ok := q[spaceKey]
	if !ok {
		return false
	}

	delete(q, spaceKey)
	slog.Info("Removing all releases from release map", "space", spaceKey)
	return true
}

func (q ReleaseMap) RemoveByViewId(viewId string) (SpaceKey, bool) {
	spaceKey := SpaceKey("")
	for space, pool := range q {
		if pool.ByViewId(viewId) != nil {
			spaceKey = space
			break
		}
	}
	if spaceKey == "" {
		return spaceKey, false
	}

	slog.Info("Removing from release map", "space", spaceKey)
	delete(q, spaceKey)
	return spaceKey, true
}

func (q ReleaseMap) Add(
	viewId,
	releaserName,
	releaserId,
	ownerName,
	ownerId string,
	space *Space,
) (*ReleaseInfo, error) {
	spaceKey := space.Key()
	slog.Info("Adding to release map",
		"space",
		spaceKey,
		"releaser",
		releaserName,
		"owner",
		ownerName,
	)
	releaseInfo := &ReleaseInfo{
		RootViewId: viewId,
		ReleaserId: releaserId,
		OwnerName:  ownerName,
		OwnerId:    ownerId,
		Space:      space,
		Submitted:  false,
	}

	_, found := q[spaceKey]
	if !found {
		q[spaceKey] = NewReleasePool()
	}

	q[spaceKey].Put(releaseInfo)
	return releaseInfo, nil
}
