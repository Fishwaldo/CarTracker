package web

import (
	"net/http"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/websocket"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("web.port", 8080)
}

type WebServer struct {
	Echo *echo.Echo
	log  logr.Logger
}
var (
	Web WebServer
	connectionPool = struct {
		sync.RWMutex
		connections map[*websocket.Conn]struct{}
	}{
		connections: make(map[*websocket.Conn]struct{}),
	}
)

func (web *WebServer) Start(log logr.Logger) {
	web.log = log
	web.Echo = echo.New()
	// Middleware
	web.Echo.Use(middleware.Logger())
	web.Echo.Use(middleware.Recover())

	// Routes
	web.Echo.GET("/", web.homePage)
	web.Echo.GET("/ws", web.WebSocket)

	// Start server
	go func() {
		port := viper.GetInt("web.port")
		bind := fmt.Sprintf(":%d", port)
		err := web.Echo.Start(bind)
		if err != nil {
			log.Error(err, "Web Server Error")
		}
	}()
}

func (web *WebServer) GetEchoServer() (*echo.Echo) {
	return Web.Echo
}

func  (web *WebServer) homePage(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func (web *WebServer) WebSocket(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		web.log.Info("Opened WebSocket")
        connectionPool.Lock()
        connectionPool.connections[ws] = struct{}{}

        defer func(connection *websocket.Conn){
            connectionPool.Lock()
            delete(connectionPool.connections, connection)
            connectionPool.Unlock()
        }(ws)

        connectionPool.Unlock()
		defer ws.Close()
		for {
			// Read
			msg := ""
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error(err)
				break;
			}
			web.log.Info("%s\n", msg)
		}
		web.log.Info("Closed WebSocket")
	}).ServeHTTP(c.Response(), c.Request())
	return nil	
}

func (web *WebServer) Broadcast(message string) {
	connectionPool.RLock()
	for connection := range connectionPool.connections {
		err := websocket.Message.Send(connection, message)
		if err != nil {
			web.log.Error(err, "Broadcast Failure")
		}
	}
	connectionPool.RUnlock()
}