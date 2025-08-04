package cache

import (
	"fmt"
	"sync"
)

type CacheKey string

type CachedDocResp struct {
	Status int
	Body   []byte
}

type StructuredCache struct {
	mu sync.RWMutex

	ownerDocs map[string]map[CacheKey]CachedDocResp
	grantDocs map[string]map[string]map[CacheKey]CachedDocResp // grantee -> owner -> token -> resp
}

type Cache interface {
	SetOwner(login string, key CacheKey, value CachedDocResp)
	GetOwner(login string, key CacheKey) (CachedDocResp, bool)
	InvalidateOwnerList(login string)
	SetGrant(grantee, owner string, key CacheKey, value CachedDocResp)
	GetGrant(grantee, owner string, key CacheKey) (CachedDocResp, bool)
	InvalidateGrant(owner string, grantees []string)
}

func NewStructuredCache() *StructuredCache {
	return &StructuredCache{
		ownerDocs: make(map[string]map[CacheKey]CachedDocResp),
		grantDocs: make(map[string]map[string]map[CacheKey]CachedDocResp),
	}
}

func (c *StructuredCache) SetOwner(login string, key CacheKey, value CachedDocResp) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ownerDocs[login] == nil {
		c.ownerDocs[login] = make(map[CacheKey]CachedDocResp)
	}
	c.ownerDocs[login][key] = value
	fmt.Println("serOwner: ", login)
}

func (c *StructuredCache) GetOwner(login string, key CacheKey) (CachedDocResp, bool) {
	fmt.Println("GetOwner: ", login)
	c.mu.RLock()
	defer c.mu.RUnlock()
	m, ok := c.ownerDocs[login]
	if !ok {
		return CachedDocResp{}, false
	}
	v, ok := m[key]
	if ok {
		fmt.Println("ok")
	}
	return v, ok
}

func (c *StructuredCache) InvalidateOwnerList(login string) {
	fmt.Println("InvalidateOwnerList: ", login)
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.ownerDocs[login]; ok {
		fmt.Println("del: ", login)
		delete(c.ownerDocs, login)
	}
}

func (c *StructuredCache) SetGrant(grantee, owner string, key CacheKey, value CachedDocResp) {
	fmt.Println("SetGrant")
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.grantDocs[grantee] == nil {
		c.grantDocs[grantee] = make(map[string]map[CacheKey]CachedDocResp)
	}
	if c.grantDocs[grantee][owner] == nil {
		c.grantDocs[grantee][owner] = make(map[CacheKey]CachedDocResp)
	}
	c.grantDocs[grantee][owner][key] = value
}

func (c *StructuredCache) GetGrant(grantee, owner string, key CacheKey) (CachedDocResp, bool) {
	fmt.Println("getGrant")
	c.mu.RLock()
	defer c.mu.RUnlock()
	m1, ok := c.grantDocs[grantee]
	if !ok {
		return CachedDocResp{}, false
	}
	m2, ok := m1[owner]
	if !ok {
		return CachedDocResp{}, false
	}
	v, ok := m2[key]
	if ok {
		fmt.Println(v)
	}
	return v, ok
}

func (c *StructuredCache) InvalidateGrant(owner string, grantees []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Println("grantes: ", grantees)
	for _, grantee := range grantees {
		if c.grantDocs[grantee] != nil {
			fmt.Println("del list: ", owner)
			delete(c.grantDocs[grantee], owner)
			if len(c.grantDocs[grantee]) == 0 {
				delete(c.grantDocs, grantee)
			}
		}
	}
}
