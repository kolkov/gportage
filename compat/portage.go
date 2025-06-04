package compat

import (
    "os"
    "path/filepath"
)

func ConvertEbuildToNative(ebuildPath string) (pkg.PackageSpec, error) {
    // Парсинг традиционных ebuild-файлов
    // и преобразование в нативный формат
}

func MigrateDB(portageDBPath string) error {
    // Конвертация существующей базы данных Portage
    // в новую систему
}