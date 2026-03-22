package theme

import (
	"reflect"
	"testing"
)

func TestByNameValid(t *testing.T) {
	names := []string{"default", "minimal", "neon", "monochrome"}
	for _, name := range names {
		th, err := ByName(name)
		if err != nil {
			t.Errorf("ByName(%q) returned error: %v", name, err)
		}
		if th.Name != name {
			t.Errorf("ByName(%q).Name = %q", name, th.Name)
		}
	}
}

func TestByNameInvalid(t *testing.T) {
	_, err := ByName("nonexistent")
	if err == nil {
		t.Error("ByName(nonexistent) should return error")
	}
}

func TestByNameCaseInsensitive(t *testing.T) {
	th, err := ByName("NEON")
	if err != nil {
		t.Errorf("ByName(NEON) returned error: %v", err)
	}
	if th.Name != "neon" {
		t.Errorf("ByName(NEON).Name = %q, want neon", th.Name)
	}
}

func TestAllThemesHaveAllColors(t *testing.T) {
	themes := []Theme{Default(), Minimal(), Neon(), Monochrome()}
	for _, th := range themes {
		v := reflect.ValueOf(th)
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if field.Kind() == reflect.String && typ.Field(i).Name != "Name" {
				if field.String() == "" {
					t.Errorf("Theme %q: field %s is empty", th.Name, typ.Field(i).Name)
				}
			}
		}
	}
}

func TestDefaultThemeMatchesOriginalColors(t *testing.T) {
	d := Default()
	if d.ArcGreen != "#22C55E" {
		t.Errorf("Default ArcGreen = %q, want #22C55E", d.ArcGreen)
	}
	if d.ArcRed != "#EF4444" {
		t.Errorf("Default ArcRed = %q, want #EF4444", d.ArcRed)
	}
	if d.Title != "#FF6B35" {
		t.Errorf("Default Title = %q, want #FF6B35", d.Title)
	}
}
