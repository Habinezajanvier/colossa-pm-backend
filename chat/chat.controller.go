package chat

import (
	"errors"
	"net/http"

	"colossa-pm/audit"
	"colossa-pm/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin validates the request origin against ALLOWED_ORIGINS env var.
	// In development with ALLOWED_ORIGINS unset, all origins are allowed.
	// CheckOrigin: func(r *http.Request) bool {
	// 	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	// 	if allowedOrigins == "" {
	// 		return true
	// 	}
	// 	origin := r.Header.Get("Origin")
	// 	for _, allowed := range strings.Split(allowedOrigins, ",") {
	// 		if strings.TrimSpace(allowed) == origin {
	// 			return true
	// 		}
	// 	}
	// 	return false
	// },
}

type Handler struct {
	svc Service
	hub *Hub
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc, hub: GetHub()}
}

// --- Conversations ---

func (h *Handler) StartConversation(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input StartConversationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conv, err := h.svc.StartConversation(userID, input)
	if err != nil {
		if errors.Is(err, ErrNotWorkspaceMember) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start conversation"})
		return
	}

	audit.SetAction(c, "chat.conversation_started")
	audit.SetEntity(c, "conversation", conv.ID)
	c.JSON(http.StatusCreated, conv)
}

func (h *Handler) GetConversations(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var params helpers.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	params.Normalize()

	result, err := h.svc.GetConversations(userID, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch conversations"})
		return
	}

	audit.SetAction(c, "chat.conversations_listed")
	c.JSON(http.StatusOK, result)
}

// --- Messages ---

func (h *Handler) GetDMConversations(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var params helpers.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	params.Normalize()

	result, err := h.svc.GetDMConversations(userID, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch DM conversations"})
		return
	}

	audit.SetAction(c, "chat.dm_conversations_listed")
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetParticipants(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversationId"})
		return
	}

	participants, err := h.svc.GetParticipants(userID, conversationID)
	if err != nil {
		if errors.Is(err, ErrNotParticipant) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch participants"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": participants})
}

func (h *Handler) GetMessages(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversationId"})
		return
	}

	var params helpers.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	params.Normalize()

	result, err := h.svc.GetMessages(userID, conversationID, params)
	if err != nil {
		if errors.Is(err, ErrNotParticipant) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetReplies(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversationId"})
		return
	}

	parentID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid messageId"})
		return
	}

	replies, err := h.svc.GetReplies(userID, conversationID, parentID)
	if err != nil {
		if errors.Is(err, ErrNotParticipant) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch replies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": replies})
}

func (h *Handler) SendMessage(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Parse multipart form — max 50MB total
	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	body := c.PostForm("body")
	parentID := c.PostForm("parentId")

	var bodyPtr *string
	if body != "" {
		bodyPtr = &body
	}
	var parentIDPtr *string
	if parentID != "" {
		parentIDPtr = &parentID
	}

	files := c.Request.MultipartForm.File["files"]

	input := SendMessageInput{
		ConversationID: c.Param("conversationId"),
		Body:           bodyPtr,
		ParentID:       parentIDPtr,
		Files:          files,
	}

	msg, err := h.svc.SendMessage(userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotParticipant):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, ErrConversationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send message"})
		}
		return
	}

	audit.SetAction(c, "chat.message_sent")
	audit.SetEntity(c, "conversation", msg.ConversationID)
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) DeleteMessage(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid messageId"})
		return
	}

	if err := h.svc.DeleteMessage(userID, messageID); err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete message"})
		return
	}

	audit.SetAction(c, "chat.message_deleted")
	c.JSON(http.StatusOK, gin.H{"message": "message deleted"})
}

// --- Reactions ---

func (h *Handler) AddReaction(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid messageId"})
		return
	}

	var input AddReactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.AddReaction(userID, messageID, input); err != nil {
		if errors.Is(err, ErrAlreadyReacted) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add reaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reaction added"})
}

func (h *Handler) RemoveReaction(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid messageId"})
		return
	}

	emoji := c.Param("emoji")
	if emoji == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "emoji is required"})
		return
	}

	if err := h.svc.RemoveReaction(userID, messageID, emoji); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove reaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reaction removed"})
}

// --- WebSocket ---

func (h *Handler) ServeWS(c *gin.Context) {
	// Browsers cannot send custom Authorization headers on WebSocket connections.
	// Token must be passed as a query param: ws://host/ws/:id?token=<access_token>
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token query param required"})
		return
	}

	claims, err := helpers.ValidateAccessToken(token)
	if err != nil {
		if errors.Is(err, helpers.ErrTokenExpired) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token has expired"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Reject refresh tokens — only access tokens are valid here
	if claims.TokenType != helpers.AccessToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access token required"})
		return
	}

	userID := claims.UserID

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversationId"})
		return
	}

	// Verify participant before upgrading — once upgraded we can no longer send JSON errors
	if err := h.svc.VerifyParticipant(userID, conversationID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Upgrade HTTP → WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	cl := &client{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 256),
	}

	h.hub.join <- &roomJoin{conversationID: conversationID, client: cl}

	go cl.writePump()

	// readPump — keeps connection alive, handles client disconnect
	defer func() {
		h.hub.leave <- &roomLeave{conversationID: conversationID, client: cl}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// Incoming WS messages are ignored — sending is done via REST
	}
}
