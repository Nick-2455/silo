package notemodel

import (
	"strings"
	"testing"
)

func TestValidType(t *testing.T) {
	cases := []struct {
		input Type
		want  bool
	}{
		{TypeConcept, true},
		{TypeResource, true},
		{TypeRoadmap, true},
		{TypeCollection, true},
		{Type("quiz"), false},
		{Type(""), false},
		{Type("CONCEPT"), false},
	}
	for _, tc := range cases {
		got := ValidType(tc.input)
		if got != tc.want {
			t.Errorf("ValidType(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestValidKind(t *testing.T) {
	cases := []struct {
		t    Type
		kind string
		want bool
	}{
		// empty kind is always valid
		{TypeCollection, "", true},
		{TypeResource, "", true},
		{TypeConcept, "", true},
		// valid kinds
		{TypeCollection, "subject", true},
		{TypeCollection, "project", true},
		{TypeCollection, "other", true},
		{TypeResource, "book", true},
		{TypeResource, "article", true},
		{TypeResource, "other", true},
		// invalid kinds for types with restrictions
		{TypeCollection, "homework", false},
		{TypeResource, "podcast", false},
		// types with no kind restriction accept any kind
		{TypeConcept, "anything", true},
		{TypeRoadmap, "quarterly", true},
	}
	for _, tc := range cases {
		got := ValidKind(tc.t, tc.kind)
		if got != tc.want {
			t.Errorf("ValidKind(%q, %q) = %v, want %v", tc.t, tc.kind, got, tc.want)
		}
	}
}

func TestGet_HitAndMiss(t *testing.T) {
	tmpl, ok := Get(TypeConcept)
	if !ok {
		t.Fatal("Get(TypeConcept) returned false")
	}
	if tmpl.Type != TypeConcept {
		t.Errorf("expected TypeConcept, got %q", tmpl.Type)
	}
	if tmpl.Description == "" {
		t.Error("expected non-empty description")
	}

	_, ok = Get(Type("nonexistent"))
	if ok {
		t.Error("Get(nonexistent) should return false")
	}
}

func TestApplyDefaults_MergesKindDefault(t *testing.T) {
	// collection without kind → defaults to "other"
	fm := ApplyDefaults(TypeCollection, "", map[string]any{"title": "My Collection"})
	if fm["type"] != "collection" {
		t.Errorf("expected type=collection, got %v", fm["type"])
	}
	if fm["kind"] != "other" {
		t.Errorf("expected kind=other (default), got %v", fm["kind"])
	}
	if fm["title"] != "My Collection" {
		t.Errorf("expected title preserved, got %v", fm["title"])
	}
}

func TestApplyDefaults_ExplicitKindKept(t *testing.T) {
	fm := ApplyDefaults(TypeCollection, "subject", nil)
	if fm["kind"] != "subject" {
		t.Errorf("expected kind=subject, got %v", fm["kind"])
	}
}

func TestApplyDefaults_ConceptHasNoKind(t *testing.T) {
	fm := ApplyDefaults(TypeConcept, "", nil)
	if _, exists := fm["kind"]; exists {
		t.Errorf("concept with no kind should not have kind key, got %v", fm["kind"])
	}
}

func TestApplyDefaults_UnknownTypeSoftPass(t *testing.T) {
	fm := ApplyDefaults(Type("custom"), "", nil)
	if fm["type"] != "custom" {
		t.Errorf("unknown type should still be set, got %v", fm["type"])
	}
}

func TestApplyDefaults_DoesNotMutateInput(t *testing.T) {
	original := map[string]any{"title": "x"}
	ApplyDefaults(TypeResource, "book", original)
	if len(original) != 1 {
		t.Error("ApplyDefaults must not mutate the input map")
	}
}

func TestTemplates_ReturnsFour(t *testing.T) {
	tmpls := Templates()
	if len(tmpls) != 4 {
		t.Fatalf("expected 4 templates, got %d", len(tmpls))
	}
	types := map[Type]bool{}
	for _, tmpl := range tmpls {
		types[tmpl.Type] = true
	}
	for _, expected := range []Type{TypeConcept, TypeResource, TypeRoadmap, TypeCollection} {
		if !types[expected] {
			t.Errorf("missing template for type %q", expected)
		}
	}
}

func TestEmbeddedFS_LoadsAllFourTemplates(t *testing.T) {
	for _, expected := range []Type{TypeConcept, TypeResource, TypeRoadmap, TypeCollection} {
		tmpl, ok := Get(expected)
		if !ok {
			t.Fatalf("Get(%q) returned false", expected)
		}
		body := tmpl.Body()
		if body == "" {
			t.Errorf("template %q has empty body", expected)
		}
		if !strings.Contains(body, "---") {
			t.Errorf("template %q body should contain frontmatter delimiter", expected)
		}
	}
}

func TestValidate_KnownTypeValidKind(t *testing.T) {
	if err := Validate(TypeResource, "book"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_UnknownType(t *testing.T) {
	err := Validate(Type("quiz"), "")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	if !strings.Contains(err.Error(), "unknown note type") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_UnknownKind(t *testing.T) {
	err := Validate(TypeCollection, "homework")
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
	if !strings.Contains(err.Error(), "unknown kind") {
		t.Errorf("unexpected error message: %v", err)
	}
}
