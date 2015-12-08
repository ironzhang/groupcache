/*
Copyright 2012 Google Inc.

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

// Package singleflight provides a duplicate function call suppression
// mechanism.
// singleflight包提供了一种抑制函数重复调用的机制
package singleflight

import "sync"

// call is an in-flight or completed Do call
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group represents a class of work and forms a namespace in which
// units of work can be executed with duplicate suppression.
type Group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call // lazily initialized
}

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
// 该函数的作用是将重复的(key相同)多次函数调用归并为一次函数调用
// 后调的函数会等待前调的函数执行完毕，并使用其结果来作为返回值
// eg.
// 在两个goroutine中并发执行如下代码
// goroutine1
//   val, err := g.Do("iron", LoadUserFromDB)
// goroutine2
//   val, err := g.Do("iron", LoadUserFromDB)
// 只有一个goroutine会真正调用LoadUserFromDB，另一个会等待真正调用的结果，这也是这个包叫singleflight的原因吧 ^_^
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 如果函数已在调用中，则无需执行，等待之前调用的函数执行完成即可
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait() // 等待函数执行完毕
		return c.val, c.err // 返回函数的执行结果
	}
	// 没有在调用中，则新建一个call
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	// 执行函数，并将结果保存在call中
	c.val, c.err = fn()
	c.wg.Done() // 函数执行完毕，解锁其他在等待中的执行流

	// 函数执行完毕，移除call
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	// 返回执行结果
	return c.val, c.err
}
