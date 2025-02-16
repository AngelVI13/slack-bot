package spaces

import (
	"fmt"
	"log"
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
	Cancelled  bool
	Submitted  bool
	UniqueId   int
	Active     bool

	// These are only used while the user is choosing date range to refer
	// between space selected and release range selected (i.e. between booking modal
	// and corresponding release modal)
	RootViewId string
	ViewId     string
}

func (i *ReleaseInfo) MarkSubmitted(releaser string) {
	slog.Info("ReleaseInfo Submitted", "releaser", releaser, "info", i)
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

func (i *ReleaseInfo) MarkCancelled() {
	slog.Info("ReleaseInfo Cancelled", "info", i)
	i.Cancelled = true
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

func (i ReleaseInfo) DateRange() string {
	startDateStr := "nil"
	if i.StartDate != nil {
		startDateStr = i.StartDate.Format("2006-01-02")
	}

	endDateStr := "nil"
	if i.EndDate != nil {
		endDateStr = i.EndDate.Format("2006-01-02")
	}
	return fmt.Sprintf("%s -> %s", startDateStr, endDateStr)
}

// TODO: Check for memory leaks afterward whole implementation is done
// ... a lot of dangling pointers around..
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

func (q ReleaseMap) CheckOverlap(release *ReleaseInfo) []string {
	spaceKey := release.Space.Key()
	var overlaps []string

	for _, r := range q.GetAll(spaceKey) {
		if !r.Submitted || r.UniqueId == release.UniqueId {
			continue
		}

		// Overlap on start and end date
		if common.EqualDate(*release.StartDate, *r.StartDate) ||
			common.EqualDate(*release.EndDate, *r.EndDate) ||
			common.EqualDate(*release.StartDate, *r.EndDate) ||
			common.EqualDate(*release.EndDate, *r.StartDate) {

			overlaps = append(overlaps, r.DateRange())
			continue
		}

		// Left overlap i.e. S ------ s ------- E ---- e
		//                   |startA  |startB   |endA  |endb
		if release.StartDate.Before(*r.StartDate) &&
			release.EndDate.After(*r.StartDate) &&
			release.EndDate.Before(*r.EndDate) {

			overlaps = append(overlaps, r.DateRange())
			continue
		}

		// Right overlap i.e. s ------ S ------- e ---- E
		//                    |startB  |startA   |endB  |endA
		if release.StartDate.After(*r.StartDate) &&
			release.StartDate.Before(*r.EndDate) &&
			release.EndDate.After(*r.EndDate) {

			overlaps = append(overlaps, r.DateRange())
			continue
		}

		// Inside overlap i.e. s ------ S ------- E ---- e
		//                     |startB  |startA   |endA  |endB
		if release.StartDate.After(*r.StartDate) &&
			release.StartDate.Before(*r.EndDate) &&
			release.EndDate.Before(*r.EndDate) {

			overlaps = append(overlaps, r.DateRange())
			continue
		}

		// Outside overlap i.e. S ------ s ------- e ---- E
		//                      |startA  |startB   |endB  |endA
		if release.StartDate.Before(*r.StartDate) &&
			release.EndDate.After(*r.EndDate) {

			overlaps = append(overlaps, r.DateRange())
			continue
		}
	}

	return overlaps
}

func (q ReleaseMap) Remove(release *ReleaseInfo) error {
	return q.removeRelease(release.Space.Key(), release.UniqueId)
}

func (q ReleaseMap) removeRelease(spaceKey SpaceKey, id int) error {
	pool, ok := q[spaceKey]
	if !ok {
		return fmt.Errorf("spaceKey not in release map: %q", spaceKey)
	}

	slog.Info("Removing release from release map", "space", spaceKey, "release", id)
	err := pool.Remove(id)
	return err
}

func (q ReleaseMap) RemoveAllReleases(spaceKey SpaceKey) {
	_, found := q[spaceKey]
	if !found {
		return
	}

	delete(q, spaceKey)
	slog.Info("Removing all releases from release map", "space", spaceKey)
}

func (q ReleaseMap) RemoveByViewId(viewId string) (SpaceKey, bool) {
	spaceKey := SpaceKey("")
	for space, pool := range q {
		releaseInfo := pool.ByViewId(viewId)
		if releaseInfo == nil {
			continue
		}

		err := pool.Remove(releaseInfo.UniqueId)
		if err != nil {
			log.Fatalf("failed to remove release by view id: %v", err)
		}
		slog.Info("Removing from release map", "space", spaceKey)
		return space, true
	}

	return spaceKey, false
}

func (q ReleaseMap) Add(
	viewId,
	releaserName,
	releaserId string,
	space *Space,
) *ReleaseInfo {
	spaceKey := space.Key()

	pool, found := q[spaceKey]
	if !found {
		pool = NewReleasePool()
		q[spaceKey] = pool
	}

	// NOTE: if there is an active release then take the owner from that.
	// This is to prevent the following situation
	// 1. Person A temporary releases their space X
	// 2. While space X is temporary released Person B temporary reserves it
	// 3. While space X is temporary reserved by Person B, Person A adds an
	// additional temporary release
	// The following logic ensures that the owner will be taken from the original
	// release and not from the current temporary reserver of the space
	ownerName := space.ReservedBy
	ownerId := space.ReservedById

	active := pool.Active()
	if active != nil {
		ownerName = active.OwnerName
		ownerId = active.OwnerId
	}

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

	q[spaceKey].Put(releaseInfo)
	return releaseInfo
}
