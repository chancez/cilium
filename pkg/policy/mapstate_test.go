// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

//go:build !privileged_tests
// +build !privileged_tests

package policy

import (
	"gopkg.in/check.v1"

	"github.com/cilium/cilium/pkg/checker"
	"github.com/cilium/cilium/pkg/labels"
	"github.com/cilium/cilium/pkg/policy/trafficdirection"
	"github.com/cilium/cilium/pkg/u8proto"
)

// WithOwners replaces owners of 'e' with 'owners'.
// No owners is represented with a 'nil' map.
func (e MapStateEntry) WithOwners(owners ...MapStateOwner) MapStateEntry {
	e.owners = make(map[MapStateOwner]struct{}, len(owners))
	for _, cs := range owners {
		e.owners[cs] = struct{}{}
	}
	return e
}

// WithoutOwners clears the 'owners' of 'e'.
// Note: This is used only in unit tests and helps test readability.
func (e MapStateEntry) WithoutOwners() MapStateEntry {
	e.owners = nil
	return e
}

// WithDependents 'e' adds 'keys' to 'e.dependents'.
func (e MapStateEntry) WithDependents(keys ...Key) MapStateEntry {
	if len(keys) > 0 {
		for _, key := range keys {
			e.AddDependent(key)
		}
	}
	return e
}

func (ds *PolicyTestSuite) TestPolicyKeyTrafficDirection(c *check.C) {
	k := Key{TrafficDirection: trafficdirection.Ingress.Uint8()}
	c.Assert(k.IsIngress(), check.Equals, true)
	c.Assert(k.IsEgress(), check.Equals, false)

	k = Key{TrafficDirection: trafficdirection.Egress.Uint8()}
	c.Assert(k.IsIngress(), check.Equals, false)
	c.Assert(k.IsEgress(), check.Equals, true)
}

func (ds *PolicyTestSuite) TestMapState_DenyPreferredInsert(c *check.C) {
	type args struct {
		key   Key
		entry MapStateEntry
	}
	tests := []struct {
		name       string
		keys, want MapState
		args       args
	}{
		{
			name: "test-1 - no KV added, map should remain the same",
			keys: MapState{
				Key{
					Identity:         0,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: 0,
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
			args: args{
				key:   Key{},
				entry: MapStateEntry{},
			},
			want: MapState{
				Key{
					Identity:         0,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: 0,
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
		},
		{
			name: "test-2 - L3 allow KV should not overwrite deny entry",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			args: args{
				key: Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
			want: MapState{
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
		},
		{
			name: "test-3 - L3-L4 allow KV should not overwrite deny entry",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			args: args{
				key: Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
			want: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
		},
		{
			name: "test-4 - L3-L4 deny KV should overwrite allow entry",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
			args: args{
				key: Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			want: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
		},
		{
			name: "test-5 - L3 deny KV should overwrite all L3-L4 allow and L3 allow entries for the same L3",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         2,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         2,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
			args: args{
				key: Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			want: MapState{
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
				Key{
					Identity:         2,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         2,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
		},
		{
			name: "test-6 - L3 egress deny KV should not overwrite any existing ingress allow",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         2,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         2,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
			args: args{
				key: Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Egress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			want: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Egress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
				Key{
					Identity:         2,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         2,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
		},
		{
			name: "test-7 - L3 ingress deny KV should not be overwritten by a L3-L4 ingress allow",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			args: args{
				key: Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
			want: MapState{
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
		},
		{
			name: "test-8 - L3 ingress deny KV should not be overwritten by a L3-L4-L7 ingress allow",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			args: args{
				key: Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        8080,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
			want: MapState{
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
		},
		{
			name: "test-9 - L3 ingress deny KV should overwrite by a L3-L4-L7 ingress allow",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        8080,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
			args: args{
				key: Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			want: MapState{
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
		},
		{
			name: "test-10 - L3 ingress deny KV should overwrite by a L3-L4-L7 ingress allow and a L3-L4 deny",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        8080,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			args: args{
				key: Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			want: MapState{
				Key{
					Identity:         1,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
		},
		{
			name: "test-11 - L3 ingress allow should not be allowed if there is a L3 'all' deny",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Egress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        8080,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         0,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			args: args{
				key: Key{
					Identity:         100,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
			},
			want: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Egress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        8080,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         0,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
		}, {
			name: "test-12 - inserting a L3 'all' deny should delete all entries for that direction",
			keys: MapState{
				Key{
					Identity:         1,
					DestPort:         80,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        8080,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         1,
					DestPort:         5,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        8080,
					DerivedFromRules: nil,
					IsDeny:           false,
				},
				Key{
					Identity:         100,
					DestPort:         5,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Egress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        8080,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			args: args{
				key: Key{
					Identity:         0,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				},
				entry: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
			want: MapState{
				Key{
					Identity:         0,
					DestPort:         0,
					Nexthdr:          0,
					TrafficDirection: trafficdirection.Ingress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        0,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
				Key{
					Identity:         100,
					DestPort:         5,
					Nexthdr:          3,
					TrafficDirection: trafficdirection.Egress.Uint8(),
				}: MapStateEntry{
					ProxyPort:        8080,
					DerivedFromRules: nil,
					IsDeny:           true,
				},
			},
		},
	}
	for _, tt := range tests {
		tt.keys.DenyPreferredInsert(tt.args.key, tt.args.entry)
		c.Assert(tt.keys, checker.DeepEquals, tt.want, check.Commentf(tt.name))
	}
}

func testKey(id int, port uint16, proto uint8, direction trafficdirection.TrafficDirection) Key {
	return Key{
		Identity:         uint32(id),
		DestPort:         port,
		Nexthdr:          proto,
		TrafficDirection: direction.Uint8(),
	}
}

func testIngressKey(id int, port uint16, proto uint8) Key {
	return testKey(id, port, proto, trafficdirection.Ingress)
}

func testEgressKey(id int, port uint16, proto uint8) Key {
	return testKey(id, port, proto, trafficdirection.Egress)
}

func DNSUDPEgressKey(id int) Key {
	return testEgressKey(id, 53, 17)
}

func DNSTCPEgressKey(id int) Key {
	return testEgressKey(id, 53, 6)
}

func HostIngressKey() Key {
	return testIngressKey(1, 0, 0)
}

func AnyIngressKey() Key {
	return testIngressKey(0, 0, 0)
}

func AnyEgressKey() Key {
	return testEgressKey(0, 0, 0)
}

func HttpIngressKey(id int) Key {
	return testIngressKey(id, 80, 6)
}

func HttpEgressKey(id int) Key {
	return testEgressKey(id, 80, 6)
}

func allowEntry(proxyPort uint16, owners ...MapStateOwner) MapStateEntry {
	return testEntry(proxyPort, false, owners...)
}

func denyEntry(proxyPort uint16, owners ...MapStateOwner) MapStateEntry {
	return testEntry(proxyPort, true, owners...)
}

func testEntry(proxyPort uint16, deny bool, owners ...MapStateOwner) MapStateEntry {
	entry := MapStateEntry{
		ProxyPort: proxyPort,
		IsDeny:    deny,
	}
	if len(owners) > 0 {
		entry.owners = make(map[MapStateOwner]struct{}, len(owners))
	}
	for _, owner := range owners {
		entry.owners[owner] = struct{}{}
	}
	return entry
}

func allowEntryD(proxyPort uint16, derivedFrom labels.LabelArrayList, owners ...MapStateOwner) MapStateEntry {
	return testEntryD(proxyPort, false, derivedFrom, owners...)
}

func denyEntryD(proxyPort uint16, derivedFrom labels.LabelArrayList, owners ...MapStateOwner) MapStateEntry {
	return testEntryD(proxyPort, true, derivedFrom, owners...)
}

func testEntryD(proxyPort uint16, deny bool, derivedFrom labels.LabelArrayList, owners ...MapStateOwner) MapStateEntry {
	entry := testEntry(proxyPort, deny, owners...)
	entry.DerivedFromRules = derivedFrom
	return entry
}

func (ds *PolicyTestSuite) TestMapState_AccumulateMapChangesDeny(c *check.C) {
	csFoo := newTestCachedSelector("Foo", false)
	csBar := newTestCachedSelector("Bar", false)

	type args struct {
		cs       *testCachedSelector
		adds     []int
		deletes  []int
		port     uint16
		proto    uint8
		ingress  bool
		redirect bool
		deny     bool
	}
	tests := []struct {
		continued bool // Start from the end state of the previous test
		name      string
		setup     MapState
		args      []args // changes applied, in order
		state     MapState
		adds      MapState
		deletes   MapState
	}{{
		name: "test-1a - Adding L3-deny to an existing allow-all with L4-only allow redirect map state entries",
		setup: MapState{
			AnyIngressKey():   allowEntry(0),
			HttpIngressKey(0): allowEntry(12345, nil),
		},
		args: []args{
			{cs: csFoo, adds: []int{41}, deletes: []int{}, port: 0, proto: 0, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			AnyIngressKey():          allowEntry(0),
			testIngressKey(41, 0, 0): denyEntry(0, csFoo).WithDependents(HttpIngressKey(41)),
			HttpIngressKey(0):        allowEntry(12345, nil),
			HttpIngressKey(41):       denyEntry(0).WithOwners(testIngressKey(41, 0, 0)),
		},
		adds: MapState{
			testIngressKey(41, 0, 0): denyEntry(0),
			HttpIngressKey(41):       denyEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-1b - Adding 2nd identity",
		args: []args{
			{cs: csFoo, adds: []int{42}, deletes: []int{}, port: 0, proto: 0, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			AnyIngressKey():          allowEntry(0),
			testIngressKey(41, 0, 0): denyEntry(0, csFoo).WithDependents(HttpIngressKey(41)),
			testIngressKey(42, 0, 0): denyEntry(0, csFoo).WithDependents(HttpIngressKey(42)),
			HttpIngressKey(0):        allowEntry(12345, nil),
			HttpIngressKey(41):       denyEntry(0).WithOwners(testIngressKey(41, 0, 0)),
			HttpIngressKey(42):       denyEntry(0).WithOwners(testIngressKey(42, 0, 0)),
		},
		adds: MapState{
			testIngressKey(42, 0, 0): denyEntry(0),
			HttpIngressKey(42):       denyEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-1c - Removing the same key",
		args: []args{
			{cs: csFoo, adds: nil, deletes: []int{42}, port: 0, proto: 0, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			AnyIngressKey():          allowEntry(0),
			testIngressKey(41, 0, 0): denyEntry(0, csFoo).WithDependents(HttpIngressKey(41)),
			HttpIngressKey(0):        allowEntry(12345, nil),
			HttpIngressKey(41):       denyEntry(0).WithOwners(testIngressKey(41, 0, 0)),
		},
		adds: MapState{},
		deletes: MapState{
			testIngressKey(42, 0, 0): denyEntry(0), // removed key
			HttpIngressKey(42):       denyEntry(0), // removed dependent key
		},
	}, {
		name: "test-2a - Adding 2 identities, and deleting a nonexisting key on an empty state",
		args: []args{
			{cs: csFoo, adds: []int{42, 43}, deletes: []int{50}, port: 80, proto: 6, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			HttpIngressKey(42): denyEntry(0, csFoo),
			HttpIngressKey(43): denyEntry(0, csFoo),
		},
		adds: MapState{
			HttpIngressKey(42): denyEntry(0),
			HttpIngressKey(43): denyEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2b - Adding Bar also selecting 42",
		args: []args{
			{cs: csBar, adds: []int{42, 44}, deletes: []int{50}, port: 80, proto: 6, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			HttpIngressKey(42): denyEntry(0, csFoo, csBar),
			HttpIngressKey(43): denyEntry(0, csFoo),
			HttpIngressKey(44): denyEntry(0, csBar),
		},
		adds: MapState{
			HttpIngressKey(44): denyEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2c - Deleting 42 from Foo, remains on Bar and no deletes",
		args: []args{
			{cs: csFoo, adds: []int{}, deletes: []int{42}, port: 80, proto: 6, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			HttpIngressKey(42): denyEntry(0, csBar),
			HttpIngressKey(43): denyEntry(0, csFoo),
			HttpIngressKey(44): denyEntry(0, csBar),
		},
		adds:    MapState{},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2d - Deleting 42 from Foo again, not deleted",
		args: []args{
			{cs: csFoo, adds: []int{}, deletes: []int{42}, port: 80, proto: 6, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			HttpIngressKey(42): denyEntry(0, csBar),
			HttpIngressKey(43): denyEntry(0, csFoo),
			HttpIngressKey(44): denyEntry(0, csBar),
		},
		adds:    MapState{},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2e - Deleting 42 from Bar, deleted",
		args: []args{
			{cs: csBar, adds: []int{}, deletes: []int{42}, port: 80, proto: 6, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			HttpIngressKey(43): denyEntry(0, csFoo),
			HttpIngressKey(44): denyEntry(0, csBar),
		},
		adds: MapState{},
		deletes: MapState{
			HttpIngressKey(42): denyEntry(0),
		},
	}, {
		continued: true,
		name:      "test-2f - Adding an entry that already exists, no adds",
		args: []args{
			{cs: csBar, adds: []int{44}, deletes: []int{}, port: 80, proto: 6, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			HttpIngressKey(43): denyEntry(0, csFoo),
			HttpIngressKey(44): denyEntry(0, csBar),
		},
		adds:    MapState{},
		deletes: MapState{},
	}, {
		continued: false,
		name:      "test-3a - egress allow with deny-L3",
		setup: MapState{
			AnyIngressKey():         allowEntry(0),
			HostIngressKey():        allowEntry(0),
			testEgressKey(42, 0, 0): denyEntry(0, csFoo),
		},
		args: []args{
			{cs: csBar, adds: []int{42}, deletes: []int{}, port: 53, proto: 17, ingress: false, redirect: false, deny: false},
			{cs: csBar, adds: []int{42}, deletes: []int{}, port: 53, proto: 6, ingress: false, redirect: false, deny: false},
		},
		state: MapState{
			AnyIngressKey():         allowEntry(0),
			HostIngressKey():        allowEntry(0),
			testEgressKey(42, 0, 0): denyEntry(0, csFoo),
		},
		adds:    MapState{},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-3b - egress allow DNS on another ID with deny-L3",
		args: []args{
			{cs: csBar, adds: []int{43}, deletes: []int{}, port: 53, proto: 17, ingress: false, redirect: false, deny: false},
			{cs: csBar, adds: []int{43}, deletes: []int{}, port: 53, proto: 6, ingress: false, redirect: false, deny: false},
		},
		state: MapState{
			AnyIngressKey():         allowEntry(0),
			HostIngressKey():        allowEntry(0),
			testEgressKey(42, 0, 0): denyEntry(0, csFoo),
			DNSUDPEgressKey(43):     allowEntry(0, csBar),
			DNSTCPEgressKey(43):     allowEntry(0, csBar),
		},
		adds: MapState{
			DNSUDPEgressKey(43): allowEntry(0),
			DNSTCPEgressKey(43): allowEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-3c - egress allow HTTP proxy with deny-L3",
		args: []args{
			{cs: csFoo, adds: []int{43}, deletes: []int{}, port: 80, proto: 6, ingress: false, redirect: true, deny: false},
		},
		state: MapState{
			AnyIngressKey():         allowEntry(0),
			HostIngressKey():        allowEntry(0),
			testEgressKey(42, 0, 0): denyEntry(0, csFoo),
			DNSUDPEgressKey(43):     allowEntry(0, csBar),
			DNSTCPEgressKey(43):     allowEntry(0, csBar),
			HttpEgressKey(43):       allowEntry(1, csFoo),
		},
		adds: MapState{
			HttpEgressKey(43): allowEntry(1),
		},
		deletes: MapState{},
	}, {
		name: "test-4 - Adding L3-deny to an existing allow-all",
		setup: MapState{
			AnyIngressKey(): allowEntry(0),
		},
		args: []args{
			{cs: csFoo, adds: []int{41}, deletes: []int{}, port: 0, proto: 0, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			AnyIngressKey():          allowEntry(0),
			testIngressKey(41, 0, 0): denyEntry(0, csFoo),
		},
		adds: MapState{
			testIngressKey(41, 0, 0): denyEntry(0),
		},
		deletes: MapState{},
	}, {
		name: "test-5 - Multiple dependent entries",
		setup: MapState{
			AnyEgressKey():     allowEntry(0),
			HttpEgressKey(0):   allowEntry(12345, nil),
			DNSUDPEgressKey(0): allowEntry(12346, nil),
		},
		args: []args{
			{cs: csFoo, adds: []int{41}, deletes: []int{}, port: 0, proto: 0, ingress: false, redirect: false, deny: true},
		},
		state: MapState{
			AnyEgressKey():          allowEntry(0),
			testEgressKey(41, 0, 0): denyEntry(0, csFoo).WithDependents(HttpEgressKey(41), DNSUDPEgressKey(41)),
			HttpEgressKey(0):        allowEntry(12345, nil),
			HttpEgressKey(41):       denyEntry(0).WithOwners(testEgressKey(41, 0, 0)),
			DNSUDPEgressKey(0):      allowEntry(12346, nil),
			DNSUDPEgressKey(41):     denyEntry(0).WithOwners(testEgressKey(41, 0, 0)),
		},
		adds: MapState{
			testEgressKey(41, 0, 0): denyEntry(0),
			HttpEgressKey(41):       denyEntry(0),
			DNSUDPEgressKey(41):     denyEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: false,
		name:      "test-n - title",
		args:      []args{
			//{cs: csFoo, adds: []int{42, 43}, deletes: []int{50}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{
			//HttpIngressKey(42): allowEntry(0, csFoo),
		},
		adds: MapState{
			//HttpIngressKey(42): allowEntry(0),
		},
		deletes: MapState{
			//HttpIngressKey(43): allowEntry(0),
		},
	},
	}

	policyMapState := MapState{}

	for _, tt := range tests {
		policyMaps := MapChanges{}
		if !tt.continued {
			if tt.setup != nil {
				policyMapState = tt.setup
			} else {
				policyMapState = MapState{}
			}
		}
		for _, x := range tt.args {
			dir := trafficdirection.Egress
			if x.ingress {
				dir = trafficdirection.Ingress
			}
			adds := x.cs.addSelections(x.adds...)
			deletes := x.cs.deleteSelections(x.deletes...)
			var cs CachedSelector
			if x.cs != nil {
				cs = x.cs
			}
			policyMaps.AccumulateMapChanges(cs, adds, deletes, x.port, x.proto, dir, x.redirect, x.deny, nil)
		}
		adds, deletes := policyMaps.consumeMapChanges(policyMapState)
		c.Assert(policyMapState, checker.DeepEquals, tt.state, check.Commentf(tt.name+" (MapState)"))
		c.Assert(adds, checker.DeepEquals, tt.adds, check.Commentf(tt.name+" (adds)"))
		c.Assert(deletes, checker.DeepEquals, tt.deletes, check.Commentf(tt.name+" (deletes)"))
	}
}

func (ds *PolicyTestSuite) TestMapState_AccumulateMapChanges(c *check.C) {
	csFoo := newTestCachedSelector("Foo", false)
	csBar := newTestCachedSelector("Bar", false)

	type args struct {
		cs       *testCachedSelector
		adds     []int
		deletes  []int
		port     uint16
		proto    uint8
		ingress  bool
		redirect bool
		deny     bool
	}
	tests := []struct {
		continued bool // Start from the end state of the previous test
		name      string
		args      []args // changes applied, in order
		state     MapState
		adds      MapState
		deletes   MapState
	}{{
		name: "test-1a - Adding identity to an empty state",
		args: []args{
			{cs: csFoo, adds: []int{42}, deletes: []int{}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{
			HttpIngressKey(42): allowEntry(0, csFoo),
		},
		adds: MapState{
			HttpIngressKey(42): allowEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-1b - Removing the sole key",
		args: []args{
			{cs: csFoo, adds: nil, deletes: []int{42}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{},
		adds:  MapState{},
		deletes: MapState{
			HttpIngressKey(42): allowEntry(0),
		},
	}, {
		name: "test-2a - Adding 2 identities, and deleting a nonexisting key on an empty state",
		args: []args{
			{cs: csFoo, adds: []int{42, 43}, deletes: []int{50}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{
			HttpIngressKey(42): allowEntry(0, csFoo),
			HttpIngressKey(43): allowEntry(0, csFoo),
		},
		adds: MapState{
			HttpIngressKey(42): allowEntry(0),
			HttpIngressKey(43): allowEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2b - Adding Bar also selecting 42",
		args: []args{
			{cs: csBar, adds: []int{42, 44}, deletes: []int{50}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{
			HttpIngressKey(42): allowEntry(0, csFoo, csBar),
			HttpIngressKey(43): allowEntry(0, csFoo),
			HttpIngressKey(44): allowEntry(0, csBar),
		},
		adds: MapState{
			HttpIngressKey(44): allowEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2c - Deleting 42 from Foo, remains on Bar and no deletes",
		args: []args{
			{cs: csFoo, adds: []int{}, deletes: []int{42}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{
			HttpIngressKey(42): allowEntry(0, csBar),
			HttpIngressKey(43): allowEntry(0, csFoo),
			HttpIngressKey(44): allowEntry(0, csBar),
		},
		adds:    MapState{},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2d - Deleting 42 from Foo again, not deleted",
		args: []args{
			{cs: csFoo, adds: []int{}, deletes: []int{42}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{
			HttpIngressKey(42): allowEntry(0, csBar),
			HttpIngressKey(43): allowEntry(0, csFoo),
			HttpIngressKey(44): allowEntry(0, csBar),
		},
		adds:    MapState{},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2e - Deleting 42 from Bar, deleted",
		args: []args{
			{cs: csBar, adds: []int{}, deletes: []int{42}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{
			HttpIngressKey(43): allowEntry(0, csFoo),
			HttpIngressKey(44): allowEntry(0, csBar),
		},
		adds: MapState{},
		deletes: MapState{
			HttpIngressKey(42): allowEntry(0),
		},
	}, {
		continued: true,
		name:      "test-2f - Adding an entry that already exists, no adds",
		args: []args{
			{cs: csBar, adds: []int{44}, deletes: []int{}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{
			HttpIngressKey(43): allowEntry(0, csFoo),
			HttpIngressKey(44): allowEntry(0, csBar),
		},
		adds:    MapState{},
		deletes: MapState{},
	}, {
		continued: false,
		name:      "test-3a - egress HTTP proxy (setup)",
		args: []args{
			{cs: nil, adds: []int{0}, deletes: []int{}, port: 0, proto: 0, ingress: true, redirect: false, deny: false},
			{cs: nil, adds: []int{1}, deletes: []int{}, port: 0, proto: 0, ingress: true, redirect: false, deny: false},
			{cs: csBar, adds: []int{42}, deletes: []int{}, port: 53, proto: 17, ingress: false, redirect: false, deny: false},
			{cs: csBar, adds: []int{42}, deletes: []int{}, port: 53, proto: 6, ingress: false, redirect: false, deny: false},
		},
		state: MapState{
			AnyIngressKey():     allowEntry(0, nil),
			HostIngressKey():    allowEntry(0, nil),
			DNSUDPEgressKey(42): allowEntry(0, csBar),
			DNSTCPEgressKey(42): allowEntry(0, csBar),
		},
		adds: MapState{
			AnyIngressKey():     allowEntry(0),
			HostIngressKey():    allowEntry(0),
			DNSUDPEgressKey(42): allowEntry(0),
			DNSTCPEgressKey(42): allowEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-3b - egress HTTP proxy (incremental update)",
		args: []args{
			{cs: csFoo, adds: []int{43}, deletes: []int{}, port: 80, proto: 6, ingress: false, redirect: true, deny: false},
		},
		state: MapState{
			AnyIngressKey():     allowEntry(0, nil),
			HostIngressKey():    allowEntry(0, nil),
			DNSUDPEgressKey(42): allowEntry(0, csBar),
			DNSTCPEgressKey(42): allowEntry(0, csBar),
			HttpEgressKey(43):   allowEntry(1, csFoo),
		},
		adds: MapState{
			HttpEgressKey(43): allowEntry(1),
		},
		deletes: MapState{},
	}, {
		continued: false,
		name:      "test-n - title",
		args:      []args{
			//{cs: csFoo, adds: []int{42, 43}, deletes: []int{50}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{
			//HttpIngressKey(42): allowEntry(0, csFoo),
		},
		adds: MapState{
			//HttpIngressKey(42): allowEntry(0),
		},
		deletes: MapState{
			//HttpIngressKey(43): allowEntry(0),
		},
	},
	}

	policyMapState := MapState{}

	for _, tt := range tests {
		policyMaps := MapChanges{}
		if !tt.continued {
			policyMapState = MapState{}
		}
		for _, x := range tt.args {
			dir := trafficdirection.Egress
			if x.ingress {
				dir = trafficdirection.Ingress
			}
			adds := x.cs.addSelections(x.adds...)
			deletes := x.cs.deleteSelections(x.deletes...)
			var cs CachedSelector
			if x.cs != nil {
				cs = x.cs
			}
			policyMaps.AccumulateMapChanges(cs, adds, deletes, x.port, x.proto, dir, x.redirect, x.deny, nil)
		}
		adds, deletes := policyMaps.consumeMapChanges(policyMapState)
		c.Assert(policyMapState, checker.DeepEquals, tt.state, check.Commentf(tt.name+" (MapState)"))
		c.Assert(adds, checker.DeepEquals, tt.adds, check.Commentf(tt.name+" (adds)"))
		c.Assert(deletes, checker.DeepEquals, tt.deletes, check.Commentf(tt.name+" (deletes)"))
	}
}

var testLabels = labels.LabelArray{
	labels.NewLabel("test", "ing", labels.LabelSourceReserved),
}

func (ds *PolicyTestSuite) TestMapState_AddVisibilityKeys(c *check.C) {
	csFoo := newTestCachedSelector("Foo", false)
	csBar := newTestCachedSelector("Bar", false)

	type args struct {
		redirectPort uint16
		visMeta      VisibilityMetadata
	}
	tests := []struct {
		name       string
		keys, want MapState
		args       args
	}{
		{
			name: "test-1 - Add HTTP ingress visibility - allow-all",
			keys: MapState{
				AnyIngressKey(): allowEntry(0),
			},
			args: args{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			want: MapState{
				AnyIngressKey():   allowEntry(0),
				HttpIngressKey(0): allowEntryD(12345, visibilityDerivedFrom, nil),
			},
		},
		{
			name: "test-2 - Add HTTP ingress visibility - no allow-all",
			keys: MapState{},
			args: args{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			want: MapState{},
		},
		{
			name: "test-3 - Add HTTP ingress visibility - L4-allow",
			keys: MapState{
				HttpIngressKey(0): allowEntryD(0, labels.LabelArrayList{testLabels}),
			},
			args: args{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			want: MapState{
				HttpIngressKey(0): allowEntryD(12345, labels.LabelArrayList{testLabels, visibilityDerivedFromLabels}),
			},
		},
		{
			name: "test-4 - Add HTTP ingress visibility - L3/L4-allow",
			keys: MapState{
				HttpIngressKey(123): allowEntryD(0, labels.LabelArrayList{testLabels}, csBar),
			},
			args: args{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			want: MapState{
				HttpIngressKey(123): allowEntryD(12345, labels.LabelArrayList{testLabels, visibilityDerivedFromLabels}, csBar),
			},
		},
		{
			name: "test-5 - Add HTTP ingress visibility - L3-allow (host)",
			keys: MapState{
				HostIngressKey(): allowEntry(0),
			},
			args: args{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			want: MapState{
				HostIngressKey():  allowEntry(0).WithDependents(HttpIngressKey(1)),
				HttpIngressKey(1): allowEntryD(12345, labels.LabelArrayList{visibilityDerivedFromLabels}).WithOwners(HostIngressKey()),
			},
		},
		{
			name: "test-6 - Add HTTP ingress visibility - L3/L4-allow on different port",
			keys: MapState{
				testIngressKey(123, 88, 6): allowEntryD(0, labels.LabelArrayList{testLabels}, csBar),
			},
			args: args{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			want: MapState{
				testIngressKey(123, 88, 6): allowEntryD(0, labels.LabelArrayList{testLabels}, csBar),
			},
		},
		{
			name: "test-7 - Add HTTP ingress visibility - allow-all + L4-deny (no change)",
			keys: MapState{
				AnyIngressKey():   allowEntry(0),
				HttpIngressKey(0): denyEntry(0),
			},
			args: args{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			want: MapState{
				AnyIngressKey():   allowEntry(0),
				HttpIngressKey(0): denyEntry(0),
			},
		},
		{
			name: "test-8 - Add HTTP ingress visibility - allow-all + L3-deny",
			keys: MapState{
				AnyIngressKey():           allowEntry(0),
				testIngressKey(234, 0, 0): denyEntry(0, csFoo),
			},
			args: args{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			want: MapState{
				AnyIngressKey():           allowEntry(0),
				testIngressKey(234, 0, 0): denyEntry(0, csFoo).WithDependents(HttpIngressKey(234)),
				HttpIngressKey(0):         allowEntryD(12345, visibilityDerivedFrom, nil),
				HttpIngressKey(234):       denyEntry(0, csFoo).WithOwners(testIngressKey(234, 0, 0)),
			},
		},
		{
			name: "test-9 - Add HTTP ingress visibility - allow-all + L3/L4-deny",
			keys: MapState{
				AnyIngressKey():     allowEntry(0),
				HttpIngressKey(132): denyEntry(0, csBar),
			},
			args: args{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			want: MapState{
				AnyIngressKey():     allowEntry(0),
				HttpIngressKey(132): denyEntry(0, csBar),
				HttpIngressKey(0):   allowEntryD(12345, visibilityDerivedFrom, nil),
			},
		},
		{
			name: "test-10 - Add HTTP egress visibility",
			keys: MapState{
				AnyEgressKey(): allowEntry(0),
			},
			args: args{
				redirectPort: 12346,
				visMeta:      VisibilityMetadata{Ingress: false, Port: 80, Proto: u8proto.TCP},
			},
			want: MapState{
				AnyEgressKey():   allowEntry(0),
				HttpEgressKey(0): allowEntryD(12346, visibilityDerivedFrom, nil),
			},
		},
	}
	for _, tt := range tests {
		old := make(MapState, len(tt.keys))
		for k, v := range tt.keys {
			old[k] = v
		}
		adds := make(MapState)
		deletes := make(MapState)
		tt.keys.AddVisibilityKeys(DummyOwner{}, tt.args.redirectPort, &tt.args.visMeta, adds, deletes)
		c.Assert(tt.keys, checker.DeepEquals, tt.want, check.Commentf(tt.name))
		// Find new and updated entries
		wantAdds := make(MapState)
		wantDeletes := make(MapState)
		for k, v := range old {
			if _, ok := tt.keys[k]; !ok {
				wantDeletes[k] = v.export()
			}
		}
		for k, v := range tt.keys {
			if v2, ok := old[k]; ok {
				v = v.export()
				v2 = v2.export()
				if equals, _ := checker.DeepEqual(v2, v); !equals {
					wantAdds[k] = v
					wantDeletes[k] = v2
				}
			} else {
				wantAdds[k] = v.export()
			}
		}
		c.Assert(adds, checker.DeepEquals, wantAdds, check.Commentf(tt.name))
		c.Assert(deletes, checker.DeepEquals, wantDeletes, check.Commentf(tt.name))
	}
}

func (ds *PolicyTestSuite) TestMapState_AccumulateMapChangesOnVisibilityKeys(c *check.C) {
	csFoo := newTestCachedSelector("Foo", false)
	csBar := newTestCachedSelector("Bar", false)

	type args struct {
		cs       *testCachedSelector
		adds     []int
		deletes  []int
		port     uint16
		proto    uint8
		ingress  bool
		redirect bool
		deny     bool
	}
	type visArgs struct {
		redirectPort uint16
		visMeta      VisibilityMetadata
	}
	tests := []struct {
		continued  bool // Start from the end state of the previous test
		name       string
		setup      MapState
		visArgs    []visArgs
		visAdds    MapState
		visDeletes MapState
		args       []args // changes applied, in order
		state      MapState
		adds       MapState
		deletes    MapState
	}{{
		name: "test-1a - Adding identity to deny with visibilty",
		setup: MapState{
			AnyIngressKey():           allowEntry(0),
			testIngressKey(234, 0, 0): denyEntry(0, csFoo),
		},
		visArgs: []visArgs{{
			redirectPort: 12345,
			visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
		}},
		visAdds: MapState{
			HttpIngressKey(0):   allowEntryD(12345, visibilityDerivedFrom),
			HttpIngressKey(234): denyEntry(0),
		},
		args: []args{
			{cs: csFoo, adds: []int{235}, deletes: []int{}, port: 0, proto: 0, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			AnyIngressKey():           allowEntry(0),
			testIngressKey(234, 0, 0): denyEntry(0, csFoo).WithDependents(HttpIngressKey(234)),
			testIngressKey(235, 0, 0): denyEntry(0, csFoo).WithDependents(HttpIngressKey(235)),
			HttpIngressKey(0):         allowEntryD(12345, visibilityDerivedFrom, nil),
			HttpIngressKey(234):       denyEntry(0).WithOwners(testIngressKey(234, 0, 0)),
			HttpIngressKey(235):       denyEntry(0).WithOwners(testIngressKey(235, 0, 0)),
		},
		adds: MapState{
			testIngressKey(235, 0, 0): denyEntry(0),
			HttpIngressKey(235):       denyEntry(0),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-1b - Removing the sole key",
		args: []args{
			{cs: csFoo, adds: nil, deletes: []int{235}, port: 0, proto: 0, ingress: true, redirect: false, deny: true},
		},
		state: MapState{
			AnyIngressKey():           allowEntry(0),
			testIngressKey(234, 0, 0): denyEntry(0, csFoo).WithDependents(HttpIngressKey(234)),
			HttpIngressKey(0):         allowEntryD(12345, visibilityDerivedFrom, nil),
			HttpIngressKey(234):       denyEntry(0).WithOwners(testIngressKey(234, 0, 0)),
		},
		adds: MapState{},
		deletes: MapState{
			testIngressKey(235, 0, 0): denyEntry(0),
			HttpIngressKey(235):       denyEntry(0),
		},
	}, {
		name: "test-2a - Adding 2 identities, and deleting a nonexisting key on an empty state",
		args: []args{
			{cs: csFoo, adds: []int{235, 236}, deletes: []int{50}, port: 0, proto: 0, ingress: true, redirect: false, deny: false},
		},
		visArgs: []visArgs{{
			redirectPort: 12345,
			visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
		}},
		state: MapState{
			testIngressKey(235, 0, 0): allowEntry(0, csFoo).WithDependents(HttpIngressKey(235)),
			testIngressKey(236, 0, 0): allowEntry(0, csFoo).WithDependents(HttpIngressKey(236)),
			HttpIngressKey(235):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(235, 0, 0)),
			HttpIngressKey(236):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(236, 0, 0)),
		},
		adds: MapState{
			testIngressKey(235, 0, 0): allowEntry(0),
			HttpIngressKey(235):       allowEntryD(12345, visibilityDerivedFrom),
			testIngressKey(236, 0, 0): allowEntry(0),
			HttpIngressKey(236):       allowEntryD(12345, visibilityDerivedFrom),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2b - Adding Bar also selecting 235",
		args: []args{
			{cs: csBar, adds: []int{235, 237}, deletes: []int{50}, port: 0, proto: 0, ingress: true, redirect: false, deny: false},
		},
		visArgs: []visArgs{{
			redirectPort: 12345,
			visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
		}},
		state: MapState{
			testIngressKey(235, 0, 0): allowEntry(0, csFoo, csBar).WithDependents(HttpIngressKey(235)),
			testIngressKey(236, 0, 0): allowEntry(0, csFoo).WithDependents(HttpIngressKey(236)),
			HttpIngressKey(235):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(235, 0, 0)),
			HttpIngressKey(236):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(236, 0, 0)),
			testIngressKey(237, 0, 0): allowEntry(0, csBar).WithDependents(HttpIngressKey(237)),
			HttpIngressKey(237):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(237, 0, 0)),
		},
		adds: MapState{
			testIngressKey(237, 0, 0): allowEntry(0),
			HttpIngressKey(237):       allowEntryD(12345, visibilityDerivedFrom),
		},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2c - Deleting 235 from Foo, remains on Bar and no deletes",
		args: []args{
			{cs: csFoo, adds: []int{}, deletes: []int{235}, port: 0, proto: 0, ingress: true, redirect: false, deny: true},
		},
		visArgs: []visArgs{{
			redirectPort: 12345,
			visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
		}},
		state: MapState{
			testIngressKey(235, 0, 0): allowEntry(0, csBar).WithDependents(HttpIngressKey(235)),
			testIngressKey(236, 0, 0): allowEntry(0, csFoo).WithDependents(HttpIngressKey(236)),
			HttpIngressKey(235):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(235, 0, 0)),
			HttpIngressKey(236):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(236, 0, 0)),
			testIngressKey(237, 0, 0): allowEntry(0, csBar).WithDependents(HttpIngressKey(237)),
			HttpIngressKey(237):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(237, 0, 0)),
		},
		adds:    MapState{},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2d - Deleting 235 from Foo again, not deleted",
		args: []args{
			{cs: csFoo, adds: []int{}, deletes: []int{235}, port: 0, proto: 0, ingress: true, redirect: false, deny: true},
		},
		visArgs: []visArgs{{
			redirectPort: 12345,
			visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
		}},
		state: MapState{
			testIngressKey(235, 0, 0): allowEntry(0, csBar).WithDependents(HttpIngressKey(235)),
			testIngressKey(236, 0, 0): allowEntry(0, csFoo).WithDependents(HttpIngressKey(236)),
			HttpIngressKey(235):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(235, 0, 0)),
			HttpIngressKey(236):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(236, 0, 0)),
			testIngressKey(237, 0, 0): allowEntry(0, csBar).WithDependents(HttpIngressKey(237)),
			HttpIngressKey(237):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(237, 0, 0)),
		},
		adds:    MapState{},
		deletes: MapState{},
	}, {
		continued: true,
		name:      "test-2e - Deleting 235 from Bar, deleted",
		args: []args{
			{cs: csBar, adds: []int{}, deletes: []int{235}, port: 0, proto: 0, ingress: true, redirect: false, deny: true},
		},
		visArgs: []visArgs{{
			redirectPort: 12345,
			visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
		}},
		state: MapState{
			testIngressKey(236, 0, 0): allowEntry(0, csFoo).WithDependents(HttpIngressKey(236)),
			HttpIngressKey(236):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(236, 0, 0)),
			testIngressKey(237, 0, 0): allowEntry(0, csBar).WithDependents(HttpIngressKey(237)),
			HttpIngressKey(237):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(237, 0, 0)),
		},
		adds: MapState{},
		deletes: MapState{
			testIngressKey(235, 0, 0): allowEntry(0),
			HttpIngressKey(235):       allowEntryD(12345, visibilityDerivedFrom),
		},
	}, {
		continued: true,
		name:      "test-2f - Adding an entry that already exists, no adds",
		args: []args{
			{cs: csBar, adds: []int{237}, deletes: []int{}, port: 0, proto: 0, ingress: true, redirect: false, deny: false},
		},
		visArgs: []visArgs{{
			redirectPort: 12345,
			visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
		}},
		state: MapState{
			testIngressKey(236, 0, 0): allowEntry(0, csFoo).WithDependents(HttpIngressKey(236)),
			HttpIngressKey(236):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(236, 0, 0)),
			testIngressKey(237, 0, 0): allowEntry(0, csBar).WithDependents(HttpIngressKey(237)),
			HttpIngressKey(237):       allowEntryD(12345, visibilityDerivedFrom).WithOwners(testIngressKey(237, 0, 0)),
		},
		adds:    MapState{},
		deletes: MapState{},
	}, {
		continued: false,
		name:      "test-3a - egress HTTP proxy (setup)",
		setup: MapState{
			AnyIngressKey():  allowEntry(0),
			HostIngressKey(): allowEntry(0),
			HttpEgressKey(0): allowEntry(0),
		},
		visArgs: []visArgs{
			{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			{
				redirectPort: 12346,
				visMeta:      VisibilityMetadata{Ingress: false, Port: 80, Proto: u8proto.TCP},
			},
			{
				redirectPort: 12347,
				visMeta:      VisibilityMetadata{Ingress: false, Port: 53, Proto: u8proto.UDP},
			},
		},
		visAdds: MapState{
			HttpIngressKey(0): allowEntryD(12345, visibilityDerivedFrom),
			HttpEgressKey(0):  allowEntryD(12346, visibilityDerivedFrom),
		},
		visDeletes: MapState{
			// Old value for the modified entry
			HttpEgressKey(0): allowEntry(0),
		},
		args: []args{
			{cs: csBar, adds: []int{42}, deletes: []int{}, port: 53, proto: 17, ingress: false, redirect: false, deny: false},
			{cs: csBar, adds: []int{42}, deletes: []int{}, port: 53, proto: 6, ingress: false, redirect: false, deny: false},
		},
		state: MapState{
			AnyIngressKey():  allowEntry(0),
			HostIngressKey(): allowEntry(0),
			// Entry added solely due to visibility annotation has a 'nil' owner
			HttpIngressKey(0): allowEntryD(12345, visibilityDerivedFrom).WithOwners(nil),
			// Entries modified due to visibility annotation keep their existing owners (here none)
			HttpEgressKey(0):    allowEntryD(12346, visibilityDerivedFrom),
			DNSUDPEgressKey(42): allowEntryD(12347, visibilityDerivedFrom, csBar),
			DNSTCPEgressKey(42): allowEntry(0, csBar),
		},
		adds: MapState{
			DNSUDPEgressKey(42): allowEntryD(12347, visibilityDerivedFrom),
			DNSTCPEgressKey(42): allowEntry(0),
		},
		deletes: MapState{
			// AddVisibilityKeys() returns overwritten entries in 'deletes'
			DNSUDPEgressKey(42): allowEntry(0),
		},
	}, {
		continued: true,
		name:      "test-3b - egress HTTP proxy (incremental update)",
		args: []args{
			{cs: csFoo, adds: []int{43}, deletes: []int{}, port: 80, proto: 6, ingress: false, redirect: true, deny: false},
		},
		visArgs: []visArgs{
			{
				redirectPort: 12345,
				visMeta:      VisibilityMetadata{Ingress: true, Port: 80, Proto: u8proto.TCP},
			},
			{
				redirectPort: 12346,
				visMeta:      VisibilityMetadata{Ingress: false, Port: 80, Proto: u8proto.TCP},
			},
			{
				redirectPort: 12347,
				visMeta:      VisibilityMetadata{Ingress: false, Port: 53, Proto: u8proto.UDP},
			},
		},
		state: MapState{
			AnyIngressKey():     allowEntry(0),
			HostIngressKey():    allowEntry(0),
			HttpIngressKey(0):   allowEntryD(12345, visibilityDerivedFrom).WithOwners(nil),
			HttpEgressKey(0):    allowEntryD(12346, visibilityDerivedFrom),
			DNSUDPEgressKey(42): allowEntryD(12347, visibilityDerivedFrom, csBar),
			DNSTCPEgressKey(42): allowEntry(0, csBar),
			// Redirect entries are not modified by visibility annotations
			HttpEgressKey(43): allowEntry(1, csFoo),
		},
		adds: MapState{
			HttpEgressKey(43): allowEntry(1),
		},
		deletes: MapState{},
	}, {
		continued: false,
		name:      "test-n - title",
		args:      []args{
			//{cs: csFoo, adds: []int{42, 43}, deletes: []int{50}, port: 80, proto: 6, ingress: true, redirect: false, deny: false},
		},
		state: MapState{
			//HttpIngressKey(42): allowEntry(0, csFoo),
		},
		adds: MapState{
			//HttpIngressKey(42): allowEntry(0),
		},
		deletes: MapState{
			//HttpIngressKey(43): allowEntry(0),
		},
	},
	}

	policyMapState := MapState{}

	for _, tt := range tests {
		// Allow omit empty maps
		if tt.visAdds == nil {
			tt.visAdds = make(MapState)
		}
		if tt.visDeletes == nil {
			tt.visDeletes = make(MapState)
		}
		if tt.adds == nil {
			tt.adds = make(MapState)
		}
		if tt.deletes == nil {
			tt.deletes = make(MapState)
		}
		policyMaps := MapChanges{}
		if !tt.continued {
			if tt.setup != nil {
				policyMapState = tt.setup
			} else {
				policyMapState = MapState{}
			}
		}
		adds := make(MapState)
		deletes := make(MapState)
		for _, arg := range tt.visArgs {
			policyMapState.AddVisibilityKeys(DummyOwner{}, arg.redirectPort, &arg.visMeta, adds, deletes)
		}
		c.Assert(adds, checker.DeepEquals, tt.visAdds, check.Commentf(tt.name+" (visAdds)"))
		c.Assert(deletes, checker.DeepEquals, tt.visDeletes, check.Commentf(tt.name+" (visDeletes)"))
		for _, x := range tt.args {
			dir := trafficdirection.Egress
			if x.ingress {
				dir = trafficdirection.Ingress
			}
			adds := x.cs.addSelections(x.adds...)
			deletes := x.cs.deleteSelections(x.deletes...)
			var cs CachedSelector
			if x.cs != nil {
				cs = x.cs
			}
			policyMaps.AccumulateMapChanges(cs, adds, deletes, x.port, x.proto, dir, x.redirect, x.deny, nil)
		}
		adds, deletes = policyMaps.consumeMapChanges(policyMapState)
		// Visibilty redirects need to be re-applied after consumeMapChanges()
		for _, arg := range tt.visArgs {
			policyMapState.AddVisibilityKeys(DummyOwner{}, arg.redirectPort, &arg.visMeta, adds, deletes)
		}
		c.Assert(policyMapState, checker.DeepEquals, tt.state, check.Commentf(tt.name+" (MapState)"))
		c.Assert(adds, checker.DeepEquals, tt.adds, check.Commentf(tt.name+" (adds)"))
		c.Assert(deletes, checker.DeepEquals, tt.deletes, check.Commentf(tt.name+" (deletes)"))
	}
}
