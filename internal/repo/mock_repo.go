package repo

import (
	"fmt"

	"github.com/kolkov/gportage/internal/pkg"
)

type MockRepository struct {
	packages map[string]*pkg.Package
}

func NewMockRepository() *MockRepository {
	packages := make(map[string]*pkg.Package)

	// Создаем пакет hello
	hello := pkg.NewPackage("app-misc/hello", "2.10", "0")
	hello.AddDependency(pkg.Constraint{
		Type:    pkg.ConstraintTypeVersion,
		Name:    "sys-libs/zlib",
		Version: pkg.NewVersionConstraint(pkg.OpGreaterEqual, "1.2.13"),
	})

	// Создаем пакет zlib
	zlib := pkg.NewPackage("sys-libs/zlib", "1.2.13", "0")

	packages["app-misc/hello"] = hello
	packages["sys-libs/zlib"] = zlib

	// Добавляем конфликтующий пакет
	conflict := pkg.NewPackage("conflict/example", "1.0", "0")
	conflict.AddDependency(pkg.Constraint{
		Type:    pkg.ConstraintTypeVersion,
		Name:    "sys-libs/zlib",
		Version: pkg.NewVersionConstraint(pkg.OpLess, "1.2.0"), // Конфликтующая версия
	})
	packages["conflict/example"] = conflict

	return &MockRepository{
		packages: packages,
	}
}

func (m *MockRepository) LoadPackages(names []string) ([]*pkg.Package, error) {
	result := make([]*pkg.Package, 0, len(names))
	for _, name := range names {
		if pkg, exists := m.packages[name]; exists {
			// Создаем копию пакета
			copyPkg := *pkg
			result = append(result, &copyPkg)
		} else {
			return nil, fmt.Errorf("package %s not found", name)
		}
	}
	return result, nil
}

func (m *MockRepository) LoadPackage(name string) (*pkg.Package, error) {
	if pkg, exists := m.packages[name]; exists {
		// Создаем копию пакета
		copyPkg := *pkg
		return &copyPkg, nil
	}
	return nil, fmt.Errorf("package %s not found", name)
}

func (m *MockRepository) AddPackage(p *pkg.Package) error {
	// Создаем копию перед сохранением
	copyPkg := *p
	m.packages[p.Name] = &copyPkg
	return nil
}
