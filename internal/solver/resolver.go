package solver

import (
	"fmt"
	"log"

	"github.com/kolkov/gportage/internal/pkg"
	"github.com/kolkov/gportage/internal/repo"
)

type PortageResolver struct {
	repo repo.Repository
}

func NewResolver(r repo.Repository) *PortageResolver {
	return &PortageResolver{repo: r}
}

func (r *PortageResolver) collectDependencies(pkg *pkg.Package, allPackages map[string]*pkg.Package) error {
	if _, exists := allPackages[pkg.Name]; exists {
		return nil // Уже обработан
	}

	// Сохраняем копию пакета
	copyPkg := *pkg
	allPackages[pkg.Name] = &copyPkg

	// Обрабатываем зависимости
	for _, dep := range pkg.Deps {
		// Загружаем зависимый пакет
		depPkg, err := r.repo.LoadPackage(dep.Name)
		if err != nil {
			log.Printf("Warning: dependency %s for %s not found: %v", dep.Name, pkg.Name, err)
			continue
		}

		// Рекурсивно обрабатываем его зависимости
		if err := r.collectDependencies(depPkg, allPackages); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	return nil
}

func (r *PortageResolver) Resolve(packages []string) (map[string]*pkg.Package, error) {
	adapter := NewGophersatAdapter()
	allPackages := make(map[string]*pkg.Package)

	// Загрузка и сбор всех зависимостей
	for _, pkgName := range packages {
		p, err := r.repo.LoadPackage(pkgName)
		if err != nil {
			return nil, fmt.Errorf("failed to load package %s: %w", pkgName, err)
		}

		log.Printf("Resolving package: %s-%s with %d dependencies",
			p.Name, p.Version, len(p.Deps))

		if err := r.collectDependencies(p, allPackages); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	log.Printf("Total packages in dependency graph: %d", len(allPackages))

	// Сначала добавляем ВСЕ пакеты в решатель
	for _, p := range allPackages {
		adapter.AddPackage(p)
	}

	// Затем добавляем ограничения
	for _, p := range allPackages {
		// Добавляем ограничение, что основной пакет должен быть установлен
		if contains(packages, p.Name) {
			versionStr := "any"
			if p.Version != "" {
				versionStr = p.Version
			}
			log.Printf("Adding constraint for required package: %s = %s", p.Name, versionStr)

			adapter.AddConstraint(pkg.Constraint{
				Type:    pkg.ConstraintTypeVersion,
				Name:    p.Name,
				Version: pkg.NewVersionConstraint(pkg.OpEqual, p.Version),
			})
		}

		// Добавляем зависимости пакета
		for _, dep := range p.Deps {
			// Проверяем существует ли пакет
			if _, ok := allPackages[dep.Name]; !ok {
				log.Printf("Skipping unresolved dependency: %s", dep.Name)
				continue
			}

			versionStr := "any"
			if dep.Version != nil {
				versionStr = dep.Version.String()
			}
			log.Printf("Adding dependency constraint: %s %s", dep.Name, versionStr)

			if err := adapter.AddConstraint(dep); err != nil {
				log.Printf("Warning: failed to add constraint: %v", err)
			}
		}
	}

	// УБРАТЬ: дублирующий вызов AddExactlyOneConstraint
	/*
	   processedPackages := make(map[string]bool)
	   for _, p := range allPackages {
	       if !processedPackages[p.Name] {
	           versions := adapter.GetPackageVersions(p.Name)
	           if len(versions) > 0 {
	               adapter.AddExactlyOneConstraint(p.Name, versions)
	               processedPackages[p.Name] = true
	           }
	       }
	   }
	*/

	log.Printf("Total clauses in SAT problem: %d", len(adapter.clauses))

	// Решение
	status, solution, err := adapter.Solve()
	if err != nil {
		return nil, err
	}

	if status != pkg.StatusSat {
		log.Printf("UNSAT core analysis:")
		for i, clause := range adapter.clauses {
			log.Printf("Clause %d: %v", i, clause)
		}
		return nil, fmt.Errorf("no solution found")
	}

	// Построение результата
	result := make(map[string]*pkg.Package)
	for name, _ := range solution {
		p, err := r.repo.LoadPackage(name)
		if err != nil {
			log.Printf("Warning: package %s not found: %v", name, err)
			continue
		}
		result[name] = p
	}

	// Вывод красивого списка пакетов
	log.Println("\nResolved packages:")
	for name, p := range result {
		log.Printf("- %s-%s [slot:%s]", name, p.Version, p.Slot.Name)
	}

	return result, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
