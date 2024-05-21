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
	Cancelled  bool

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

type ReleaseMap map[SpaceKey]*ReleaseInfo

func (q ReleaseMap) Get(spaceKey SpaceKey) *ReleaseInfo {
	releaseInfo, ok := q[spaceKey]
	if !ok {
		return nil
	}
	return releaseInfo
}

func (q ReleaseMap) GetByReleaserId(userId string) *ReleaseInfo {
	for _, item := range q {
		if item.ReleaserId == userId {
			return item
		}
	}
	return nil
}

func (q ReleaseMap) GetByRootViewId(rootId string) *ReleaseInfo {
	for _, item := range q {
		if item.RootViewId == rootId {
			return item
		}
	}
	return nil
}

func (q ReleaseMap) GetByViewId(viewId string) *ReleaseInfo {
	for _, item := range q {
		if item.ViewId == viewId {
			return item
		}
	}
	return nil
}

func (q ReleaseMap) Remove(spaceKey SpaceKey) bool {
	_, ok := q[spaceKey]
	if !ok {
		return false
	}

	slog.Info("Removing from release map", "space", spaceKey)
	delete(q, spaceKey)
	return true
}

func (q ReleaseMap) RemoveByViewId(viewId string) (SpaceKey, bool) {
	spaceKey := SpaceKey("")
	for space, info := range q {
		if info.ViewId == viewId {
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
	releaserId,
	ownerName,
	ownerId string,
	space *Space,
) (*ReleaseInfo, error) {
	spaceKey := space.Key()
	if q.Get(spaceKey) != nil {
		return nil, fmt.Errorf("Space %s already marked for release", spaceKey)
	}

	slog.Info("Adding to release map", "space", spaceKey)
	releaseInfo := &ReleaseInfo{
		RootViewId: viewId,
		ReleaserId: releaserId,
		OwnerName:  ownerName,
		OwnerId:    ownerId,
		Space:      space,
		Submitted:  false,
	}

	q[spaceKey] = releaseInfo
	return releaseInfo, nil
}
