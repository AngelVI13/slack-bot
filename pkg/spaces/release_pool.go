package spaces

import (
	"errors"
	"fmt"
	"log"
)

const defaultRingBufCapacity = 10

var (
	ErrEmpty           = errors.New("empty")
	ErrNotFound        = errors.New("notFound")
	ErrOutOfRange      = errors.New("id out of range")
	ErrReleaseMismatch = errors.New("release mismatch")
)

type ReleasePool struct {
	Capacity int
	Data     []*ReleaseInfo
	PutNum   int
}

func NewReleasePoolWithCapacity(cap int) (*ReleasePool, error) {
	if cap <= 0 {
		return nil, fmt.Errorf("capacity must be > 0: %d", cap)
	}
	return &ReleasePool{
		Capacity: cap,
		Data:     make([]*ReleaseInfo, cap),
		PutNum:   0,
	}, nil
}

func NewReleasePool() *ReleasePool {
	p, _ := NewReleasePoolWithCapacity(defaultRingBufCapacity)
	return p
}

// freeIdx find first free index in the pool
func (p *ReleasePool) freeIdx() int {
	for i, v := range p.Data {
		if v == nil {
			return i
		}
	}

	return -1
}

func (p *ReleasePool) grow(new_size int) {
	// Reallocate the whole array with 2x cap
	new_data := make([]*ReleaseInfo, new_size)

	// Realign start to the beginning of the array
	n_copied := copy(new_data, p.Data)
	if n_copied != p.Capacity {
		log.Fatalf("copied %d but have %d data", n_copied, p.Capacity)
	}

	p.Data = new_data
	p.Capacity = new_size
}

func (p *ReleasePool) Put(v *ReleaseInfo) {
	idx := p.freeIdx()
	if idx == -1 {
		p.grow(2 * p.Capacity)
		idx = p.freeIdx()
	}

	v.UniqueId = p.PutNum
	p.Data[idx] = v
	p.PutNum++
}

// Remove replace the first element of pool that matches the provided
// value with an empty value
func (p *ReleasePool) Remove(id int) error {
	if id < 0 && id > p.Capacity {
		return fmt.Errorf(
			"%w: can't remove release with id=%d from pool with size %d",
			ErrOutOfRange,
			id,
			p.Capacity,
		)
	}

	if p.Data[id] == nil {
		return fmt.Errorf(
			"%w: can't remove release with id=%d from pool - no value at that idx",
			ErrEmpty,
			id,
		)
	}

	p.Data[id] = nil
	return nil
}

func (p *ReleasePool) ByIdx(id int) *ReleaseInfo {
	return p.Data[id]
}

func (p *ReleasePool) ByRootViewId(id string) *ReleaseInfo {
	for _, v := range p.Data {
		if v != nil && v.RootViewId == id {
			return v
		}
	}
	return nil
}

func (p *ReleasePool) ByViewId(id string) *ReleaseInfo {
	for _, v := range p.Data {
		if v != nil && v.ViewId == id {
			return v
		}
	}
	return nil
}

func (p *ReleasePool) All() []*ReleaseInfo {
	var releases []*ReleaseInfo

	for _, release := range p.Data {
		if release == nil {
			continue
		}

		releases = append(releases, release)
	}

	return releases
}

func (p *ReleasePool) Active() *ReleaseInfo {
	for _, v := range p.Data {
		if v != nil && v.Active {
			return v
		}
	}
	return nil
}
