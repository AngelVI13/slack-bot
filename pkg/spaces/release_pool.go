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
	rb, _ := NewWithCapacity(defaultRingBufCapacity)
	return rb
}

// freeIdx find first free index in the pool
func (rb *ReleasePool) freeIdx() int {
	for i, v := range rb.data {
		if v == nil {
			return i
		}
	}

	return -1
}

func (rb *ReleasePool) grow(new_size int) {
	// Reallocate the whole array with 2x cap
	new_data := make([]*ReleaseInfo, new_size)

	// Realign start to the beginning of the array
	n_copied := copy(new_data, rb.data)
	if n_copied != rb.capacity {
		log.Fatalf("copied %d but have %d data", n_copied, rb.capacity)
	}

	rb.data = new_data
	rb.capacity = new_size
}

func (rb *ReleasePool) Put(v *ReleaseInfo) {
	idx := rb.freeIdx()
	if idx == -1 {
		rb.grow(2 * rb.capacity)
		idx = rb.freeIdx()
	}

	v.UniqueId = rb.putNum
	rb.data[idx] = v
	rb.putNum++
}

// Remove replace the first element of pool that matches the provided
// value with an empty value
func (rb *ReleasePool) Remove(release *ReleaseInfo) error {
	if release.UniqueId < 0 && release.UniqueId > rb.capacity {
		return fmt.Errorf(
			"%w: can't remove release with id=%d from pool with size %d",
			ErrOutOfRange,
			release.UniqueId,
			rb.capacity,
		)
	}

	if rb.data[release.UniqueId] == nil {
		return fmt.Errorf(
			"%w: can't remove release with id=%d from pool - no value at that idx",
			ErrEmpty,
			release.UniqueId,
		)
	}

	if rb.data[release.UniqueId].Space.Key() != release.Space.Key() {
		return fmt.Errorf(
			"%w: can't remove release with id=%d from pool - space mismatch."+
				"Another space is at that localtion: %s vs %s",
			ErrReleaseMismatch,
			release.UniqueId,
			release.Space.Key(),
			rb.data[release.UniqueId].Space.Key(),
		)
	}

	rb.data[release.UniqueId] = nil
	return nil
}

func (rb *ReleasePool) ByRootViewId(id string) (*ReleaseInfo, error) {
	for _, v := range rb.data {
		if v != nil && v.RootViewId == id {
			return v, nil
		}
	}
	return nil, ErrNotFound
}

func (rb *ReleasePool) ByViewId(id string) (*ReleaseInfo, error) {
	for _, v := range rb.data {
		if v != nil && v.ViewId == id {
			return v, nil
		}
	}
	return nil, ErrNotFound
}
