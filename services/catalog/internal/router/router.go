package router

import (
	"net/http"

	"catalog/internal/handler"
	"catalog/internal/identity"
	"catalog/internal/middleware"
	"catalog/internal/response"

	"github.com/gin-gonic/gin"
)

func New(identityClient *identity.Client, brands *handler.BrandHandler, imageStore handler.ImageStore) (*gin.Engine, error) {
	engine := gin.New()
	if err := engine.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	engine.Use(middleware.RequestHeaders, middleware.Recovery())

	defaultBodyLimit := middleware.LimitBody(16 * 1024)
	engine.GET("/health", defaultBodyLimit, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	engine.NoRoute(defaultBodyLimit, func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "route not found")
	})

	api := engine.Group("/api")
	v1 := api.Group("/v1")
	catalogGroup := v1.Group("/catalog")

	admin := catalogGroup.Group("/admin", middleware.RequireAdmin(identityClient))

	brandRoutes := admin.Group("/brands", defaultBodyLimit)
	brandRoutes.GET("", brands.List)
	brandRoutes.POST("", brands.Create)
	brandRoutes.PATCH("/:id", brands.Update)

	admin.POST("/images", middleware.LimitBody(6*1024*1024), handler.UploadImage(imageStore))

	return engine, nil
}
