package cache

import (
	"container/list"
)

type lru struct {
	size int
	list *list.List
	hash map[string]*list.Element
}

func newLRU(size int) *lru {
	return &lru{
		size: size,
		list: list.New(),
		hash: map[string]*list.Element{},
	}
}

func (l *lru) Has(elem string) bool {
	_, exists := l.hash[elem]
	return exists
}

func (l *lru) Use(elem string) bool {
	e, exists := l.hash[elem]
	if !exists {
		return false
	}
	l.list.MoveToFront(e)
	return true
}

func (l *lru) Put(elem string) (evicted string, eviction bool) {
	if !l.Use(elem) {
		l.hash[elem] = l.list.PushFront(elem)
	}

	if l.list.Len() <= l.size {
		return "", false
	}

	elem = l.list.Remove(l.list.Back()).(string)
	delete(l.hash, elem)
	return elem, true
}

func (l *lru) Remove(elem string) bool {
	e, exists := l.hash[elem]
	if !exists {
		return false
	}
	l.list.Remove(e)
	return true
}

func (l *lru) Save() (rv []string) {
	for e := l.list.Front(); e != nil; e = e.Next() {
		rv = append(rv, e.Value.(string))
	}
	return rv
}

func (l *lru) Load(vals []string) {
	for i := len(vals) - 1; i >= 0; i-- {
		if !l.Use(vals[i]) {
			l.hash[vals[i]] = l.list.PushFront(vals[i])
		}
	}
}
