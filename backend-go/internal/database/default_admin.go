package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/models"
)

// KnownDefaultPasswords are passwords shipped in old examples, tutorials or
// earlier versions of this project. An admin account still using one of them
// is an open door.
var KnownDefaultPasswords = []string{"admin123", "admin", "password", "changeme", "devopslab"}

// AdminsWithDefaultPassword returns the usernames of admin accounts whose
// password hash matches one of KnownDefaultPasswords. bcrypt comparisons run
// in parallel (cost 12 is ~0.25s each).
func AdminsWithDefaultPassword(ctx context.Context) ([]string, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var admins []models.User
	if err := DB.WithContext(ctx).Select("id", "username", "password").
		Where("role = ?", models.RoleAdmin).Find(&admins).Error; err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	return adminsWithDefaultPassword(admins), nil
}

func adminsWithDefaultPassword(admins []models.User) []string {
	hits := make([]bool, len(admins))
	var wg sync.WaitGroup
	for i := range admins {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, hits[i] = matchesDefaultPassword(admins[i].Password)
		}(i)
	}
	wg.Wait()
	var out []string
	for i, hit := range hits {
		if hit {
			out = append(out, admins[i].Username)
		}
	}
	return out
}

// matchesDefaultPassword reports which known default password (if any) hash belongs to.
func matchesDefaultPassword(hash string) (string, bool) {
	if hash == "" {
		return "", false
	}
	for _, p := range KnownDefaultPasswords {
		if auth.CheckPassword(p, hash) {
			return p, true
		}
	}
	return "", false
}
