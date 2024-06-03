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
	capacity int
	data     []*ReleaseInfo
	putNum   int
}

func NewWithCapacity(cap int) (*ReleasePool, error) {
	if cap <= 0 {
		return nil, fmt.Errorf("capacity must be > 0: %d", cap)
	}
	return &ReleasePool{
		capacity: cap,
		data:     make([]*ReleaseInfo, cap),
		putNum:   0,
	}, nil
}

func New() *ReleasePool {
	p, _ := NewWithCapacity(defaultRingBufCapacity)
	return p
}

// freeIdx find first free index in the pool
func (p *ReleasePool) freeIdx() int {
	for i, v := range p.data {
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
	n_copied := copy(new_data, p.data)
	if n_copied != p.capacity {
		log.Fatalf("copied %d but have %d data", n_copied, p.capacity)
	}

	p.data = new_data
	p.capacity = new_size
}

func (p *ReleasePool) Put(v *ReleaseInfo) {
	idx := p.freeIdx()
	if idx == -1 {
		p.grow(2 * p.capacity)
		idx = p.freeIdx()
	}

	v.UniqueId = p.putNum
	p.data[idx] = v
	p.putNum++
}

// Remove replace the first element of pool that matches the provided
// value with an empty value
func (p *ReleasePool) Remove(release *ReleaseInfo) error {
	if release.UniqueId < 0 && release.UniqueId > p.capacity {
		return fmt.Errorf(
			"%w: can't remove release with id=%d from pool with size %d",
			ErrOutOfRange,
			release.UniqueId,
			p.capacity,
		)
	}

	if p.data[release.UniqueId] == nil {
		return fmt.Errorf(
			"%w: can't remove release with id=%d from pool - no value at that idx",
			ErrEmpty,
			release.UniqueId,
		)
	}

	if p.data[release.UniqueId].Space.Key() != release.Space.Key() {
		return fmt.Errorf(
			"%w: can't remove release with id=%d from pool - space mismatch."+
				"Another space is at that localtion: %s vs %s",
			ErrReleaseMismatch,
			release.UniqueId,
			release.Space.Key(),
			p.data[release.UniqueId].Space.Key(),
		)
	}

	p.data[release.UniqueId] = nil
	return nil
}

func (p *ReleasePool) ByRootViewId(id string) (*ReleaseInfo, error) {
	for _, v := range p.data {
		if v != nil && v.RootViewId == id {
			return v, nil
		}
	}
	return nil, ErrNotFound
}

func (p *ReleasePool) ByViewId(id string) (*ReleaseInfo, error) {
	for _, v := range p.data {
		if v != nil && v.ViewId == id {
			return v, nil
		}
	}
	return nil, ErrNotFound
}
