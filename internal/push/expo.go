// Package push envía notificaciones push a los dispositivos mobile vía la
// Expo Push API (https://exp.host/--/api/v2/push/send).
//
// TAREA (mobile): "notificaciones push para alertas". Se eligió la Expo
// Push API en vez de llamar directo al Firebase Admin SDK porque el
// cliente mobile (mobile/src/notifications/push.ts) obtiene un Expo push
// token vía expo-notifications, que ya envuelve el token nativo FCM/APNs
// — Expo se encarga de la entrega final a Google/Apple. Esto permite
// probar push end-to-end (backend -> Expo -> dispositivo) sin que este
// backend necesite credenciales propias de un proyecto Firebase (que el
// repo no tiene configurado, ver mobile/README.md). Si en el futuro se
// necesita Firebase Admin SDK directo (p.ej. bypass de Expo), este
// package es el único punto de reemplazo — nadie más arma el request de
// push.
package push

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

const expoPushURL = "https://exp.host/--/api/v2/push/send"

type Client struct {
	httpClient *http.Client
	endpoint   string
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 10 * time.Second}, endpoint: expoPushURL}
}

type message struct {
	To    string            `json:"to"`
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Sound string            `json:"sound,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
}

// IsExpoPushToken filtra tokens que claramente no son un Expo push token
// (ej. basura, o un token FCM/APNs crudo mandado por error) antes de
// gastar una llamada a la Expo Push API con ellos. No es una validación
// exhaustiva (Expo también acepta "ExpoPushToken[...]" legado), solo
// descarta lo obviamente inválido.
func IsExpoPushToken(token string) bool {
	return strings.HasPrefix(token, "ExponentPushToken[") || strings.HasPrefix(token, "ExpoPushToken[")
}

// Send manda title/body a cada token en tokens. Best-effort: igual que
// ws.Hub.Broadcast, un fallo de push no debe tumbar ni retrasar la
// ingesta de sensores que lo disparó — solo se loggea. Pensado para
// llamarse en su propia goroutine (ver broadcastIngestResult en
// internal/httpapi/sensors_handlers.go).
func (c *Client) Send(tokens []string, title, body string, data map[string]string) {
	valid := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if IsExpoPushToken(t) {
			valid = append(valid, t)
		}
	}
	if len(valid) == 0 {
		return
	}

	messages := make([]message, len(valid))
	for i, t := range valid {
		messages[i] = message{To: t, Title: title, Body: body, Sound: "default", Data: data}
	}

	payload, err := json.Marshal(messages)
	if err != nil {
		log.Printf("push: error serializando mensajes: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		log.Printf("push: error armando request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("push: error enviando a Expo: %v", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		log.Printf("push: Expo respondió %s", res.Status)
	}
}
