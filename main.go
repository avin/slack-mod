package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	scriptFilePath = "./injection/script.js"
	styleFilePath  = "./injection/style.css"
	retryInterval  = 1 * time.Second // Интервал между повторными попытками
	maxRetries     = 30              // Максимальное количество повторных попыток

)

func allocateDebuggingPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("unexpected listener address type")
	}

	return addr.Port, nil
}

type Injector struct {
	script string
	style  string
}

func NewInjector() (*Injector, error) {
	script, style, err := readInjection()
	if err != nil {
		return nil, err
	}

	return &Injector{
		script: script,
		style:  style,
	}, nil
}

func (i *Injector) Inject(ws *websocket.Conn) error {
	if err := ws.WriteJSON(map[string]interface{}{
		"id":     1,
		"method": "Runtime.evaluate",
		"params": map[string]interface{}{"expression": i.script},
	}); err != nil {
		return err
	}

	encodedStyle := base64.StdEncoding.EncodeToString([]byte(i.style))

	styleInjection := fmt.Sprintf(`document.body.appendChild(document.createElement('style')).textContent = atob('%s')`, encodedStyle)

	return ws.WriteJSON(map[string]interface{}{
		"id":     2,
		"method": "Runtime.evaluate",
		"params": map[string]interface{}{"expression": styleInjection},
	})
}

func getWebSocketUrl(port int) (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/json/list", port))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data []map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	for _, item := range data {
		title, ok := item["title"].(string)
		if ok && strings.HasSuffix(title, "Slack") {
			webSocketDebuggerUrl, ok := item["webSocketDebuggerUrl"].(string)
			if ok {
				return webSocketDebuggerUrl, nil
			}
		}
	}

	return "", errors.New("no WebSocket URL found for Slack")
}

func createWebSocketConnection(url string, onOpen func(ws *websocket.Conn)) (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	onOpen(c)

	return c, nil
}

func readInjection() (string, string, error) {
	scriptBytes, err := ioutil.ReadFile(scriptFilePath)
	if err != nil {
		return "", "", err
	}

	styleBytes, err := ioutil.ReadFile(styleFilePath)
	if err != nil {
		return "", "", err
	}

	return string(scriptBytes), string(styleBytes), nil
}

func waitForWebSocketUrl(port int) (string, error) {
	for retries := 0; retries < maxRetries; retries++ {
		wsUrl, err := getWebSocketUrl(port)
		if err == nil && wsUrl != "" {
			return wsUrl, nil // URL найден, возвращаем его
		}

		// Если URL не найден или произошла ошибка, ждем и повторяем попытку
		time.Sleep(retryInterval)
	}
	return "", errors.New("failed to get WebSocket URL after retrying")
}

func main() {
	injector, err := NewInjector()
	if err != nil {
		fmt.Println("Error initializing injector:", err)
		return
	}

	port, err := allocateDebuggingPort()
	if err != nil {
		fmt.Println("Error allocating debugging port:", err)
		return
	}

	if err := launchSlack(port); err != nil {
		fmt.Println("Error launching Slack:", err)
		return
	}

	wsUrl, err := waitForWebSocketUrl(port)
	if err != nil {
		fmt.Println("Error getting WebSocket URL:", err)
		return
	}

	conn, err := createWebSocketConnection(wsUrl, func(ws *websocket.Conn) {
		if err := injector.Inject(ws); err != nil {
			fmt.Println("Error injecting:", err)
		}
	})
	if err != nil {
		fmt.Println("Error creating WebSocket connection:", err)
		return
	}
	defer conn.Close()
}
