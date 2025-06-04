package solver

import (
    "github.com/khulnasoft-lab/go-sat"
)

type PackageConstraint struct {
    Name      string
    Versions  []string
    Slot      string
    UseFlags  map[string]bool
}

func ResolveDependencies(rootPkg string, constraints []PackageConstraint) (map[string]string, error) {
    solver := sat.NewSolver()
    
    // Преобразование зависимостей в булевы ограничения
    for _, c := range constraints {
        vars := make([]sat.Var, 0)
        for _, ver := range c.Versions {
            // Создание переменной для каждой версии пакета
            v := solver.NewVar()
            vars = append(vars, v)
        }
        // Добавление ограничения "ровно одна версия"
        solver.AddExactlyOne(vars...)
    }
    
    // Решение системы ограничений
    if !solver.Solve() {
        return nil, errors.New("conflict detected")
    }
    
    // Построение результата
    solution := make(map[string]string)
    for _, c := range constraints {
        for i, ver := range c.Versions {
            if solver.Value(c.Var[i]) {
                solution[c.Name] = ver
                break
            }
        }
    }
    return solution, nil
}