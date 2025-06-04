package pkg

import (
	"regexp"
	"strconv"
	"strings"
)

// ConstraintType определяет тип ограничения
type ConstraintType int

const (
	ConstraintTypeVersion ConstraintType = iota
	ConstraintTypeSlot
	ConstraintTypeUseFlag
)

// Status представляет статус решения
type Status int

const (
	StatusIndet Status = iota // Не определен
	StatusSat                 // Решение найдено
	StatusUnsat               // Конфликт зависимостей
)

// VersionOperator определяет оператор сравнения версий
type VersionOperator int

const (
	OpEqual VersionOperator = iota
	OpGreater
	OpGreaterEqual
	OpLess
	OpLessEqual
)

// VersionConstraint представляет ограничение версии
type VersionConstraint struct {
	Operator VersionOperator
	Version  string
}

// Constraint представляет общее ограничение для пакета
type Constraint struct {
	Type      ConstraintType
	Name      string
	Version   *VersionConstraint // Для ограничений версий
	Slot      string             // Для ограничений слота
	Flag      string             // Для USE-флагов
	Required  bool               // Обязательное требование
	Condition string             // Условие USE-флага
}

func (c Constraint) String() string {
	if c.Version == nil {
		return c.Name
	}
	return c.Name + " " + c.Version.String()
}

// NewVersionConstraint создает новое ограничение версии
func NewVersionConstraint(operator VersionOperator, version string) *VersionConstraint {
	return &VersionConstraint{
		Operator: operator,
		Version:  version,
	}
}

// NewExactVersionConstraint создает ограничение на точную версию
func NewExactVersionConstraint(version string) *VersionConstraint {
	return NewVersionConstraint(OpEqual, version)
}

// NewMinVersionConstraint создает ограничение на минимальную версию
func NewMinVersionConstraint(version string) *VersionConstraint {
	return NewVersionConstraint(OpGreaterEqual, version)
}

// NewMaxVersionConstraint создает ограничение на максимальную версию
func NewMaxVersionConstraint(version string) *VersionConstraint {
	return NewVersionConstraint(OpLessEqual, version)
}

// String возвращает строковое представление ограничения версии
func (vc *VersionConstraint) String() string {
	if vc == nil {
		return "any"
	}

	switch vc.Operator {
	case OpEqual:
		return vc.Version
	case OpGreater:
		return ">" + vc.Version
	case OpGreaterEqual:
		return ">=" + vc.Version
	case OpLess:
		return "<" + vc.Version
	case OpLessEqual:
		return "<=" + vc.Version
	default:
		return "unknown"
	}
}

// Satisfies проверяет, удовлетворяет ли версия ограничению
func (vc *VersionConstraint) Satisfies(version string) bool {
	if vc == nil {
		return true
	}

	switch vc.Operator {
	case OpEqual:
		return version == vc.Version
	case OpGreater:
		return CompareVersions(version, vc.Version) > 0
	case OpGreaterEqual:
		return CompareVersions(version, vc.Version) >= 0
	case OpLess:
		return CompareVersions(version, vc.Version) < 0
	case OpLessEqual:
		return CompareVersions(version, vc.Version) <= 0
	default:
		return true
	}
}

// CompareVersions сравнивает версии в формате Gentoo
func CompareVersions(v1, v2 string) int {
	// Разбиваем версии на компоненты: 1.2.3_alpha4-r5 -> [1, 2, 3, "alpha", 4, 5]
	splitVersion := func(v string) []interface{} {
		// Разделяем цифры и буквы
		re := regexp.MustCompile(`(\d+)|([a-zA-Z]+)`)
		parts := re.FindAllString(v, -1)

		var result []interface{}
		for _, part := range parts {
			if num, err := strconv.Atoi(part); err == nil {
				result = append(result, num)
			} else {
				result = append(result, part)
			}
		}
		return result
	}

	parts1 := splitVersion(v1)
	parts2 := splitVersion(v2)

	// Сравниваем компоненты
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		switch a := parts1[i].(type) {
		case int:
			if b, ok := parts2[i].(int); ok {
				if a != b {
					return a - b
				}
			} else {
				// Число всегда больше строки
				return 1
			}
		case string:
			if b, ok := parts2[i].(string); ok {
				if cmp := strings.Compare(a, b); cmp != 0 {
					return cmp
				}
			} else {
				// Строка всегда меньше числа
				return -1
			}
		}
	}

	// Если все компоненты равны, более длинная версия считается большей
	return len(parts1) - len(parts2)
}

// ParseVersionConstraint парсит строковое представление ограничения
func ParseVersionConstraint(s string) (*VersionConstraint, error) {
	if s == "" {
		return nil, nil
	}

	operators := map[string]VersionOperator{
		"=":  OpEqual,
		">":  OpGreater,
		">=": OpGreaterEqual,
		"<":  OpLess,
		"<=": OpLessEqual,
	}

	for opStr, op := range operators {
		if strings.HasPrefix(s, opStr) {
			version := strings.TrimSpace(strings.TrimPrefix(s, opStr))
			return &VersionConstraint{
				Operator: op,
				Version:  version,
			}, nil
		}
	}

	// По умолчанию считается точной версией
	return &VersionConstraint{
		Operator: OpEqual,
		Version:  s,
	}, nil
}

// NewSimpleConstraint создает ограничение без указания версии
func NewSimpleConstraint(name string) Constraint {
	return Constraint{
		Type: ConstraintTypeVersion,
		Name: name,
	}
}
