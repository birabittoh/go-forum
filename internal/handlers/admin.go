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
	"gorm.io/gorm"
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

func (h *Handler) AdminSettingsForm(c *gin.Context) {
	user := h.getCurrentUser(c)
	if !user.IsAdmin() {
		renderError(c, "Access denied", http.StatusForbidden)
		return
	}
	var settings models.Settings
	if err := h.db.First(&settings, 1).Error; err != nil {
		renderError(c, "Failed to load settings", http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"title":    "Site Settings",
		"user":     user,
		"settings": settings,
		"config":   h.config,
	}
	renderTemplate(c, data, C.SettingsPath)
}

func (h *Handler) AdminSettingsUpdate(c *gin.Context) {
	user := h.getCurrentUser(c)
	if !user.IsAdmin() {
		renderError(c, "Access denied", http.StatusForbidden)
		return
	}
	var settings models.Settings
	if err := h.db.First(&settings, 1).Error; err != nil {
		renderError(c, "Failed to load settings", http.StatusInternalServerError)
		return
	}

	settings.SiteURL = c.PostForm("SiteURL")
	settings.SiteName = c.PostForm("SiteName")
	settings.SiteMotto = c.PostForm("SiteMotto")
	settings.MaxPostLength, _ = strconv.Atoi(c.PostForm("MaxPostLength"))
	settings.MaxMottoLength, _ = strconv.Atoi(c.PostForm("MaxMottoLength"))
	settings.MaxSignatureLength, _ = strconv.Atoi(c.PostForm("MaxSignatureLength"))
	settings.TopicPageSize, _ = strconv.Atoi(c.PostForm("TopicPageSize"))

	if err := h.db.Save(&settings).Error; err != nil {
		data := map[string]any{
			"title":    "Site Settings",
			"user":     user,
			"settings": settings,
			"error":    "Failed to update settings",
			"config":   h.config,
		}
		renderTemplate(c, data, C.SettingsPath)
		return
	}

	h.config.LoadSettings(&settings)

	c.Redirect(http.StatusFound, "/admin/settings")
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
		"timezones":  C.TimezonesList(),
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
	targetUser.Signature = c.PostForm("signature")
	targetUser.Timezone = c.PostForm("timezone")

	if err := C.Cache.UpdateUser(&targetUser); err != nil {
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

	user, ok := C.Cache.GetUserByID(uint(id))
	if !ok {
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

	if err := C.Cache.UpdateUser(&user); err != nil {
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

	user, ok := C.Cache.GetUserByID(uint(id))
	if !ok {
		renderError(c, "User not found", http.StatusNotFound)
		return
	}

	user.IsBanned = false
	user.BanReason = ""
	user.BannedAt = nil
	user.BannedUntil = nil

	if err := C.Cache.UpdateUser(&user); err != nil {
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

	user, ok := C.Cache.GetUserByID(uint(id))
	if !ok {
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

	if err := C.Cache.UpdateUser(&user); err != nil {
		renderError(c, "Failed to update user type", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *Handler) ComputeAI(c *gin.Context) {
	if !h.config.AIEnabled {
		renderError(c, "AI detection is not enabled", http.StatusBadRequest)
		return
	}

	// get all posts that have not been processed yet (AIProbability is nil)
	var posts []models.Post
	if err := h.db.Where("ai_probability IS NULL").Find(&posts).Error; err != nil {
		renderError(c, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	for _, post := range posts {
		if err := h.aiService.EnqueueDetection(&post); err != nil {
			renderError(c, "Failed to enqueue post ID "+strconv.Itoa(int(post.ID))+": "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	c.Redirect(http.StatusFound, "/admin")
}

// Unified Sections & Categories management
func (h *Handler) AdminSections(c *gin.Context) {
	user := h.getCurrentUser(c)

	var sections []models.Section
	if err := h.db.Preload("Categories", func(db *gorm.DB) *gorm.DB {
		return db.Order("\"order\" ASC")
	}).Order("\"order\" ASC").Find(&sections).Error; err != nil {
		renderError(c, "Failed to load sections", http.StatusInternalServerError)
		return
	}

	var categories []models.Category
	if err := h.db.Preload("Section").Order("section_id ASC, \"order\" ASC").Find(&categories).Error; err != nil {
		renderError(c, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"title":      "Sections & Categories Management",
		"sections":   sections,
		"categories": categories,
		"user":       user,
		"config":     h.config,
	}
	renderTemplate(c, data, "templates/sections.html")
}

func (h *Handler) CreateSection(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")

	if name == "" {
		renderError(c, "Section name is required", http.StatusBadRequest)
		return
	}

	// Find max order value among sections
	var maxOrder int
	err := h.db.Model(&models.Section{}).Select("COALESCE(MAX(\"order\"), 0)").Scan(&maxOrder).Error
	if err != nil {
		renderError(c, "Failed to create section", http.StatusInternalServerError)
		return
	}

	section := &models.Section{
		Name:        name,
		Description: description,
		Order:       maxOrder + 1,
	}

	if err := h.db.Create(section).Error; err != nil {
		renderError(c, "Failed to create section", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/sections")
}

func (h *Handler) UpdateSection(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid section ID", http.StatusBadRequest)
		return
	}
	name := c.PostForm("name")
	description := c.PostForm("description")
	if name == "" {
		renderError(c, "Section name is required", http.StatusBadRequest)
		return
	}
	var section models.Section
	if err := h.db.First(&section, id).Error; err != nil {
		renderError(c, "Section not found", http.StatusNotFound)
		return
	}
	section.Name = name
	section.Description = description
	if err := h.db.Save(&section).Error; err != nil {
		renderError(c, "Failed to update section", http.StatusInternalServerError)
		return
	}
	c.Redirect(http.StatusFound, "/admin/sections")
}

// Move section to a specific order (handles soft deletion)
func (h *Handler) MoveSection(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid section ID", http.StatusBadRequest)
		return
	}
	newOrder, err := strconv.Atoi(c.Param("order"))
	if err != nil {
		renderError(c, "Invalid order value", http.StatusBadRequest)
		return
	}
	var section models.Section
	if err := h.db.First(&section, id).Error; err != nil {
		renderError(c, "Section not found", http.StatusNotFound)
		return
	}

	tx := h.db.Begin()

	var target models.Section
	err = h.db.Where("\"order\" = ?", newOrder).First(&target).Error
	if err == nil {
		// Swap order values
		section.Order, target.Order = target.Order, section.Order
		err = h.db.Save(&target).Error
		if err != nil {
			tx.Rollback()
			renderError(c, "Failed to move section", http.StatusInternalServerError)
			return
		}
	} else {
		section.Order = newOrder
	}

	err = tx.Save(&section).Error
	if err != nil {
		tx.Rollback()
		renderError(c, "Failed to move section", http.StatusInternalServerError)
		return
	}

	tx.Commit()
	c.Redirect(http.StatusFound, "/admin/sections")
}

func (h *Handler) DeleteSection(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid section ID", http.StatusBadRequest)
		return
	}

	section := models.Section{}
	if err := h.db.Where("deleted_at IS NULL").First(&section, id).Error; err != nil {
		renderError(c, "Section not found", http.StatusNotFound)
		return
	}

	tx := h.db.Begin()

	section.Order = 0
	err = tx.Save(&section).Error
	if err != nil {
		tx.Rollback()
		renderError(c, "Failed to delete section", http.StatusInternalServerError)
		return
	}

	if err := tx.Delete(&models.Section{}, id).Error; err != nil {
		tx.Rollback()
		renderError(c, "Failed to delete section", http.StatusInternalServerError)
		return
	}

	tx.Commit()

	c.Redirect(http.StatusFound, "/admin/sections")
}

// Category management

func (h *Handler) CreateCategory(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	sectionID, _ := strconv.Atoi(c.PostForm("section_id"))

	if name == "" || sectionID == 0 {
		renderError(c, "Category name and section are required", http.StatusBadRequest)
		return
	}

	// Find max order value among categories in the selected section
	var maxOrder int
	err := h.db.Model(&models.Category{}).Where("section_id = ?", sectionID).Select("COALESCE(MAX(\"order\"), 0)").Scan(&maxOrder).Error
	if err != nil {
		renderError(c, "Failed to create category", http.StatusInternalServerError)
		return
	}

	category := &models.Category{
		SectionID:   uint(sectionID),
		Name:        name,
		Description: description,
		Order:       maxOrder + 1,
	}

	if err := h.db.Create(category).Error; err != nil {
		renderError(c, "Failed to create category", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/admin/sections")
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid category ID", http.StatusBadRequest)
		return
	}
	name := c.PostForm("name")
	description := c.PostForm("description")
	if name == "" {
		renderError(c, "Category name is required", http.StatusBadRequest)
		return
	}
	var category models.Category
	if err := h.db.First(&category, id).Error; err != nil {
		renderError(c, "Category not found", http.StatusNotFound)
		return
	}
	category.Name = name
	category.Description = description
	if err := h.db.Save(&category).Error; err != nil {
		renderError(c, "Failed to update category", http.StatusInternalServerError)
		return
	}
	c.Redirect(http.StatusFound, "/admin/sections")
}

// Move category to a specific order (handles soft deletion)
func (h *Handler) MoveCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid category ID", http.StatusBadRequest)
		return
	}
	newOrder, err := strconv.Atoi(c.Param("order"))
	if err != nil {
		renderError(c, "Invalid order value", http.StatusBadRequest)
		return
	}
	var category models.Category
	if err := h.db.Where("deleted_at IS NULL").First(&category, id).Error; err != nil {
		renderError(c, "Category not found", http.StatusNotFound)
		return
	}

	tx := h.db.Begin()
	// Find target category at newOrder in same section (not deleted)
	var target models.Category
	if err := h.db.Where("section_id = ? AND \"order\" = ? AND deleted_at IS NULL", category.SectionID, newOrder).First(&target).Error; err == nil {
		category.Order, target.Order = target.Order, category.Order
		err := tx.Save(&target).Error
		if err != nil {
			tx.Rollback()
			renderError(c, "Failed to move category", http.StatusInternalServerError)
			return
		}
	} else {
		category.Order = newOrder
	}

	err = tx.Save(&category).Error
	if err != nil {
		tx.Rollback()
		renderError(c, "Failed to move category", http.StatusInternalServerError)
		return
	}

	tx.Commit()
	c.Redirect(http.StatusFound, "/admin/sections")
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var category models.Category
	if err := h.db.First(&category, id).Error; err != nil {
		renderError(c, "Category not found", http.StatusNotFound)
		return
	}

	tx := h.db.Begin()

	category.Order = 0
	if err := tx.Save(&category).Error; err != nil {
		tx.Rollback()
		renderError(c, "Failed to delete category", http.StatusInternalServerError)
		return
	}

	if err := tx.Delete(&models.Category{}, id).Error; err != nil {
		renderError(c, "Failed to delete category", http.StatusInternalServerError)
		return
	}

	tx.Commit()

	// Invalidate relevant caches
	C.Cache.InvalidateTopicsInCategory(uint(id))

	c.Redirect(http.StatusFound, "/admin/sections")
}
