package pkg

import (
	"strings"
)

// Slot представляет слот пакета
type Slot struct {
	Name    string
	Subslot string
}

func (s Slot) String() string {
	if s.Subslot != "" {
		return s.Name + "/" + s.Subslot
	}
	return s.Name
}

// ParseSlot парсит строковое представление слота
func ParseSlot(slot string) Slot {
	parts := strings.Split(slot, "/")
	if len(parts) > 1 {
		return Slot{
			Name:    parts[0],
			Subslot: parts[1],
		}
	}
	return Slot{
		Name:    slot,
		Subslot: "",
	}
}

type Package struct {
	Name     string
	Version  string
	Slot     Slot
	UseFlags map[string]bool
	Deps     []Constraint
	Provides []Constraint // Виртуальные пакеты
}

// NewPackage создает новый экземпляр пакета
func NewPackage(name, version, slotStr string) *Package {
	return &Package{
		Name:     name,
		Version:  version,
		Slot:     ParseSlot(slotStr),
		UseFlags: make(map[string]bool),
		Deps:     make([]Constraint, 0),
		Provides: make([]Constraint, 0),
	}
}

// AddDependency добавляет зависимость к пакету
func (p *Package) AddDependency(constraint Constraint) {
	p.Deps = append(p.Deps, constraint)
}

// ConflictsWith проверяет конфликт слотов
func (p *Package) ConflictsWith(other *Package) bool {
	// Пакеты с разными именами могут конфликтовать из-за слотов
	if p.Name == other.Name {
		return false // Разные версии одного пакета обрабатываются отдельно
	}

	// Конфликт, если слоты совпадают, но под-слоты разные
	return p.Slot.Name == other.Slot.Name && p.Slot.Subslot != other.Slot.Subslot
}
