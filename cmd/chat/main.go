package main

import (
	"context"
	"fmt"
	"github.com/mbobrovskyi/chat-management-go/configs"
	chatapplication "github.com/mbobrovskyi/chat-management-go/internal/chat/application/http"
	chatpubsub "github.com/mbobrovskyi/chat-management-go/internal/chat/application/pubsub"
	"github.com/mbobrovskyi/chat-management-go/internal/chat/application/websocket"
	chatdomain "github.com/mbobrovskyi/chat-management-go/internal/chat/domain"
	"github.com/mbobrovskyi/chat-management-go/internal/chat/domain/chat"
	"github.com/mbobrovskyi/chat-management-go/internal/chat/domain/message"
	repositories2 "github.com/mbobrovskyi/chat-management-go/internal/chat/infrastructure/repositories"
	"github.com/mbobrovskyi/chat-management-go/internal/common/api"
	"github.com/mbobrovskyi/chat-management-go/internal/common/domain/connector"
	"github.com/mbobrovskyi/chat-management-go/internal/common/domain/pubsub/publisher"
	"github.com/mbobrovskyi/chat-management-go/internal/common/domain/pubsub/subscriber"
	"github.com/mbobrovskyi/chat-management-go/internal/infrastructure/database/redis"
	"github.com/mbobrovskyi/chat-management-go/internal/infrastructure/logger/logrus"
	"github.com/mbobrovskyi/chat-management-go/internal/infrastructure/server"
	"github.com/mbobrovskyi/chat-management-go/internal/user/contracts"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := configs.NewConfig()
	if err != nil {
		panic(fmt.Errorf("error on init config: %w", err))
	}

	fileVersion, err := os.ReadFile("VERSION")
	if err != nil {
		panic(fmt.Errorf("error on read VERSION file: %w", err))
	}

	version := string(fileVersion)

	log, err := logrus.NewLogger(cfg.LogLevel)
	if err != nil {
		panic(fmt.Errorf("error on init logger: %w", err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	//dbConn, err := postgres.NewPostgres(ctx, cfg.PostgresUri)
	//if err != nil {
	//	log.Fatal(fmt.Errorf("error on connection to postgres: %w", err))
	//}

	redisClient, err := redis.NewRedis(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDb)
	if err != nil {
		log.Fatal(fmt.Errorf("error on connection to redis: %w", err))
	}

	chatPublisher := publisher.NewPublisher(redisClient, cfg.ChatPubSubPrefix)

	chatRepository := repositories2.NewChatRepository()
	messageRepository := repositories2.NewMessageRepository()

	chatService := chat.NewService(chatRepository)
	messageService := message.NewMessageService(messageRepository, chatPublisher)

	chatEventHandler := websocket.NewMessageEventHandler(messageService)
	chatConnector := connector.NewConnector(chatEventHandler, connector.Config{Logger: log})

	chatSubscriberHandler := chatpubsub.NewChatSubscriberHandler(messageService, chatConnector)
	chatSubscriber := subscriber.NewSubscriber(log, redisClient, chatSubscriberHandler, cfg.ChatPubSubPrefix)

	userContract := contracts.NewUserContract()

	mainController := api.NewMainController(version)
	authMiddleware := api.NewAuthMiddleware(userContract)
	chatController := chatapplication.NewChatController(authMiddleware, chatService, messageService, chatConnector)

	httpServer := server.NewHttpServer(
		cfg,
		log,
		api.NewErrorHandler(cfg, log).Handle,
		[]server.Controller{mainController, chatController},
	)

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		if err := chatConnector.Start(ctx); err != nil {
			log.Errorf("Error on running connector: %s", err.Error())
			return err
		}

		log.Info("Chat connector gracefully stopped")

		return nil
	})

	eg.Go(func() error {
		if err := chatSubscriber.Start(ctx, chatdomain.GetAllPubSubEventTypes()); err != nil {
			log.Errorf("Error on running pubsub subscriber: %s", err.Error())
			return err
		}

		log.Info("Chat subscriber gracefully stopped")

		return nil
	})

	eg.Go(func() error {
		if err := httpServer.Start(ctx); err != nil {
			log.Errorf("Error on running http server: %s", err.Error())
			return err
		}

		log.Info("HTTP server gracefully stopped")

		return nil
	})

	if err = eg.Wait(); err != nil {
		log.Error(err)
	}
}
