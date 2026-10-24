package handler

import (
	"yunshu/internal/service"
)

type DbmgmtHandler struct {
	svc *service.DbmgmtService
}

func NewDbmgmtHandler(svc *service.DbmgmtService) *DbmgmtHandler {
	return &DbmgmtHandler{svc: svc}
}
