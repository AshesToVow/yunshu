package handler

import (
	dbmgmtsvc "yunshu/internal/service/dbmgmt"
)

type DbmgmtHandler struct {
	svc *dbmgmtsvc.Service
}

func NewDbmgmtHandler(svc *dbmgmtsvc.Service) *DbmgmtHandler {
	return &DbmgmtHandler{svc: svc}
}
