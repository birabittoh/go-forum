package constants

import (
	"fmt"
	"goforum/internal/cache"
	"goforum/internal/models"
	"html/template"
	"log"
	"os"
	"path/filepath"
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
	BackupPath              = templates + "backup.html"
	CategoryPath            = templates + "category.html"
	ConfirmPath             = templates + "confirm.html"
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
	ResetPasswordPath       = templates + "reset_password.html"
	SectionsPath            = templates + "sections.html"
	SetNewPasswordPath      = templates + "set_new_password.html"
	SettingsPath            = templates + "settings.html"
	SignupSuccessPath       = templates + "signup_success.html"
	SignupPath              = templates + "signup.html"
	TopicPath               = templates + "topic.html"
	UserListPath            = templates + "user_list.html"
	VerificationSuccessPath = templates + "verification_success.html"

	FaviconTemplate = "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 100 100\"><text y=\".9em\" font-size=\"80\" fill=\"%s\">🗫</text></svg>"
)

var (
	TemplatePaths = []string{
		AdminPanelPath,
		BackupPath,
		CategoryPath,
		ConfirmPath,
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
		ResetPasswordPath,
		SectionsPath,
		SetNewPasswordPath,
		SettingsPath,
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
			return ValidateTheme(theme).ID
		},
		"getColor": func(theme string) string {
			return strings.TrimPrefix(ValidateTheme(theme).Color, "#")
		},
		"until": func(count int) []int {
			var i []int
			for j := range count {
				i = append(i, j)
			}
			return i
		},
	}

	Cache = cache.New()

	Tmpl = make(map[string]*template.Template)

	Themes        map[string]models.Theme
	UsernameRegex = `^[a-zA-Z0-9][a-zA-Z0-9_.-]{3,19}$` // 4-20 chars, letters, numbers, _ and - .

	jsonReplacer = strings.NewReplacer(
		`'`, `\'`,
		`"`, `\"`,
		`\`, `\\`,
		"\n", `\n`,
		"\r", `\r`,
	)

	Manifest = map[string]any{
		"name":             "",
		"short_name":       "",
		"description":      "",
		"start_url":        "/",
		"display":          "standalone",
		"background_color": "#ffffff",
		"theme_color":      "#1976d2",
		"icons": []map[string]string{
			{

				"src":   "/favicon.svg",
				"sizes": "any",
				"type":  "image/svg+xml",
			},
		},
	}
)

func SeedThemes() {
	cssFiles, err := filepath.Glob(filepath.Join("static", "themes", "*.css"))
	if err != nil {
		log.Fatal("Failed to read themes directory:", err)
	}

	Themes = make(map[string]models.Theme) // Initialize the map
	for _, file := range cssFiles {
		_, fileName := filepath.Split(file)
		themeID := fileName[:len(fileName)-4] // remove .css extension
		words := strings.Split(themeID, "-")
		for i, word := range words {
			if len(word) > 0 && (word != "and" && word != "or" && word != "the" && word != "of") {
				words[i] = strings.ToUpper(word[:1]) + word[1:]
			}
		}
		displayName := strings.Join(words, " ")

		// read first line of css file to find icon color comment
		var iconColor string
		f, err := os.Open(file)
		if err == nil {
			var line string
			_, err = fmt.Fscanf(f, "/*%s", &line)
			if err == nil {
				iconColor = strings.TrimSpace(strings.TrimSuffix(line, "*/"))
			}
			f.Close()
		}
		if iconColor == "" {
			iconColor = "white" // Default icon color
			log.Printf("Warning: No icon color specified for theme %s, defaulting to white", themeID)
		}

		Themes[themeID] = models.Theme{
			ID:          themeID,
			DisplayName: displayName,
			Color:       iconColor,
		}
	}
}

func ValidateTheme(theme string) models.Theme {
	t, ok := Themes[theme]
	if ok {
		return t
	}
	return Themes["default"]
}

func EscapeJSON(s string) string {
	return jsonReplacer.Replace(s)
}
