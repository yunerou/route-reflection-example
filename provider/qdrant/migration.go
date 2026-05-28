package qdrant

import (
	"context"
	"fmt"
	"sort"
)

type Migration struct {
	Version int
	Name    string
	Apply   func(context.Context, QdrantProvider) error
	Verify  func(context.Context, QdrantProvider) error
}

func LatestMigrationVersion(migrations []Migration) (int, error) {
	normalized, err := normalizeMigrations(migrations)
	if err != nil {
		return 0, err
	}
	if len(normalized) == 0 {
		return 0, nil
	}
	return normalized[len(normalized)-1].Version, nil
}

func HandleStartupMigration(ctx context.Context, client *client, autoMigration bool, migrations []Migration) error {
	if autoMigration {
		return RunMigration(ctx, client, migrations)
	}
	return VerifyMigration(ctx, client, migrations)
}

func RunMigration(ctx context.Context, client *client, migrations []Migration) error {
	normalized, err := normalizeMigrations(migrations)
	if err != nil {
		return err
	}

	for _, migration := range normalized {
		if migration.Apply == nil {
			return fmt.Errorf("qdrant: migration %s has no apply function", migration.Name)
		}
		if err := migration.Apply(ctx, client); err != nil {
			return fmt.Errorf("qdrant: failed to apply migration %s: %w", migration.Name, err)
		}
		if migration.Verify != nil {
			if err := migration.Verify(ctx, client); err != nil {
				return fmt.Errorf("qdrant: migration %s did not verify after apply: %w", migration.Name, err)
			}
		}
	}

	return nil
}

func VerifyMigration(ctx context.Context, client *client, migrations []Migration) error {
	normalized, err := normalizeMigrations(migrations)
	if err != nil {
		return err
	}

	for _, migration := range normalized {
		if migration.Verify == nil {
			return fmt.Errorf("qdrant: migration %s has no verify function", migration.Name)
		}
		if err := migration.Verify(ctx, client); err != nil {
			return fmt.Errorf("qdrant: migration verification failed for %s: %w", migration.Name, err)
		}
	}

	return nil
}

func normalizeMigrations(migrations []Migration) ([]Migration, error) {
	normalized := make([]Migration, len(migrations))
	copy(normalized, migrations)

	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Version < normalized[j].Version
	})

	seenVersions := make(map[int]string, len(normalized))
	for _, migration := range normalized {
		if migration.Version <= 0 {
			return nil, fmt.Errorf("qdrant: migration %s has invalid version %d", migration.Name, migration.Version)
		}
		if migration.Name == "" {
			return nil, fmt.Errorf("qdrant: migration version %d has empty name", migration.Version)
		}
		if previous, exists := seenVersions[migration.Version]; exists {
			return nil, fmt.Errorf(
				"qdrant: duplicated migration version %d in %s and %s",
				migration.Version, previous, migration.Name,
			)
		}
		seenVersions[migration.Version] = migration.Name
	}

	return normalized, nil
}
