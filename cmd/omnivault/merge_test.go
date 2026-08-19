package main

import (
	"testing"

	"github.com/54wu/omnivault/internal/merge"
)

func TestResolveMergeField(t *testing.T) {
	cases := []struct {
		label string
		want  string
	}{
		{"姓名", "identity.name"},
		{"手机号", "identity.phone"},
		{"联系电话", "identity.phone"},
		{"身份证号", "identity.id_number"},
		{"硕士院校", "education.postgrad_school"},
		{"母亲姓名", "family.mother_name"},
		{"不存在的字段", ""},
	}
	for _, c := range cases {
		f := merge.ResolveField(c.label)
		if c.want == "" {
			if f != nil {
				t.Fatalf("label %q: expected nil, got %s", c.label, f.FieldID)
			}
			continue
		}
		if f == nil || f.FieldID != c.want {
			t.Fatalf("label %q: expected %s, got %v", c.label, c.want, f)
		}
	}
}

func TestVerifyOwnership(t *testing.T) {
	persons := []merge.Person{
		{ID: "p1", Name: "徐小明", IDNumber: "440101199501011234", Phone: "13900001111", Email: "xuxiaoming@example.com"},
		{ID: "p2", Name: "张三", IDNumber: "110101199201011234", Phone: "13800000000", Email: "zhangsan@example.com"},
	}

	match := merge.VerifyOwnership(merge.Person{Name: "徐小明", IDNumber: "440101199501011234", Phone: "13900001111"}, persons)
	if !match.Matched || match.PersonID != "p1" {
		t.Fatalf("expected match p1, got matched=%v id=%s score=%.2f", match.Matched, match.PersonID, match.Score)
	}

	diff := merge.VerifyOwnership(merge.Person{Name: "李四", IDNumber: "111111111111111111", Phone: "13900000000"}, persons)
	if diff.Matched {
		t.Fatalf("expected no match, got id=%s score=%.2f", diff.PersonID, diff.Score)
	}

	partial := merge.VerifyOwnership(merge.Person{Name: "徐小明", Phone: "13900001111"}, persons)
	if !partial.Matched || partial.PersonID != "p1" {
		t.Fatalf("expected name+phone to match p1, got matched=%v id=%s", partial.Matched, partial.PersonID)
	}
}