package main

import (
	"strings"
	"sync"
)

type StringList struct {
	mu    sync.RWMutex
	items []*Word

	muC    sync.RWMutex
	cached map[string]bool
}

type Word struct {
	Item       string
	SearchType int
}

func (s *StringList) Add(item string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	word := &Word{
		Item:       item,
		SearchType: 0,
	}

	if len(item) > 2 && item[0] == '=' { // ==
		word.Item = item[1:]
		word.SearchType = 1
	} else if len(item) > 2 && item[0] == '<' { // Prefix
		word.Item = item[1:]
		word.SearchType = 2
	} else if len(item) > 2 && item[0] == '>' { // Suffix
		word.Item = item[1:]
		word.SearchType = 3
	}
	//else if len(item) > 2 && item[0] == '!' { // Exclude
	//	word.Item = item[1:]
	//	word.SearchType = 4
	//}

	s.items = append(s.items, word)
}

func (s *StringList) Contains(item string) bool {
	item = strings.TrimSpace(item)

	s.muC.RLock()
	c, ex := s.cached[item]
	s.muC.RUnlock()

	if ex {
		return c
	}

	var is = false
	for _, itm := range s.items {
		if itm.SearchType == 1 {
			if item == itm.Item {
				is = true
				break
			}
		} else if itm.SearchType == 2 {
			if strings.HasPrefix(item, itm.Item) {
				is = true
				break
			}
		} else if itm.SearchType == 3 {
			if strings.HasSuffix(item, itm.Item) {
				is = true
				break
			}
		} else {
			if strings.Contains(item, itm.Item) {
				is = true
				break
			}
		}
	}

	s.muC.Lock()
	s.cached[item] = is
	s.muC.Unlock()

	return is
}
