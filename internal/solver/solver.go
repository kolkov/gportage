package solver

import "github.com/kolkov/gportage/internal/pkg"

// Status представляет статус решения
type Status int

const (
	Indet Status = iota // Не определен
	Sat                 // Решение найдено
	Unsat               // Конфликт зависимостей
)

// Solver интерфейс для решателя зависимостей
type Solver interface {
	AddPackage(pkg *pkg.Package)
	AddConstraint(constraint pkg.Constraint) error
	Solve() (Status, map[string]string, error)
}
