// Copyright 2016 The etcd-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package etcdutil

import "testing"

func TestMemberSetIsEqual(t *testing.T) {
	ma := &Member{Name: "a"}
	mb := &Member{Name: "b"}
	tests := []struct {
		ms1, ms2 MemberSet
		wEqual   bool
	}{{
		ms1:    NewMemberSet(mb, ma),
		ms2:    NewMemberSet(mb, ma),
		wEqual: true,
	}, {
		ms1:    NewMemberSet(ma, mb),
		ms2:    NewMemberSet(ma),
		wEqual: false,
	}, {
		ms1:    NewMemberSet(ma),
		ms2:    NewMemberSet(ma, mb),
		wEqual: false,
	}, {
		ms1:    NewMemberSet(),
		ms2:    NewMemberSet(),
		wEqual: true,
	}, {
		ms1:    NewMemberSet(),
		ms2:    NewMemberSet(ma),
		wEqual: false,
	}, {
		ms1:    NewMemberSet(ma),
		ms2:    NewMemberSet(),
		wEqual: false,
	}}
	for i, tt := range tests {
		eq := tt.ms1.IsEqual(tt.ms2)
		if eq != tt.wEqual {
			t.Errorf("#%d: equal get=%v, want=%v, sets: %v, %v", i, eq, tt.wEqual, tt.ms1, tt.ms2)
		}
	}
}

func TestMemberSetDiff(t *testing.T) {
	ma := &Member{Name: "a", Online: true, AppRunning: true}
	mb := &Member{Name: "b", Online: true, AppRunning: true}
	mboff := &Member{Name: "b"}
	tests := []struct {
		ms1, ms2 MemberSet
		wDiff    MemberSet
	}{{
		ms1:   NewMemberSet(mb, ma),
		ms2:   NewMemberSet(mb, ma),
		wDiff: MemberSet{},
	}, {
		ms1:   NewMemberSet(ma, mb),
		ms2:   NewMemberSet(ma),
		wDiff: NewMemberSet(mb),
	}, {
		ms1:   NewMemberSet(ma),
		ms2:   NewMemberSet(ma, mb),
		wDiff: MemberSet{},
	}, {
		ms1:   NewMemberSet(),
		ms2:   NewMemberSet(),
		wDiff: MemberSet{},
	}, {
		ms1:   NewMemberSet(),
		ms2:   NewMemberSet(ma),
		wDiff: MemberSet{},
	}, {
		ms1:   NewMemberSet(ma),
		ms2:   NewMemberSet(),
		wDiff: NewMemberSet(ma),
	}, {
		ms1:   NewMemberSet(ma, mb),
		ms2:   NewMemberSet(ma, mboff),
		wDiff: NewMemberSet(mb),
	}, {
		ms1:   NewMemberSet(ma, mboff),
		ms2:   NewMemberSet(ma, mb),
		wDiff: NewMemberSet(mboff),
	}}
	for i, tt := range tests {
		diff := tt.ms1.Diff(tt.ms2)
		if !diff.IsEqual(tt.wDiff) {
			t.Errorf("#%d: diff get=%v, want=%v, sets: %v, %v", i, diff, tt.wDiff, tt.ms1, tt.ms2)
		} else {
			t.Logf("#%d: diff get=%v, want=%v, sets: %v, %v", i, diff, tt.wDiff, tt.ms1, tt.ms2)
		}
	}
}
