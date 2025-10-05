package handlers

import (
	"bytes"
	C "goforum/internal/constants"
	"goforum/internal/database"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goforum/internal/models"

	"github.com/gin-gonic/gin"
)

func renderError(c *gin.Context, message string, status int) error {
	config, _ := c.Get("config")
	data := map[string]any{
		"title":   "Error",
		"message": message,
		"config":  config,
	}
	c.Status(status)
	return C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
}

func renderTemplateStatus(c *gin.Context, data map[string]any, templatePath string, status int) error {
	t, ok := C.Tmpl[templatePath]
	if !ok {
		return renderError(c, "Template not found: "+templatePath, http.StatusInternalServerError)
	}

	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return renderError(c, err.Error(), http.StatusInternalServerError)
	}

	_, err := io.Copy(c.Writer, buf)
	c.Status(status)
	return err
}

func renderTemplate(c *gin.Context, data map[string]any, templatePath string) error {
	return renderTemplateStatus(c, data, templatePath, http.StatusOK)
}

// Admin panel
func (h *Handler) AdminPanel(c *gin.Context) {
	user := h.getCurrentUser(c)

	users, err := C.Cache.CountAllUsers(h.db)
	if err != nil {
		renderError(c, "Failed to load stats", http.StatusInternalServerError)
		return
	}
	topics, err := C.Cache.CountAllTopics(h.db)
	if err != nil {
		renderError(c, "Failed to load stats", http.StatusInternalServerError)
		return
	}
	replies, err := C.Cache.CountAllReplies(h.db)
	if err != nil {
		renderError(c, "Failed to load stats", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"title":  "Admin Panel",
		"user":   user,
		"config": h.config,

		"users":   users,
		"topics":  topics,
		"replies": replies,
	}
	renderTemplate(c, data, C.AdminPanelPath)
}

func (h *Handler) Backup(c *gin.Context) {
	user := h.getCurrentUser(c)

	data := map[string]any{
		"title":     "Import/Export Data",
		"user":      user,
		"config":    h.config,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	renderTemplate(c, data, C.BackupPath)
}

func (h *Handler) ExportBackup(c *gin.Context) {
	data, err := database.ExportJSON(h.db)
	if err != nil {
		renderError(c, "Failed to export data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	siteName := strings.ReplaceAll(strings.ToLower(h.config.SiteName), " ", "-")
	filename := siteName + "_" + time.Now().Format("20060102150405") + ".json"

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/json", data)
}

func (h *Handler) ImportBackup(c *gin.Context) {
	file, err := c.FormFile("backup_file")
	if err != nil {
		renderError(c, "Failed to read uploaded file: "+err.Error(), http.StatusBadRequest)
		return
	}

	f, err := file.Open()
	if err != nil {
		renderError(c, "Failed to open uploaded file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		renderError(c, "Failed to read uploaded file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := database.ImportJSON(h.db, data); err != nil {
		renderError(c, "Failed to import data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin")
}

// User management
func (h *Handler) UserList(c *gin.Context) {
	user := h.getCurrentUser(c)

	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	// Pagination
	pageSize := 20
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	totalUsers, err := C.Cache.CountAllUsers(h.db)
	if err != nil {
		renderError(c, "Failed to load users", http.StatusInternalServerError)
		return
	}
	totalPages := int((totalUsers + int64(pageSize) - 1) / int64(pageSize))

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

	if err := query.Limit(pageSize).Offset((page - 1) * pageSize).Find(&users).Error; err != nil {
		renderError(c, "Failed to load users", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"title":      "User Management",
		"users":      users,
		"user":       user,
		"sortBy":     sortBy,
		"order":      order,
		"page":       page,
		"totalPages": totalPages,
		"config":     h.config,
	}
	renderTemplate(c, data, C.UserListPath)
}

func (h *Handler) EditUser(c *gin.Context) {
	currentUser := h.getCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var targetUser models.User
	if err := h.db.First(&targetUser, id).Error; err != nil {
		renderError(c, "User not found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"title":      "Edit User",
		"user":       currentUser,
		"targetUser": targetUser,
		"config":     h.config,
	}
	renderTemplate(c, data, C.EditUserPath)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	currentUser := h.getCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var targetUser models.User
	if err := h.db.First(&targetUser, id).Error; err != nil {
		renderError(c, "User not found", http.StatusNotFound)
		return
	}

	// Update fields
	targetUser.Motto = c.PostForm("motto")
	targetUser.ProfilePicURL, _ = strings.CutPrefix(c.PostForm("profile_pic_url"), h.config.ProfilePicsBaseURL)
	targetUser.Signature = c.PostForm("signature")

	if err := h.db.Save(&targetUser).Error; err != nil {
		data := map[string]any{
			"title":      "Edit User",
			"user":       currentUser,
			"targetUser": targetUser,
			"error":      "Failed to update user",
			"config":     h.config,
		}
		renderTemplate(c, data, C.EditUserPath)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *Handler) BanUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		renderError(c, "User not found", http.StatusNotFound)
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
		renderError(c, "Failed to ban user", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *Handler) UnbanUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		renderError(c, "User not found", http.StatusNotFound)
		return
	}

	user.IsBanned = false
	user.BanReason = ""
	user.BannedAt = nil
	user.BannedUntil = nil

	if err := h.db.Save(&user).Error; err != nil {
		renderError(c, "Failed to unban user", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *Handler) ChangeUserType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		renderError(c, "User not found", http.StatusNotFound)
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
		renderError(c, "Failed to update user type", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

// Section management
func (h *Handler) SectionList(c *gin.Context) {
	user := h.getCurrentUser(c)

	var sections []models.Section
	if err := h.db.Order("\"order\" ASC").Find(&sections).Error; err != nil {
		renderError(c, "Failed to load sections", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"title":    "Section Management",
		"sections": sections,
		"user":     user,
		"config":   h.config,
	}
	renderTemplate(c, data, C.SectionListPath)
}

func (h *Handler) CreateSection(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	order, _ := strconv.Atoi(c.PostForm("order"))

	if name == "" {
		renderError(c, "Section name is required", http.StatusBadRequest)
		return
	}

	section := &models.Section{
		Name:        name,
		Description: description,
		Order:       order,
	}

	if err := h.db.Create(section).Error; err != nil {
		renderError(c, "Failed to create section", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/sections")
}

func (h *Handler) DeleteSection(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid section ID", http.StatusBadRequest)
		return
	}

	if err := h.db.Delete(&models.Section{}, id).Error; err != nil {
		renderError(c, "Failed to delete section", http.StatusInternalServerError)
		return
	}

	C.Cache.InvalidateCountsForUser(h.db, uint(id))

	c.Redirect(http.StatusFound, "/admin/sections")
}

// Category management
func (h *Handler) CategoryList(c *gin.Context) {
	user := h.getCurrentUser(c)

	// Pagination
	pageSize := 20
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	var totalCategories int64
	h.db.Model(&models.Category{}).Count(&totalCategories)
	totalPages := int((totalCategories + int64(pageSize) - 1) / int64(pageSize))

	var categories []models.Category
	if err := h.db.Preload("Section").
		Order("section_id ASC, \"order\" ASC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&categories).Error; err != nil {
		renderError(c, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	var sections []models.Section
	if err := h.db.Order("\"order\" ASC").Find(&sections).Error; err != nil {
		renderError(c, "Failed to load sections", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"title":      "Category Management",
		"categories": categories,
		"sections":   sections,
		"user":       user,
		"page":       page,
		"totalPages": totalPages,
		"config":     h.config,
	}
	renderTemplate(c, data, C.CategoryListPath)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	sectionID, _ := strconv.Atoi(c.PostForm("section_id"))
	order, _ := strconv.Atoi(c.PostForm("order"))

	if name == "" || sectionID == 0 {
		renderError(c, "Category name and section are required", http.StatusBadRequest)
		return
	}

	category := &models.Category{
		SectionID:   uint(sectionID),
		Name:        name,
		Description: description,
		Order:       order,
	}

	if err := h.db.Create(category).Error; err != nil {
		renderError(c, "Failed to create category", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/categories")
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid category ID", http.StatusBadRequest)
		return
	}

	if err := h.db.Delete(&models.Category{}, id).Error; err != nil {
		renderError(c, "Failed to delete category", http.StatusInternalServerError)
		return
	}

	// Invalidate relevant caches
	C.Cache.InvalidateTopicsInCategory(uint(id))
	C.Cache.InvalidateCountsForCategory(h.db, uint(id))

	c.Redirect(http.StatusFound, "/admin/categories")
}
