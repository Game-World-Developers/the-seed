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

//go:embed state_machine.hpp.tmpl
var stateMachineTemplate string

//go:embed system.hpp.tmpl
var systemHppTemplate string

//go:embed event.hpp.tmpl
var eventTemplate string

//go:embed asset.hpp.tmpl
var assetTemplate string

//go:embed block.hpp.tmpl
var blockTemplate string

//go:embed system.cpp.tmpl
var systemCppTemplate string

//go:embed system_part.cpp.tmpl
var systemPartCppTemplate string

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
	case "state_machine":
		return stateMachineTemplate, nil
	case "system":
		return systemHppTemplate, nil
	case "event":
		return eventTemplate, nil
	case "asset":
		return assetTemplate, nil
	case "block":
		return blockTemplate, nil
	default:
		return "", fmt.Errorf("unknown template kind: %s", kind)
	}
}

func LoadCpp(kind string) (string, error) {
	switch kind {
	case "system":
		return systemCppTemplate, nil
	default:
		return "", fmt.Errorf("no .cpp template for kind: %s", kind)
	}
}

func LoadPartCpp(kind string) (string, error) {
	switch kind {
	case "system":
		return systemPartCppTemplate, nil
	default:
		return "", fmt.Errorf("no part .cpp template for kind: %s", kind)
	}
}
