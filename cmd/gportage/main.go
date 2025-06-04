package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kolkov/gportage/internal/repo"
	"github.com/kolkov/gportage/internal/solver"
	"github.com/kolkov/gportage/internal/state"
	"github.com/spf13/cobra"
)

var (
	useMockRepo bool
)

var (
	repoPath    = "/var/db/repos/gentoo"
	snapshotDir = "/.snapshots"
	fsType      = "btrfs"
)

var rootCmd = &cobra.Command{
	Use:   "gportage",
	Short: "Next-generation package manager for Gentoo",
}

var resolveCmd = &cobra.Command{
	Use:   "resolve [package...]",
	Short: "Resolve package dependencies",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var r repo.Repository
		var err error

		if !useMockRepo {
			// Преобразуем путь в абсолютный только для реального репозитория
			absRepoPath, absErr := filepath.Abs(repoPath)
			if absErr != nil {
				log.Fatalf("Invalid repository path: %v", absErr)
			}
			log.Printf("Using repository: %s", absRepoPath)

			r, err = repo.NewPortageRepository(absRepoPath)
			if err != nil {
				log.Fatalf("Repository error: %v", err)
			}
		} else {
			log.Printf("Using mock repository")
			r = repo.NewMockRepository()
		}

		resolver := solver.NewResolver(r)
		solution, err := resolver.Resolve(args)
		if err != nil {
			log.Fatalf("Resolution failed: %v", err)
		}

		if len(solution) == 0 {
			log.Println("No packages found in solution")
			return
		}

		fmt.Println("Dependency solution:")
		for name, pkg := range solution {
			fmt.Printf("- %s-%s [slot:%s]\n", name, pkg.Version, pkg.Slot.Name)
		}
	},
}

var installCmd = &cobra.Command{
	Use:   "install [package...]",
	Short: "Install packages with transaction safety",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Преобразуем путь в абсолютный
		absRepoPath, err := filepath.Abs(repoPath)
		if err != nil {
			log.Fatalf("Invalid repository path: %v", err)
		}

		// Инициализация менеджера снапшотов
		sm := state.NewSnapshotManager(snapshotDir, fsType)

		// Создаем снапшот перед изменениями
		snapshotID, err := sm.CreateSnapshot("/")
		if err != nil {
			log.Fatalf("Failed to create snapshot: %v", err)
		}
		log.Printf("Created system snapshot: %s", snapshotID)

		// Разрешаем зависимости
		var r repo.Repository
		if !useMockRepo {
			r, err = repo.NewPortageRepository(absRepoPath)
			if err != nil {
				log.Fatalf("Repository error: %v", err)
			}
		} else {
			r = repo.NewMockRepository()
		}
		resolver := solver.NewResolver(r)
		solution, err := resolver.Resolve(args)
		if err != nil {
			log.Fatalf("Dependency resolution failed: %v", err)
		}

		// Процесс установки (заглушка)
		log.Println("Installing packages:")
		for name, pkg := range solution {
			log.Printf("- %s-%s (slot: %s)", name, pkg.Version, pkg.Slot)
			// Реальная установка будет здесь
		}

		// Если установка прошла успешно
		log.Println("Installation completed successfully")

		// В реальности: очистка старых снапшотов, обновление конфигурации и т.д.
	},
}

func init() {
	// Флаги для команды install
	installCmd.Flags().StringVar(&repoPath, "repo", repoPath, "Path to Portage repository")
	installCmd.Flags().StringVar(&snapshotDir, "snapshot-dir", snapshotDir, "Snapshot directory")
	installCmd.Flags().StringVar(&fsType, "fs-type", fsType, "Filesystem type (btrfs or zfs)")
	resolveCmd.Flags().StringVar(&repoPath, "repo", repoPath, "Path to Portage repository")
	resolveCmd.Flags().BoolVar(&useMockRepo, "mock", false, "Use mock repository")
}

func main() {
	rootCmd.AddCommand(resolveCmd, installCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
