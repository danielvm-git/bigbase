package messaging

import (
	"encoding/json"
	"net/http"

	"github.com/danielvm/bigbase/kernel"
)

func (m *Messaging) handleEmail(w http.ResponseWriter, r *http.Request) {
	if !m.handleMethod(w, r, "POST") {
		return
	}

	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if errs := collectErrors(
		validateRequired(req.To, "to"),
		validateRequired(req.Subject, "subject"),
		validateRequired(req.Body, "body"),
	); errs != "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": errs})
		return
	}

	msg, err := m.send(r.Context(), "email", req.To, req.Subject, req.Body)
	if err != nil {
		m.logger.Error("send email", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusCreated, msg)
}

func (m *Messaging) handleSMS(w http.ResponseWriter, r *http.Request) {
	if !m.handleMethod(w, r, "POST") {
		return
	}

	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	body := req.Message
	if body == "" {
		body = req.Body
	}

	if errs := collectErrors(
		validateRequired(req.To, "to"),
		validateRequired(body, "message"),
	); errs != "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": errs})
		return
	}

	msg, err := m.send(r.Context(), "sms", req.To, "", body)
	if err != nil {
		m.logger.Error("send sms", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusCreated, msg)
}

func (m *Messaging) handlePush(w http.ResponseWriter, r *http.Request) {
	if !m.handleMethod(w, r, "POST") {
		return
	}

	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	body := req.Body
	if body == "" {
		body = req.Message
	}

	if errs := collectErrors(
		validateRequired(req.Token, "token"),
		validateRequired(req.Title, "title"),
		validateRequired(body, "body"),
	); errs != "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": errs})
		return
	}

	subject := req.Title
	msg, err := m.send(r.Context(), "push", req.Token, subject, body)
	if err != nil {
		m.logger.Error("send push", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusCreated, msg)
}

func (m *Messaging) handleList(w http.ResponseWriter, r *http.Request) {
	if !m.handleMethod(w, r, "GET") {
		return
	}

	rows, err := m.db.QueryContext(r.Context(),
		"SELECT id, channel, to_addr, COALESCE(subject,''), body, status, created_at FROM messages ORDER BY created_at DESC")
	if err != nil {
		m.logger.Error("list messages", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	messages := make([]Message, 0)
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.Channel, &msg.ToAddr, &msg.Subject, &msg.Body, &msg.Status, &msg.CreatedAt); err != nil {
			m.logger.Error("scan message", "error", err)
			kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		m.logger.Error("rows iteration", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": messages})
}
