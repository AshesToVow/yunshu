package menu

import "testing"

func TestDefaultPathBindingsCoversCatalogLeaves(t *testing.T) {
	for _, spec := range DefaultCatalog() {
		var walk func([]Spec)
		walk = func(items []Spec) {
			for _, it := range items {
				if it.Component != "" {
					path := normalizeMenuPath(it.Path)
					if _, ok := DefaultPathBindings()[path]; !ok {
						t.Errorf("missing default binding for catalog path %q", it.Path)
					}
				}
				if len(it.Children) > 0 {
					walk(it.Children)
				}
			}
		}
		walk([]Spec{spec})
	}
}

func TestUserCanAccessMenuEmptyBindings(t *testing.T) {
	if !UserCanAccessMenu(nil, 1, nil) {
		t.Fatal("empty bindings should allow access")
	}
}
