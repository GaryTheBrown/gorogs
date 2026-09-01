package systeminterface

import (
	"fmt"
	"slices"
	"strings"
)

var (
	RegisteredSystems = make(map[string]System)
	SystemList        []System
)

func Register(sys System) {
	if sys == nil {
		return
	}
	RegisteredSystems[strings.ToLower(sys.Name())] = sys
}

func InitializeAndSort() error {
	var list []System
	for _, sys := range RegisteredSystems {
		list = append(list, sys)
	}

	slices.SortFunc(list, func(a, b System) int {
		aOrder := getTypeTier(a.Type())
		bOrder := getTypeTier(b.Type())
		if aOrder != bOrder {
			return aOrder - bOrder
		}
		return b.Priority() - a.Priority()
	})

	resolved := make([]System, 0, len(list))
	visited := make(map[string]bool)
	temp := make(map[string]bool)

	var visit func(sys System) error
	visit = func(sys System) error {
		sysKey := strings.ToLower(sys.Name())
		if temp[sysKey] {
			return fmt.Errorf("circular dependency detected: %s", sysKey)
		}
		if !visited[sysKey] {
			temp[sysKey] = true

			for _, depName := range sys.Dependencies() {
				depKey := strings.ToLower(depName)
				if dep, exists := RegisteredSystems[depKey]; exists {
					if err := visit(dep); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("missing hard dependency: %s requires %s", sys.Name(), depName)
				}
			}

			for _, softName := range sys.OrderAfter() {
				softKey := strings.ToLower(softName)
				if dep, exists := RegisteredSystems[softKey]; exists {
					if err := visit(dep); err != nil {
						return err
					}
				}
			}

			temp[sysKey] = false
			visited[sysKey] = true
			resolved = append(resolved, sys)
		}
		return nil
	}

	for _, sys := range list {
		if !visited[strings.ToLower(sys.Name())] {
			if err := visit(sys); err != nil {
				return err
			}
		}
	}

	SystemList = resolved
	return nil
}

func getTypeTier(t SystemTypeEnum) int {
	switch t {
	case Utility:
		return 1
	case Share:
		return 2
	case Beacon:
		return 3
	default:
		return 4
	}
}

func GetOrderedSystems() []System {
	return SystemList
}
