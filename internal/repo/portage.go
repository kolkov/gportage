package repo

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kolkov/gportage/internal/pkg"
)

type PortageRepository struct {
	Path string
}

func NewPortageRepository(path string) (*PortageRepository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository directory does not exist: %s", absPath)
	}

	return &PortageRepository{
		Path: absPath,
	}, nil
}

func (pr *PortageRepository) LoadPackages(names []string) ([]*pkg.Package, error) {
	var packages []*pkg.Package

	for _, name := range names {
		pkg, err := pr.LoadPackage(name)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

func (pr *PortageRepository) LoadPackage(name string) (*pkg.Package, error) {
	category, pkgName, found := strings.Cut(name, "/")
	if !found {
		return nil, fmt.Errorf("invalid package name: %s", name)
	}

	pkgDir := filepath.Join(pr.Path, category, pkgName)

	// Добавляем логирование пути
	absPath, _ := filepath.Abs(pkgDir)
	log.Printf("Looking for package in: %s", absPath)

	files, err := ioutil.ReadDir(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("error reading package directory: %w", err)
	}

	var versions []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".ebuild") {
			continue
		}

		if strings.HasSuffix(file.Name(), ".ebuild") {
			version := strings.TrimSuffix(file.Name(), ".ebuild")
			version = strings.TrimPrefix(version, pkgName+"-")
			versions = append(versions, version)
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no ebuilds found for %s", name)
	}

	// Берем последнюю версию (для примера)
	latestVersion := versions[len(versions)-1]
	return pr.parseEbuild(name, filepath.Join(pkgDir, pkgName+"-"+latestVersion+".ebuild")) // Передаем имя пакета
}

func (pr *PortageRepository) parseEbuild(name, path string) (*pkg.Package, error) {
	log.Printf("Parsing ebuild: %s", path)
	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error reading ebuild file: %v", err)
		return nil, err
	}

	// Упрощенный парсер ebuild
	p := &pkg.Package{
		Name:     name, // Исправлено! Устанавливаем имя пакета
		Version:  "",
		Slot:     pkg.Slot{Name: "0"},
		UseFlags: make(map[string]bool),
		Deps:     make([]pkg.Constraint, 0),
		Provides: make([]pkg.Constraint, 0),
	}

	// Регулярные выражения для парсинга
	versionRe := regexp.MustCompile(`(?m)^VERSION="([^"]+)"`)
	slotRe := regexp.MustCompile(`(?m)^SLOT="([^"]+)"`)
	dependRe := regexp.MustCompile(`(?m)^RDEPEND="([^"]+)"`)
	iuseRe := regexp.MustCompile(`(?m)^IUSE="([^"]+)"`)
	provideRe := regexp.MustCompile(`(?m)^PROVIDE="([^"]+)"`)

	// Извлекаем версию из имени файла
	filename := filepath.Base(path)
	if matches := strings.Split(filename, "-"); len(matches) > 1 {
		p.Version = strings.TrimSuffix(matches[len(matches)-1], ".ebuild")
	}

	// Парсим метаданные
	if matches := versionRe.FindStringSubmatch(string(content)); len(matches) > 1 {
		p.Version = matches[1]
	}

	// Парсим зависимости
	if matches := dependRe.FindStringSubmatch(string(content)); len(matches) > 1 {
		deps := parseDependencies(matches[1])
		p.Deps = append(p.Deps, deps...)
		log.Printf("Parsed dependencies for %s: %v", name, deps)
	}

	if matches := slotRe.FindStringSubmatch(string(content)); len(matches) > 1 {
		p.Slot = pkg.ParseSlot(matches[1])
	}

	if matches := iuseRe.FindStringSubmatch(string(content)); len(matches) > 1 {
		flags := strings.Fields(matches[1])
		for _, flag := range flags {
			flag = strings.TrimPrefix(flag, "+")
			flag = strings.TrimPrefix(flag, "-")
			p.UseFlags[flag] = true
		}
	}

	if matches := provideRe.FindStringSubmatch(string(content)); len(matches) > 1 {
		provides := strings.Fields(matches[1])
		for _, prov := range provides {
			p.Provides = append(p.Provides, pkg.Constraint{
				Type: pkg.ConstraintTypeVersion,
				Name: prov,
			})
		}
	}

	return p, nil
}

func (pr *PortageRepository) findEbuildFiles(pkgDir string) ([]string, error) {
	var ebuildFiles []string

	err := filepath.Walk(pkgDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".ebuild") {
			ebuildFiles = append(ebuildFiles, path)
		}
		return nil
	})

	return ebuildFiles, err
}

// Улучшенный парсер зависимостей
func parseDependencies(depString string) []pkg.Constraint {
	log.Printf("Parsing dependencies: %s", depString)

	var deps []pkg.Constraint

	// Регулярка для извлечения зависимостей с операторами
	re := regexp.MustCompile(`(!)?([^! ]+)(?:([<>]=?|=)([^ ]+))?`)
	matches := re.FindAllStringSubmatch(depString, -1)

	for _, m := range matches {
		if len(m) < 3 {
			continue
		}

		name := m[2]
		if name == "" {
			continue
		}

		// Обработка оператора
		if len(m) > 4 && m[3] != "" && m[4] != "" {
			op := m[3]
			version := m[4]

			var versionConstraint *pkg.VersionConstraint
			switch op {
			case ">=":
				versionConstraint = pkg.NewVersionConstraint(pkg.OpGreaterEqual, version)
			case "<=":
				versionConstraint = pkg.NewVersionConstraint(pkg.OpLessEqual, version)
			case ">":
				versionConstraint = pkg.NewVersionConstraint(pkg.OpGreater, version)
			case "<":
				versionConstraint = pkg.NewVersionConstraint(pkg.OpLess, version)
			case "=":
				versionConstraint = pkg.NewVersionConstraint(pkg.OpEqual, version)
			}

			deps = append(deps, pkg.Constraint{
				Type:    pkg.ConstraintTypeVersion,
				Name:    name,
				Version: versionConstraint,
			})
		} else {
			// Зависимость без оператора
			deps = append(deps, pkg.Constraint{
				Type: pkg.ConstraintTypeVersion,
				Name: name,
			})
		}
	}

	return deps
}
