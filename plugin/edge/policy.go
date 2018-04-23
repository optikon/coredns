/*
 * Copyright 2018 The CoreDNS Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may
 * not use this file except in compliance with the License. You may obtain
 * a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package edge

import (
	"math/rand"
	"sync/atomic"
)

// Policy defines a policy we use for selecting upstreams.
type Policy interface {
	List([]*Proxy) []*Proxy
	String() string
}

// random is a policy that implements random upstream selection.
type random struct{}

func (r *random) String() string { return "random" }

func (r *random) List(p []*Proxy) []*Proxy {
	switch len(p) {
	case 1:
		return p
	case 2:
		if rand.Int()%2 == 0 {
			return []*Proxy{p[1], p[0]} // swap
		}
		return p
	}

	perms := rand.Perm(len(p))
	rnd := make([]*Proxy, len(p))

	for i, p1 := range perms {
		rnd[i] = p[p1]
	}
	return rnd
}

// roundRobin is a policy that selects hosts based on round robin ordering.
type roundRobin struct {
	robin uint32
}

func (r *roundRobin) String() string { return "round_robin" }

func (r *roundRobin) List(p []*Proxy) []*Proxy {
	poolLen := uint32(len(p))
	i := atomic.AddUint32(&r.robin, 1) % poolLen

	robin := []*Proxy{p[i]}
	robin = append(robin, p[:i]...)
	robin = append(robin, p[i+1:]...)

	return robin
}
