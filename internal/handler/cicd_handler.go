package handler

import (
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service/cicd"

	"github.com/gin-gonic/gin"
)

type CicdHandler struct {
	svc *cicd.Service
}

func NewCicdHandler(svc *cicd.Service) *CicdHandler {
	return &CicdHandler{svc: svc}
}

func (h *CicdHandler) cicdActor(c *gin.Context) *auth.CurrentUser {
	actor, _ := auth.CurrentUserFromContext(c)
	return actor
}

func (h *CicdHandler) requireCicdServiceAccess(c *gin.Context, projectID, serviceID uint, need string) bool {
	if err := h.svc.AssertCicdAccess(c.Request.Context(), projectID, serviceID, h.cicdActor(c), need); err != nil {
		response.Error(c, err)
		return false
	}
	return true
}

func reviewerFromContext(c *gin.Context) (*uint, string) {
	if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
		name := u.Username
		if name == "" {
			name = u.Nickname
		}
		return &u.ID, name
	}
	return nil, ""
}
