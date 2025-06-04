package repo

import (
	"github.com/kolkov/gportage/internal/pkg"
)

type Repository interface {
	LoadPackages(names []string) ([]*pkg.Package, error)
	LoadPackage(name string) (*pkg.Package, error)
}
