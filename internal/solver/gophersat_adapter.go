package solver

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/crillab/gophersat/solver"
	"github.com/kolkov/gportage/internal/pkg"
)

type GophersatAdapter struct {
	clauses      [][]int
	vars         map[string]int            // name@version -> var ID
	varNames     map[int]string            // var ID -> name@version
	packages     map[string][]*pkg.Package // name -> []versions
	addedClauses map[string]struct{}       // для предотвращения дублирования
}

func NewGophersatAdapter() *GophersatAdapter {
	return &GophersatAdapter{
		vars:         make(map[string]int),
		varNames:     make(map[int]string),
		packages:     make(map[string][]*pkg.Package),
		addedClauses: make(map[string]struct{}),
	}
}

func (g *GophersatAdapter) getVarID(key string) int {
	if id, exists := g.vars[key]; exists {
		return id
	}

	// Создаем новую переменную
	id := len(g.vars) + 1
	g.vars[key] = id
	g.varNames[id] = key
	return id
}

func (g *GophersatAdapter) addClause(clause []int) {
	// Создаем уникальный ключ для клаузы (упорядоченный)
	sortedClause := make([]int, len(clause))
	copy(sortedClause, clause)
	sort.Ints(sortedClause)
	key := fmt.Sprintf("%v", sortedClause)

	// Если клауза уже добавлена, пропускаем
	if _, exists := g.addedClauses[key]; exists {
		return
	}

	g.clauses = append(g.clauses, clause)
	g.addedClauses[key] = struct{}{}
	log.Printf("Added clause: %v", clause)
}

func (g *GophersatAdapter) AddPackage(p *pkg.Package) {
	key := p.Name + "@" + p.Version

	// Регистрируем пакет
	if _, exists := g.packages[p.Name]; !exists {
		g.packages[p.Name] = []*pkg.Package{}
	}

	// Убедимся, что эта версия еще не добавлена
	for _, existing := range g.packages[p.Name] {
		if existing.Version == p.Version {
			return
		}
	}

	g.packages[p.Name] = append(g.packages[p.Name], p)

	// Регистрируем переменную
	g.getVarID(key)

	// Логирование
	log.Printf("Added package: %s-%s", p.Name, p.Version)
}

func (g *GophersatAdapter) AddConstraint(c pkg.Constraint) error {
	switch c.Type {
	case pkg.ConstraintTypeVersion:
		return g.addVersionConstraint(c)
	case pkg.ConstraintTypeSlot:
		return g.addSlotConstraint(c)
	case pkg.ConstraintTypeUseFlag:
		return g.addUseFlagConstraint(c)
	default:
		return fmt.Errorf("unsupported constraint type: %d", c.Type)
	}
}

func (g *GophersatAdapter) addVersionConstraint(c pkg.Constraint) error {
	log.Printf("Processing constraint: %s %s", c.Name, c.Version)

	// Для простых ограничений без версии
	if c.Version == nil {
		return g.addSimpleConstraint(c.Name)
	}

	// Собираем все пакеты, удовлетворяющие ограничению
	var satisfiedVars []int
	for _, p := range g.packages[c.Name] {
		if c.Version.Satisfies(p.Version) {
			key := p.Name + "@" + p.Version
			varID := g.getVarID(key)
			satisfiedVars = append(satisfiedVars, varID)
			log.Printf("Package %s satisfies constraint %s %s", key, c.Name, c.Version.String())
		} else {
			log.Printf("Package %s does NOT satisfy constraint %s %s",
				p.Name+"@"+p.Version, c.Name, c.Version.String())
		}
	}

	if len(satisfiedVars) == 0 {
		log.Printf("Warning: no package satisfies %s %s", c.Name, c.Version.String())
		return nil
	}

	// Добавляем клаузу: хотя бы один из подходящих пакетов должен быть выбран
	g.addClause(satisfiedVars)
	return nil
}

func (g *GophersatAdapter) addSimpleConstraint(name string) error {
	// Исправлено: проверка существования пакета
	if versions, exists := g.packages[name]; exists && len(versions) > 0 {
		var packageVars []int
		for _, p := range versions {
			key := p.Name + "@" + p.Version
			varID := g.getVarID(key)
			packageVars = append(packageVars, varID)
		}
		g.addClause(packageVars)
		return nil
	}
	return fmt.Errorf("package %s not found in repository", name)
}

func (g *GophersatAdapter) AddExactlyOneConstraint(pkgName string, versions []string) {
	var versionVars []int
	for _, version := range versions {
		key := pkgName + "@" + version
		if varID, exists := g.vars[key]; exists {
			versionVars = append(versionVars, varID)
		}
	}

	if len(versionVars) == 0 {
		return
	}

	// Для одного пакета - просто обязательная установка
	if len(versionVars) == 1 {
		g.addClause([]int{versionVars[0]})
		log.Printf("Added mandatory constraint for %s: [%d]", pkgName, versionVars[0])
		return
	}

	// Добавляем клаузы для ограничения "ровно одна версия"
	for _, clause := range exactlyOne(versionVars) {
		g.addClause(clause)
	}
	log.Printf("Added exactly-one constraint for %s: %d versions", pkgName, len(versions))
}

func (g *GophersatAdapter) addSlotConstraint(c pkg.Constraint) error {
	// Находим все пакеты с указанным слотом
	var slotVars []int
	for _, pkgList := range g.packages {
		for _, p := range pkgList {
			if p.Slot.Name == c.Slot {
				key := p.Name + "@" + p.Version
				varID := g.vars[key]
				slotVars = append(slotVars, varID)
			}
		}
	}

	if len(slotVars) == 0 {
		return fmt.Errorf("no package provides slot %s", c.Slot)
	}

	// Добавляем клаузу: хотя бы один пакет в слоте должен быть установлен
	g.addClause(slotVars)
	return nil
}

func (g *GophersatAdapter) addUseFlagConstraint(c pkg.Constraint) error {
	if c.Required {
		// Создаем переменную для USE-флага
		flagVar := g.getVarID("USE_" + c.Flag)
		g.addClause([]int{flagVar})
	}
	return nil
}

// exactlyOne генерирует клаузы, гарантирующие, что ровно одна переменная из списка истинна
func exactlyOne(vars []int) [][]int {
	// Добавляем клаузу: хотя бы одна истинна
	clause := make([]int, len(vars))
	copy(clause, vars)
	clauses := [][]int{clause}

	// Добавляем попарные отрицания: не более одной истинной
	for i := 0; i < len(vars); i++ {
		for j := i + 1; j < len(vars); j++ {
			clauses = append(clauses, []int{-vars[i], -vars[j]})
		}
	}
	return clauses
}

func (g *GophersatAdapter) Solve() (pkg.Status, map[string]string, error) {
	// Логирование перед решением
	log.Printf("Solving SAT problem with %d variables and %d clauses", len(g.vars), len(g.clauses))

	// УБРАТЬ: дублирующее добавление ограничений "ровно одна версия"
	// (это уже делается в AddExactlyOneConstraint)

	// УБРАТЬ: конфликты слотов (временно, пока не реализовано)
	/*
	   for _, versions1 := range g.packages {
	       for _, p1 := range versions1 {
	           for _, versions2 := range g.packages {
	               for _, p2 := range versions2 {
	                   if p1.ConflictsWith(p2) {
	                       key1 := p1.Name + "@" + p1.Version
	                       key2 := p2.Name + "@" + p2.Version
	                       varID1 := g.vars[key1]
	                       varID2 := g.vars[key2]
	                       g.addClause([]int{-varID1, -varID2})
	                   }
	               }
	           }
	       }
	   }
	*/

	// Создаем проблему
	pb := solver.ParseSlice(g.clauses)

	// Создаем решатель
	s := solver.New(pb)
	s.Verbose = false

	// Решаем проблему
	status := s.Solve()

	if status == solver.Sat {
		log.Printf("SAT solution found")
		solution := make(map[string]string)
		model := s.Model()

		// Проходим по всем зарегистрированным переменным
		for key, varID := range g.vars {
			if varID <= len(model) && model[varID-1] {
				parts := strings.Split(key, "@")
				if len(parts) == 2 {
					solution[parts[0]] = parts[1]
				}
			}
		}
		return pkg.StatusSat, solution, nil
	}

	if status == solver.Unsat {
		log.Printf("UNSAT: no solution possible")
		return pkg.StatusUnsat, nil, nil
	}

	log.Printf("INDETERMINATE: solver timeout")
	return pkg.StatusIndet, nil, fmt.Errorf("solver timeout")
}

// Новый метод для получения версий пакета
func (g *GophersatAdapter) GetPackageVersions(name string) []string {
	var versions []string
	for _, p := range g.packages[name] {
		versions = append(versions, p.Version)
	}
	return versions
}
