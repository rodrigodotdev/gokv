package persistence

import "github.com/rodrigodotdev/gokv/internal/domain/repository"

type AOFProviderConfig struct {
	Enabled  bool
	FilePath string
}

func NewAOFProvider(cfg AOFProviderConfig) (repository.PersistenceRepository, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	return NewAOF(cfg.FilePath)
}
