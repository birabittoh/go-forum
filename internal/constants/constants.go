package constants

import (
	"goforum/internal/models"
	"html/template"
	"os"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	Base      = "base"
	ps        = string(os.PathSeparator)
	templates = "templates" + ps

	BasePath = "templates" + ps + Base + ".html"

	AdminPanelPath          = templates + "admin_panel.html"
	CategoryListPath        = templates + "category_list.html"
	CategoryPath            = templates + "category.html"
	EditPostPath            = templates + "edit_post.html"
	EditTopicPath           = templates + "edit_topic.html"
	EditUserPath            = templates + "edit_user.html"
	ErrorPath               = templates + "error.html"
	HomePath                = templates + "home.html"
	LoginPath               = templates + "login.html"
	NewPostPath             = templates + "new_post.html"
	NewTopicPath            = templates + "new_topic.html"
	ProfileEditPath         = templates + "profile_edit.html"
	ProfilePath             = templates + "profile.html"
	SectionListPath         = templates + "section_list.html"
	SignupSuccessPath       = templates + "signup_success.html"
	SignupPath              = templates + "signup.html"
	TopicPath               = templates + "topic.html"
	UserListPath            = templates + "user_list.html"
	VerificationSuccessPath = templates + "verification_success.html"
)

var (
	TemplatePaths = []string{
		AdminPanelPath,
		CategoryListPath,
		CategoryPath,
		EditPostPath,
		EditTopicPath,
		EditUserPath,
		ErrorPath,
		HomePath,
		LoginPath,
		NewPostPath,
		NewTopicPath,
		ProfileEditPath,
		ProfilePath,
		SectionListPath,
		SignupSuccessPath,
		SignupPath,
		TopicPath,
		UserListPath,
		VerificationSuccessPath,
	}

	FuncMap = template.FuncMap{
		"sub": func(a, b int) int { return a - b },
		"default": func(value, def int) int {
			if value == 0 {
				return def
			}
			return value
		},
		"substr": func(s string, start, length int) string {
			r := []rune(s)
			if start < 0 || start >= len(r) || length <= 0 {
				return ""
			}
			end := min(start+length, len(r))
			return string(r[start:end])
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"title": func(s string) string {
			return cases.Title(language.Und).String(s)
		},
		"add":      func(a, b int) int { return a + b },
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },

		"validateTheme": func(theme string) string {
			for _, t := range Themes {
				if t.ID == theme {
					return theme
				}
			}
			return "default"
		},
		"getIconColor": func(theme string) string {
			for _, t := range Themes {
				if t.ID == theme {
					return t.Color
				}
			}
			return "white"
		},
		"until": func(count int) []int {
			var i []int
			for j := range count {
				i = append(i, j)
			}
			return i
		},
	}

	Tmpl = make(map[string]*template.Template)

	Themes        []models.Theme
	UsernameRegex = `^[a-zA-Z0-9][a-zA-Z0-9_.-]{3,19}$` // 4-20 chars, letters, numbers, _ and - .
)
