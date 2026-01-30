package rest

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/LdDl/greenwave/app/configuration"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
)

// CustomHTTPErrorHandler handles HTTP errors by serving custom error pages.
func CustomHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	errorPage := fmt.Sprintf("%d.html", code)
	if err := c.File(errorPage); err != nil {
		log.Error().Err(err).Str("scope", "echo-erro-handler").Msg("Can't process request")
	}
}

// Setup initializes the Echo microservice with the provided configuration.
func Setup(appCfg *configuration.Configuration) *echo.Echo {
	microservice := echo.New()
	microservice.HTTPErrorHandler = CustomHTTPErrorHandler

	microservice.HideBanner = true
	microservice.HidePort = true

	if appCfg.UseCORS {
		allCors := middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Content-Length", "Accept", "Accept-Encoding", "X-HttpRequest"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: false,
			MaxAge:           5600,
		})
		microservice.Use(allCors)
	}
	MainAPI(microservice, appCfg)
	return microservice
}

// MainAPI sets up the main API routes for the Echo microservice.
func MainAPI(app *echo.Echo, appCfg *configuration.Configuration) {
	mainGroup := app.Group(fmt.Sprintf("/%s", appCfg.ServerCfg.MainPath))
	routerGroup := mainGroup.Group("/greenwave")
	{
		routerGroup.Static("/docs", appCfg.DocsFolder)
		routerGroup.GET("/health", GetHealth())
		routerGroup.POST("/extract", ExtractGreenWaves())
		routerGroup.POST("/optimize", RequestOptimize())
	}

	// Serve static files from the UI build folder if configured
	if appCfg.StaticFolder != "" {
		// Serve static assets
		app.Static("/", appCfg.StaticFolder)

		// SPA fallback: serve index.html for all non-API, non-file routes
		app.GET("/*", func(c echo.Context) error {
			requestPath := c.Request().URL.Path

			// Check if the requested file exists
			filePath := filepath.Join(appCfg.StaticFolder, requestPath)
			if _, err := os.Stat(filePath); err == nil {
				return c.File(filePath)
			}

			// Fallback to index.html for SPA routing
			return c.File(filepath.Join(appCfg.StaticFolder, "index.html"))
		})

		log.Info().Str("scope", "api").Str("folder", appCfg.StaticFolder).Msg("Serving static UI files")
	}
}
