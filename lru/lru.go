/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package lru implements an LRU cache.
// lur缓存组件
package lru

import "container/list"

// Cache is an LRU cache. It is not safe for concurrent access.
type Cache struct {
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	// 最大缓存对象数
	MaxEntries int

	// OnEvicted optionally specificies a callback function to be
	// executed when an entry is purged from the cache.
	// 移除缓存对象时回调
	OnEvicted func(key Key, value interface{})

	ll    *list.List
	cache map[interface{}]*list.Element
}

// A Key may be any value that is comparable. See http://golang.org/ref/spec#Comparison_operators
type Key interface{}

// 缓存对象，由键与值构成
type entry struct {
	key   Key
	value interface{}
}

// New creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
// 创建一个新的缓存
// 如果maxEntries为0，则缓存对象数量没有上限
func New(maxEntries int) *Cache {
	return &Cache{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
}

// Add adds a value to the cache.
// 添加一个值到缓存
func (c *Cache) Add(key Key, value interface{}) {
	// 支持延迟初始化
	if c.cache == nil {
		c.cache = make(map[interface{}]*list.Element)
		c.ll = list.New()
	}
	// 如果键在缓存中已存在，则将该缓存对象移到缓存队列头部，并更新其值
	if ee, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ee)
		ee.Value.(*entry).value = value
		return
	}
	// 不存在，则构建一个缓存对象并压入缓存队列头部
	ele := c.ll.PushFront(&entry{key, value})
	c.cache[key] = ele // 建立键与缓存对象的映射
	// 如果缓存对象数超过上限，则将最长时间未使用的缓存对象移除
	if c.MaxEntries != 0 && c.ll.Len() > c.MaxEntries {
		c.RemoveOldest()
	}
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key Key) (value interface{}, ok bool) {
	if c.cache == nil {
		return
	}
	// 根据键查找缓存对象
	if ele, hit := c.cache[key]; hit {
		c.ll.MoveToFront(ele) // 将缓存对象移到缓存队列头部
		return ele.Value.(*entry).value, true
	}
	return
}

// Remove removes the provided key from the cache.
// 移除指定的缓存对象
func (c *Cache) Remove(key Key) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele) // 移除缓存对象
	}
}

// RemoveOldest removes the oldest item from the cache.
// 移除最久未使用的缓存对象
func (c *Cache) RemoveOldest() {
	if c.cache == nil {
		return
	}
	ele := c.ll.Back() // 取得缓存队列某位的缓存对象
	if ele != nil {
		c.removeElement(ele) // 移除该对象
	}
}

func (c *Cache) removeElement(e *list.Element) {
	c.ll.Remove(e) // 从缓存列表中移除该元素
	kv := e.Value.(*entry)
	delete(c.cache, kv.key) // 从缓存映射表中移除该缓存对象
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value) // 移除时的回调处理
	}
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	if c.cache == nil {
		return 0
	}
	return c.ll.Len()
}
