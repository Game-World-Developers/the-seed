package templates

import (
	_ "embed"
	"fmt"
)

//go:embed component.hpp.tmpl
var componentTemplate string

//go:embed trait.hpp.tmpl
var traitTemplate string

//go:embed entity.hpp.tmpl
var entityTemplate string

//go:embed archetype.hpp.tmpl
var archetypeTemplate string

func Load(kind string) (string, error) {
	switch kind {
	case "component":
		return componentTemplate, nil
	case "trait":
		return traitTemplate, nil
	case "entity":
		return entityTemplate, nil
	case "archetype":
		return archetypeTemplate, nil
	default:
		return "", fmt.Errorf("unknown template kind: %s", kind)
	}
}
