package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"my-gin-project/internal/nats"
)

var natsClient *nats.Client

func main() {
	// инициализация Gin
	router := gin.Default()

	// подключение к NATS
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	var err error
	natsClient, err = nats.NewClient(natsURL)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize NATS client: %v", err))
	}
	defer natsClient.Close()

	// регистрация маршрута
	router.GET("/api/core/hello", handleHello)

	router.GET("/api/core/iam", handleIAM)

	// запуск сервера
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server failed: %v\n", err)
		}
	}()

	fmt.Println("API Gateway running on :8080")

	// ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server shutdown failed: %v\n", err)
	}
}

func handleHello(c *gin.Context) {

	outParam := c.Query("out")
	if outParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'out' parameter"})
		return
	}

	requestID := uuid.New().String()

	// формируем сообщение для NATS
	reqMsg := nats.Message{
		RequestID: requestID,
		Cmd:       "hello",
		Meta:      map[string]interface{}{},
		Payload: map[string]interface{}{
			"out": outParam,
		},
	}

	// отправляем запрос в NATS
	respMsg, err := natsClient.Request(c.Request.Context(), "core.income.request", reqMsg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("NATS request failed: %v", err)})
		return
	}

	// извлекаем текст из payload
	payloadMap, ok := respMsg.Payload.(map[string]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid response payload"})
		return
	}

	text, ok := payloadMap["text"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "response payload.text is not a string"})
		return
	}

	// возвращаем клиенту
	c.JSON(http.StatusOK, gin.H{"text": text})
}

func handleIAM(c *gin.Context) {
	// Генерируем UUID для запроса
	requestID := uuid.New().String()

	// Формируем сообщение для NATS
	reqMsg := nats.Message{
		RequestID: requestID,
		Cmd:       "iam",                    // ✅ Новая команда
		Meta:      map[string]interface{}{}, // пустой объект
		Payload:   map[string]interface{}{}, // пустой payload
	}

	// Отправляем запрос в NATS
	respMsg, err := natsClient.Request(c.Request.Context(), "core.income.request", reqMsg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("NATS request failed: %v", err)})
		return
	}

	// Извлекаем имя из payload
	payloadMap, ok := respMsg.Payload.(map[string]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid response payload"})
		return
	}

	name, ok := payloadMap["name"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "response payload.name is not a string"})
		return
	}

	// Возвращаем клиенту
	c.JSON(http.StatusOK, gin.H{"name": name})
}
