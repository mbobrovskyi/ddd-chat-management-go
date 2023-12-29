package api

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/mbobrovskyi/chat-management-go/internal/common/domain/http_error"
	"github.com/mbobrovskyi/chat-management-go/internal/infrastructure/configs"
	"github.com/mbobrovskyi/chat-management-go/internal/infrastructure/logger"
	"maps"
	"net/http"
)

type ErrorHandler interface {
	Handle(*fiber.Ctx, error) error
}

type errorHandler struct {
	cfg *configs.Config
	log logger.Logger
}

func (e *errorHandler) Handle(ctx *fiber.Ctx, err error) error {
	var baseError http_error.HttpError

	switch err.(type) {
	case *fiber.Error:
		var fiberErr *fiber.Error
		errors.As(err, &fiberErr)
		switch fiberErr.Code {
		case http.StatusNotFound:
			baseError = http_error.NewNotFoundError(err.Error())
		}
	case http_error.HttpError:
		errors.As(err, &baseError)
	}

	fmt.Println()

	errResponse := &ErrorResponse{
		Timestamp: baseError.GetTimestamp().Format(configs.DateTimeFormat),
		Code:      baseError.GetCode(),
		Message:   baseError.GetMessage(),
		Metadata:  maps.Clone(baseError.GetMetaData()),
	}

	if baseError.GetHttpStatusCode() >= 500 && e.cfg.Environment == configs.Development {
		errResponse.Metadata["stacktrace"] = baseError.GetStacktrace()
	}

	return ctx.Status(baseError.GetHttpStatusCode()).JSON(errResponse)
}

func NewErrorHandler(cfg *configs.Config, log logger.Logger) ErrorHandler {
	return &errorHandler{
		cfg: cfg,
		log: log,
	}
}