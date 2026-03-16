package authentication

import (
	"colossa-pm/audit"
	"colossa-pm/helpers"
	"colossa-pm/logger"
	"colossa-pm/models"
	"colossa-pm/users"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.Register(input)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrEmailTaken):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, ErrResendTooSoon):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "registration faileds"})
		}
		return
	}

	audit.SetAction(c, "user.registered")
	audit.SetAuthUser(c, uuid.MustParse(resp.UserID), input.FullName)
	audit.SetEntity(c, "user", uuid.MustParse(resp.UserID))
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	var input VerifyEmailInput
	input.UserID = c.Param("userId")
	if input.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.VerifyEmail(input)
	if err != nil {
		switch {
		case errors.Is(err, ErrTokenNotFound):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid or expired OTP"})
		case errors.Is(err, ErrTokenExpired):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		case errors.Is(err, users.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		}
		return
	}

	audit.SetAction(c, "user.email_verified")
	audit.SetAuthUser(c, resp.User.ID, resp.User.FullName)
	audit.SetEntity(c, "user", resp.User.ID)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.Login(input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case errors.Is(err, ErrNotVerified):
			c.JSON(http.StatusForbidden, gin.H{"error": "please verify your email before logging in"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
		return
	}

	audit.SetAction(c, "user.logged_in")
	audit.SetAuthUser(c, resp.User.ID, resp.User.FullName)
	audit.SetEntity(c, "user", resp.User.ID)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RefreshTokens(c *gin.Context) {
	var input RefreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokens, err := h.svc.RefreshTokens(input)
	if err != nil {
		if errors.Is(err, helpers.ErrInvalidToken) || errors.Is(err, ErrTokenExpired) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, users.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user no longer exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token refresh failed"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

func (h *Handler) RequestChangePassword(c *gin.Context) {
	var input RequestChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Always return 200 regardless of whether email exists — prevents enumeration
	resp, requestErr := h.svc.RequestChangePassword(input)

	if requestErr != nil {
		logger.Instance().ErrorMsg("Error with request change password" + requestErr.Error())
	}

	if errors.Is(requestErr, ErrNotVerified) {
		c.JSON(http.StatusConflict, gin.H{"error": ErrNotVerified.Error()})
		return
	}

	audit.SetAction(c, "user.change_password_requested")
	audit.SetAuthUser(c, resp.ID, resp.FullName)
	c.JSON(http.StatusOK, gin.H{"message": "if that email exists, a confirmation code has been sent"})
}

func (h *Handler) ConfirmChangePassword(c *gin.Context) {
	var input ConfirmChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.ConfirmChangePassword(input)
	if err != nil {
		switch {
		case errors.Is(err, ErrTokenNotFound):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid or expired OTP"})
		case errors.Is(err, ErrTokenExpired):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		case errors.Is(err, ErrSamePassword):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, users.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "password change failed"})
		}
		return
	}

	audit.SetAction(c, "user.password_changed")
	audit.SetAuthUser(c, resp.ID, resp.FullName)
	audit.SetDiff(c,
		models.JSON{"password": "[redacted]"},
		models.JSON{"password": "[redacted]"},
	)
	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}
