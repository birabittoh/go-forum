package handlers

import (
	C "goforum/internal/constants"
	"net/http"
	"strconv"
	"time"

	"goforum/internal/models"

	"github.com/gin-gonic/gin"
)

// Admin panel
func (h *Handler) AdminPanel(c *gin.Context) {
	user := h.getCurrentUser(c)

	data := map[string]interface{}{
		"title": "Admin Panel",
		"user":  user,
	}
	C.Tmpl[C.AdminPanelPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

// User management
func (h *Handler) UserList(c *gin.Context) {
	user := h.getCurrentUser(c)

	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	var users []models.User
	query := h.db.Model(&models.User{})

	switch sortBy {
	case "username":
		query = query.Order("username " + order)
	case "email":
		query = query.Order("email " + order)
	case "user_type":
		query = query.Order("user_type " + order)
	case "created_at":
		query = query.Order("created_at " + order)
	default:
		query = query.Order("created_at " + order)
	}

	if err := query.Find(&users).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to load users",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"title":  "User Management",
		"users":  users,
		"user":   user,
		"sortBy": sortBy,
		"order":  order,
	}
	C.Tmpl[C.UserListPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) EditUser(c *gin.Context) {
	currentUser := h.getCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid user ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var targetUser models.User
	if err := h.db.First(&targetUser, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "User not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"title":      "Edit User",
		"user":       currentUser,
		"targetUser": targetUser,
	}
	C.Tmpl[C.EditUserPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	currentUser := h.getCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid user ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var targetUser models.User
	if err := h.db.First(&targetUser, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "User not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	// Update fields
	targetUser.Motto = c.PostForm("motto")
	targetUser.ProfilePicURL = c.PostForm("profile_pic_url")
	targetUser.Signature = c.PostForm("signature")

	if err := h.db.Save(&targetUser).Error; err != nil {
		data := map[string]interface{}{
			"title":      "Edit User",
			"user":       currentUser,
			"targetUser": targetUser,
			"error":      "Failed to update user",
		}
		C.Tmpl[C.EditUserPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *Handler) BanUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid user ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "User not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	reason := c.PostForm("reason")
	duration := c.PostForm("duration")

	user.IsBanned = true
	user.BanReason = reason
	now := time.Now()
	user.BannedAt = &now

	if duration != "permanent" {
		days, err := strconv.Atoi(duration)
		if err == nil && days > 0 {
			until := now.AddDate(0, 0, days)
			user.BannedUntil = &until
		}
	}

	if err := h.db.Save(&user).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to ban user",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *Handler) UnbanUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid user ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "User not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	user.IsBanned = false
	user.BanReason = ""
	user.BannedAt = nil
	user.BannedUntil = nil

	if err := h.db.Save(&user).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to unban user",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *Handler) PromoteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid user ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "User not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	userType := c.PostForm("user_type")
	switch userType {
	case "user":
		user.UserType = models.UserTypeUser
	case "moderator":
		user.UserType = models.UserTypeModerator
	case "admin":
		user.UserType = models.UserTypeAdmin
	}

	if err := h.db.Save(&user).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to update user type",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *Handler) DemoteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid user ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "User not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	if user.UserType > models.UserTypeUser {
		user.UserType = models.UserTypeUser
	}

	if err := h.db.Save(&user).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to demote user",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

// Section management
func (h *Handler) SectionList(c *gin.Context) {
	user := h.getCurrentUser(c)

	var sections []models.Section
	if err := h.db.Order("\"order\" ASC").Find(&sections).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to load sections",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"title":    "Section Management",
		"sections": sections,
		"user":     user,
	}
	C.Tmpl[C.SectionListPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) CreateSection(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	order, _ := strconv.Atoi(c.PostForm("order"))

	if name == "" {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Section name is required",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	section := &models.Section{
		Name:        name,
		Description: description,
		Order:       order,
	}

	if err := h.db.Create(section).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to create section",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/sections")
}

func (h *Handler) DeleteSection(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid section ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	if err := h.db.Delete(&models.Section{}, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to delete section",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/sections")
}

// Category management
func (h *Handler) CategoryList(c *gin.Context) {
	user := h.getCurrentUser(c)

	var categories []models.Category
	if err := h.db.Preload("Section").Order("section_id ASC, \"order\" ASC").Find(&categories).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to load categories",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	var sections []models.Section
	if err := h.db.Order("\"order\" ASC").Find(&sections).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to load sections",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"title":      "Category Management",
		"categories": categories,
		"sections":   sections,
		"user":       user,
	}
	C.Tmpl[C.CategoryListPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	sectionID, _ := strconv.Atoi(c.PostForm("section_id"))
	order, _ := strconv.Atoi(c.PostForm("order"))

	if name == "" || sectionID == 0 {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Category name and section are required",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	category := &models.Category{
		SectionID:   uint(sectionID),
		Name:        name,
		Description: description,
		Order:       order,
	}

	if err := h.db.Create(category).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to create category",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/categories")
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid category ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	if err := h.db.Delete(&models.Category{}, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to delete category",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/categories")
}
