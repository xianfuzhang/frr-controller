package numberallocator

import (
	"errors"
	"fmt"
	"github.com/guohao117/frr-controller/pkg/number_allocator/allocator"
)

var (
	ErrFull      = errors.New("range is full")
	ErrAllocated = errors.New("provided IP is already allocated")
)

type Range struct {
	base  int
	max   int
	alloc allocator.Interface
}

func maximum(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func NewAllocatorRange(base, max int, allocatorFactory allocator.AllocatorFactory) (*Range, error) {
	alloc_max := maximum(0, max-base)
	r := Range{
		base: base,
		max:  max,
	}
	rangeSpec := fmt.Sprintf("%d-%d", base, max)
	var err error
	r.alloc, err = allocatorFactory(alloc_max, rangeSpec)
	return &r, err
}

func NewRange(base, max int) (*Range, error) {
	return NewAllocatorRange(base, max, func(max int, rangeSpec string) (allocator.Interface, error) {
		return allocator.NewContiguousAllocationMap(max, rangeSpec), nil
	})
}

func (r *Range) contains(vni int) (bool, int) {
	if vni < r.base || vni > r.max {
		return false, 0
	}
	return true, vni - r.base
}

func (r *Range) Allocate(vni int) error {
	ok, offset := r.contains(vni)
	if !ok {
		return fmt.Errorf("vni %d is not in range %d-%d", vni, r.base, r.max)
	}

	allocated, err := r.alloc.Allocate(offset)
	if err != nil {
		return err
	}
	if !allocated {
		return ErrAllocated
	}
	return nil
}

// Free returns the count of freed vni
func (r *Range) Free() int {
	return r.alloc.Free()
}

// Used return the count of VNI used in the range
func (r *Range) Used() int {
	return r.max - r.base + 1 - r.alloc.Free()
}

// AllocateNext returns the next available vni in the range
func (r *Range) AllocateNext() (int, error) {
	offset, ok, err := r.alloc.AllocateNext()
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrFull
	}
	return r.base + offset, nil
}

// Release the VNI to the VNI pool
func (r *Range) Release(vni int) {
	ok, offset := r.contains(vni)
	if !ok {
		return
	}
	r.alloc.Release(offset)
}

// ForEach calls the provided function for each allocated VNI in the range
func (r *Range) ForEach(fn func(vni int) error) {
	r.alloc.ForEach(func(offset int) {
		fn(r.base + offset)
	})
}

// Has returns true if the provided VNI is already allocated and a call
func (r *Range) Has(vni int) bool {
	ok, offset := r.contains(vni)
	if !ok {
		return false
	}
	return r.alloc.Has(offset)
}

func (r *Range) Desc() string {
	return fmt.Sprintf("VNI range [%d-%d]", r.base, r.max)
}
