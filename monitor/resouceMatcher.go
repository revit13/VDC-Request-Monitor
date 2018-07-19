package monitor

import (
	"errors"
	"fmt"
	"regexp"
	"sort"

	spec "github.com/DITAS-Project/blueprint-go"
	lru "github.com/hashicorp/golang-lru"
)

type ResouceCache struct {
	cache *lru.TwoQueueCache

	//path(schema):method:optID
	schema      map[string]map[string]string
	pathMatcher []matcher
}

func NewResoruceCache(blueprint *spec.BlueprintType) ResouceCache {
	lfru, _ := lru.New2Q(128)
	//TODO: errohandling?

	cache := ResouceCache{
		cache:       lfru,
		schema:      make(map[string]map[string]string),
		pathMatcher: make([]matcher, 0),
	}

	if blueprint != nil {
		ops := spec.AssembleOperationsMap(*blueprint)

		for k, v := range ops {
			if _, ok := cache.schema[v.Path]; !ok {
				cache.schema[v.Path] = make(map[string]string)
			}
			cache.schema[v.Path][v.Method] = k

			cache.pathMatcher = append(cache.pathMatcher, compile(v.Path))
		}

		sort.Sort(sorter(cache.pathMatcher))

	}

	return cache
}

type matcher struct {
	base   *regexp.Regexp
	soruce string
}
type sorter []matcher

func (a sorter) Len() int           { return len(a) }
func (a sorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sorter) Less(i, j int) bool { return len(a[i].soruce) > len(a[j].soruce) }

var templateMatcher = regexp.MustCompile("{[a-zA-Z0-9\\-_]*}")

func compile(path string) matcher {
	return matcher{
		base:   regexp.MustCompile(templateMatcher.ReplaceAllString(path, "([a-zA-Z0-9\\-_%]*)")),
		soruce: path,
	}
}

func (m *matcher) Match(path string) bool {
	return m.base.MatchString(path)
}

func (rc *ResouceCache) Get(path string, method string) (string, bool) {
	if rc.cache != nil {
		val, ok := rc.cache.Get(fmt.Sprintf("%s%s", method, path))
		if ok {
			return val.(string), ok
		}
	}
	return "", false
}

func (rc *ResouceCache) Add(path string, method string, operationID string) {
	if rc.cache != nil {
		rc.cache.Add(fmt.Sprintf("%s%s", method, path), operationID)
	}
}

func (rc *ResouceCache) Match(path string, method string) (string, error) {

	if val, ok := rc.Get(path, method); ok {
		return val, nil
	}

	for _, m := range rc.pathMatcher {
		if m.Match(path) {
			if optID, ok := rc.schema[m.soruce][method]; ok {
				rc.Add(path, method, optID)
				return optID, nil
			}
		}
	}

	return "", errors.New("no match found in cache")
}
