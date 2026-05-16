package handler

import (
	"strconv"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/dto"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/middleware"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/response"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandler menangani endpoint user management (list, detail, suspend, activate).
type UserHandler struct {
	list      usecase.ListUsersUseCase
	detail    usecase.GetUserDetailUseCase
	suspend   usecase.SuspendUserUseCase
	activate  usecase.ActivateUserUseCase
}

// NewUserHandler membuat instance.
func NewUserHandler(
	list usecase.ListUsersUseCase,
	detail usecase.GetUserDetailUseCase,
	suspend usecase.SuspendUserUseCase,
	activate usecase.ActivateUserUseCase,
) *UserHandler {
	return &UserHandler{list: list, detail: detail, suspend: suspend, activate: activate}
}

// List menangani GET /v1/admin/users.
func (h *UserHandler) List(c *gin.Context) {
	page, pageSize := paginationFromQuery(c)
	filter := repository.UserListFilter{
		Query:  c.Query("q"),
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
	if v := c.Query("is_active"); v != "" {
		b := v == "true" || v == "1"
		filter.IsActive = &b
	}
	if v := c.Query("verified"); v != "" {
		b := v == "true" || v == "1"
		filter.EmailVerified = &b
	}
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.CreatedFrom = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.CreatedTo = &t
		}
	}

	in := usecase.ListUsersInput{Filter: filter}
	if v := c.Query("tier"); v != "" {
		in.Tier = &v
	}

	out, err := h.list.Execute(c.Request.Context(), in)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	items := make([]dto.UserListItem, 0, len(out.Users))
	for _, u := range out.Users {
		items = append(items, dto.UserListItem{
			ID:                 u.ID.String(),
			Email:              u.Email,
			Username:           u.Username,
			IsActive:           u.IsActive,
			EmailVerified:      u.EmailVerified,
			SubscriptionTier:   u.SubscriptionTier,
			SubscriptionStatus: u.SubscriptionStatus,
			CreatedAt:          u.CreatedAt,
			LastLoginAt:        u.LastLoginAt,
		})
	}
	response.OKMeta(c, items, dto.PaginationMeta{Page: page, PageSize: pageSize, Total: out.Total})
}

// Detail menangani GET /v1/admin/users/:id.
func (h *UserHandler) Detail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, domainerrors.ErrValidation.Code, "ID user tidak valid.")
		return
	}
	detail, err := h.detail.Execute(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	subID := ""
	if detail.SubscriptionID != nil {
		subID = detail.SubscriptionID.String()
	}
	var subIDPtr *string
	if subID != "" {
		subIDPtr = &subID
	}

	response.OK(c, dto.UserDetailDTO{
		UserListItem: dto.UserListItem{
			ID:                 detail.ID.String(),
			Email:              detail.Email,
			Username:           detail.Username,
			IsActive:           detail.IsActive,
			EmailVerified:      detail.EmailVerified,
			SubscriptionTier:   detail.SubscriptionTier,
			SubscriptionStatus: detail.SubscriptionStatus,
			CreatedAt:          detail.CreatedAt,
			LastLoginAt:        detail.LastLoginAt,
		},
		SubscriptionID:        subIDPtr,
		SubscriptionStartedAt: detail.SubscriptionStartedAt,
		SubscriptionEndsAt:    detail.SubscriptionEndsAt,
		SubscriptionPriceIDR:  detail.SubscriptionPriceIDR,
	})
}

// Suspend menangani POST /v1/admin/users/:id/suspend.
func (h *UserHandler) Suspend(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, domainerrors.ErrValidation.Code, "ID user tidak valid.")
		return
	}
	var req dto.SuspendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, domainerrors.ErrValidation.Code, domainerrors.ErrValidation.Message)
		return
	}
	adminID, err := adminIDFromContext(c)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	if err := h.suspend.Execute(c.Request.Context(), usecase.SuspendUserInput{
		AdminID:      adminID,
		TargetUserID: targetID,
		Reason:       req.Reason,
	}); err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "user di-suspend"})
}

// Activate menangani POST /v1/admin/users/:id/activate.
func (h *UserHandler) Activate(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, domainerrors.ErrValidation.Code, "ID user tidak valid.")
		return
	}
	var req dto.ActivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, domainerrors.ErrValidation.Code, domainerrors.ErrValidation.Message)
		return
	}
	adminID, err := adminIDFromContext(c)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	if err := h.activate.Execute(c.Request.Context(), usecase.ActivateUserInput{
		AdminID:      adminID,
		TargetUserID: targetID,
		Reason:       req.Reason,
	}); err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "user diaktifkan"})
}

// Helpers ----------------------------------------------------------------

func paginationFromQuery(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 25
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.Query("page_size")); err == nil && v > 0 && v <= 100 {
		pageSize = v
	}
	return
}

func adminIDFromContext(c *gin.Context) (uuid.UUID, error) {
	idStr := c.GetString(middleware.ContextKeyAdminID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, domainerrors.ErrUnauthorized
	}
	return id, nil
}
