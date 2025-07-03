package spaces

import (
	"fmt"
	"log"

	"github.com/AngelVI13/slack-bot/pkg/model/my_err"
)

const defaultRingBufCapacity = 10

type ReleasePool struct {
	Capacity int
	Data     []ReleaseInfo
}

func NewReleasePoolWithCapacity(cap int) (*ReleasePool, error) {
	if cap <= 0 {
		return nil, fmt.Errorf("capacity must be > 0: %d", cap)
	}
	return &ReleasePool{
		Capacity: cap,
		Data:     make([]ReleaseInfo, cap),
	}, nil
}

func NewReleasePool() *ReleasePool {
	p, _ := NewReleasePoolWithCapacity(defaultRingBufCapacity)
	return p
}

// freeIdx find first free index in the pool
func (p *ReleasePool) freeIdx() int {
	for i, v := range p.Data {
		if !v.InUse {
			return i
		}
	}

	return -1
}

func (p *ReleasePool) grow(new_size int) {
	// Reallocate the whole array with 2x cap
	new_data := make([]ReleaseInfo, new_size)

	// Realign start to the beginning of the array
	n_copied := copy(new_data, p.Data)
	if n_copied != p.Capacity {
		log.Fatalf("copied %d but have %d data", n_copied, p.Capacity)
	}

	p.Data = new_data
	p.Capacity = new_size
}

func (p *ReleasePool) Add(
	viewId,
	releaserId,
	ownerId,
	ownerName string,
	spaceKey SpaceKey,
) ReleaseInfo {
	idx := p.freeIdx()
	if idx == -1 {
		p.grow(2 * p.Capacity)
		idx = p.freeIdx()
	}
	releaseInfo := NewReleaseInfo(idx, viewId, releaserId, ownerId, ownerName, spaceKey)

	p.Data[idx] = releaseInfo
	return releaseInfo
}

func (p *ReleasePool) Update(release ReleaseInfo) error {
	// TODO: store hash of last Update for a release and then check if when
	// updating a release it was not already updated by someone else
	if !release.InUse {
		return fmt.Errorf(
			"%w: can't update release which is not in use: %v",
			my_err.ErrNotInUse,
			release,
		)
	}
	id := release.UniqueId
	if id < 0 && id > p.Capacity {
		return fmt.Errorf(
			"%w: can't update release with id=%d from pool with size %d",
			my_err.ErrOutOfRange,
			id,
			p.Capacity,
		)
	}

	p.Data[id] = release
	return nil
}

// Remove replace the first element of pool that matches the provided
// value with an empty value
func (p *ReleasePool) Remove(id int) error {
	if id < 0 && id > p.Capacity {
		return fmt.Errorf(
			"%w: can't remove release with id=%d from pool with size %d",
			my_err.ErrOutOfRange,
			id,
			p.Capacity,
		)
	}

	if !p.Data[id].InUse {
		return fmt.Errorf(
			"%w: can't remove release with id=%d from pool - no value at that idx",
			my_err.ErrEmpty,
			id,
		)
	}

	p.Data[id].InUse = false
	return nil
}

func (p *ReleasePool) ByIdx(id int) ReleaseInfo {
	return p.Data[id]
}

func (p *ReleasePool) ByRootViewId(id string) (ReleaseInfo, error) {
	for _, v := range p.Data {
		if v.InUse && v.RootViewId == id {
			return v, nil
		}
	}
	return EmptyRelease, my_err.ErrNotFound
}

func (p *ReleasePool) ByViewId(id string) (ReleaseInfo, error) {
	for _, v := range p.Data {
		if v.InUse && v.ViewId == id {
			return v, nil
		}
	}
	return EmptyRelease, my_err.ErrNotFound
}

func (p *ReleasePool) All() []ReleaseInfo {
	var releases []ReleaseInfo

	for _, release := range p.Data {
		if !release.InUse {
			continue
		}

		releases = append(releases, release)
	}

	return releases
}

func (p *ReleasePool) Active() (ReleaseInfo, error) {
	for _, v := range p.Data {
		if v.InUse && v.Active {
			return v, nil
		}
	}
	return EmptyRelease, my_err.ErrNotFound
}
