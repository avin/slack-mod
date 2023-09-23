package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	PORT           = 9222
	scriptFilePath = "./injection/script.js"
	styleFilePath  = "./injection/style.css"
	retryInterval  = 1 * time.Second // Интервал между повторными попытками
	maxRetries     = 30              // Максимальное количество повторных попыток

)

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

func getSlackPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	baseDir := filepath.Join(usr.HomeDir, "AppData", "Local", "slack")

	// Получение списка файлов и папок в базовом каталоге Slack
	files, err := os.ReadDir(baseDir)
	if err != nil {
		return "", err
	}

	// Фильтрация списка, чтобы оставить только папки, начинающиеся с "app-"
	appDirs := []string{}
	for _, file := range files {
		if file.IsDir() && strings.HasPrefix(file.Name(), "app-") {
			appDirs = append(appDirs, file.Name())
		}
	}

	if len(appDirs) == 0 {
		return "", fmt.Errorf("no app- directories found in %s", baseDir)
	}

	// Сортировка папок в порядке убывания версии (предполагая, что версии представлены в виде строк)
	sort.Slice(appDirs, func(i, j int) bool {
		return appDirs[i] > appDirs[j] // сортировка в обратном порядке
	})

	// Формирование полного пути к исполняемому файлу Slack
	slackPath := filepath.Join(baseDir, appDirs[0], "slack.exe")
	return slackPath, nil
}

func launchSlack() error {
	slackPath, err := getSlackPath()
	if err != nil {
		return err
	}

	cmd := exec.Command("cmd", "/c", "start", slackPath, fmt.Sprintf("--remote-debugging-port=%d", PORT), "--remote-allow-origins=*", "--startup")
	return cmd.Start()
}

func getWebSocketUrl() (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/json/list", PORT))
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
			fmt.Println("------", title, "------")
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

func waitForWebSocketUrl() (string, error) {
	for retries := 0; retries < maxRetries; retries++ {
		wsUrl, err := getWebSocketUrl()
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

	if err := launchSlack(); err != nil {
		fmt.Println("Error launching Slack:", err)
		return
	}

	// time.Sleep(5 * time.Second)

	wsUrl, err := waitForWebSocketUrl()
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
