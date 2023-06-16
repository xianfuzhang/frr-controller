package rangemanager

import (
	"sync"

	"github.com/guohao117/frr-controller/pkg/number_allocator"
)

type RangeManager struct {
	sync.Mutex
	cache map[string]int
	alloc *numberallocator.Range
}

func NewRangeManager(min, max int) (*RangeManager, error) {
	vnialloc, err := numberallocator.NewRange(min, max)
	if err != nil {
		return nil, err
	}
	return &RangeManager{
		cache: make(map[string]int),
		alloc: vnialloc,
	}, nil

}

func (m *RangeManager) Allocate(name string) (int, error) {
	m.Lock()
	defer m.Unlock()
	if vni, ok := m.cache[name]; ok {
		return vni, nil
	}
	vni, err := m.alloc.AllocateNext()
	if err != nil {
		return 0, err
	}
	m.cache[name] = vni
	return vni, nil
}

func (m *RangeManager) Release(name string) {
	m.Lock()
	defer m.Unlock()
	if vni, ok := m.cache[name]; ok {
		delete(m.cache, name)
		m.alloc.Release(vni)
	}
}

// reserve a vni for a name
func (m *RangeManager) Reserve(name string, vni int) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.cache[name]; ok {
		return nil
	}
	if err := m.alloc.Allocate(vni); err != nil {
		return err
	}
	m.cache[name] = vni
	return nil
}
