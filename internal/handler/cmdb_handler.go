package handler

import (
	"yunshu/internal/service"
)

type CMDBHandler struct {
	svc *service.CMDBService
}

func NewCMDBHandler(svc *service.CMDBService) *CMDBHandler {
	return &CMDBHandler{svc: svc}
}
