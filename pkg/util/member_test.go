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

package util

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
			t.Errorf("#%d: FAIL equal get=%v, want=%v, sets: %v, %v", i, eq, tt.wEqual, tt.ms1, tt.ms2)
		}
	}
}

func TestMemberSetDiffExtended(t *testing.T) {
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
		diff := tt.ms1.DiffExtended(tt.ms2)
		if !diff.IsEqual(tt.wDiff) {
			t.Errorf("#%d: FAIL diff get=%v, want=%v, sets: %v, %v", i, diff, tt.wDiff, tt.ms1, tt.ms2)
		} else {
			t.Logf("#%d: PASS diff get=%v, want=%v, sets: %v, %v", i, diff, tt.wDiff, tt.ms1, tt.ms2)
		}
	}
}

func TestMemberReconcile(t *testing.T) {

	ra := &Member{Name: "a", Online: true}
	rb := &Member{Name: "b", Online: true}
	rc := &Member{Name: "c", Online: true}

	ma := &Member{Name: "a", Online: true, AppRunning: true}
	mb := &Member{Name: "b", Online: true, AppRunning: true}
	mc := &Member{Name: "c", Online: true}

	offa := &Member{Name: "a"}
	offb := &Member{Name: "b"}
	offc := &Member{Name: "c"}

	tests := []struct {
		ms1, ms2 MemberSet
		wRes     MemberSet
		max      int32
	}{{
		ms1:  NewMemberSet(ma, mb, mc),
		ms2:  NewMemberSet(ra, rc),
		wRes: NewMemberSet(ma, offb, mc),
		max:  3,
	}, {
		ms1:  NewMemberSet(ma, mc),
		ms2:  NewMemberSet(ra, rb),
		wRes: NewMemberSet(ma, rb, offc),
		max:  3,
	}, {
		ms1:  NewMemberSet(ra, mc),
		ms2:  NewMemberSet(ma, rb),
		wRes: NewMemberSet(ra, rb, offc),
		max:  3,
	}, {
		ms1:  NewMemberSet(offa, offb, mc),
		ms2:  NewMemberSet(ra, mb),
		wRes: NewMemberSet(ra, rb, offc),
		max:  3,
	}, {
		ms1:  NewMemberSet(offa, mb, mc),
		ms2:  NewMemberSet(ra, rb, rc),
		wRes: NewMemberSet(ra, mb, mc),
		max:  3,
	}}
	for i, tt := range tests {
		t.Logf("=== Test #%d ====", i)
		rec, _ := tt.ms1.Reconcile(tt.ms2, tt.max)
		if !rec.IsEqual(tt.wRes) {
			t.Errorf("#%d: FAIL get=%v, want=%v, sets: %v, %v", i, rec, tt.wRes, tt.ms1, tt.ms2)
		} else {
			t.Logf("#%d: PASS get=%v, want=%v, sets: %v, %v", i, rec, tt.wRes, tt.ms1, tt.ms2)
		}
	}
}
