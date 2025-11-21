package container

import (
	"github.com/rodrigodotdev/gokv/internal/adapter/handler"
	"github.com/rodrigodotdev/gokv/internal/adapter/protocol"
	"github.com/rodrigodotdev/gokv/internal/domain/repository"
	"github.com/rodrigodotdev/gokv/internal/infraestructure/persistence"
	"github.com/rodrigodotdev/gokv/internal/infraestructure/storage"
	"github.com/rodrigodotdev/gokv/internal/usecase"
)

type Container struct {
	Store          repository.KeyValueRepository
	Persistence    repository.PersistenceRepository
	CommandHandler *usecase.CommandHandler
	TCPHandler     *handler.TCPHandler
	Parser         *protocol.Parser
}

func NewContainer(
	store repository.KeyValueRepository,
	persist repository.PersistenceRepository,
	parser *protocol.Parser,
	commandHandler *usecase.CommandHandler,
	tcpHandler *handler.TCPHandler,
) *Container {
	return &Container{
		Store:          store,
		Persistence:    persist,
		CommandHandler: commandHandler,
		TCPHandler:     tcpHandler,
		Parser:         parser,
	}
}

func InitializeContainer(cfg persistence.AOFProviderConfig) (*Container, func(), error) {
	store := storage.NewStore()

	persist, err := persistence.NewAOFProvider(cfg)
	if err != nil {
		return nil, nil, err
	}

	parser := protocol.NewParser()

	stats := usecase.NewStats(store)

	commandHandler := usecase.NewCommandHandler(store, persist, parser, stats)

	tcpHandler := handler.NewTCPHandler(commandHandler, parser)

	container := NewContainer(store, persist, parser, commandHandler, tcpHandler)

	return container, func() {}, nil
}

func (c *Container) Close() error {
	if c.Persistence != nil {
		return c.Persistence.Close()
	}

	return nil
}
