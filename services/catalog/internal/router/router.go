package router

import (
	"net/http"

	"catalog/internal/handler"
	"catalog/internal/identity"
	"catalog/internal/middleware"
	"catalog/internal/response"

	"github.com/gin-gonic/gin"
)

func New(identityClient *identity.Client, brands *handler.BrandHandler, banners *handler.BannerHandler, cars *handler.CarHandler, imageStore handler.ImageStore) (*gin.Engine, error) {
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

	home := catalogGroup.Group("/home", defaultBodyLimit)
	home.GET("/banners", banners.ListPublic)

	publicCars := catalogGroup.Group("/cars", defaultBodyLimit)
	publicCars.GET("", cars.ListPublic)
	publicCars.GET("/:slug", cars.DetailPublic)
	publicCars.GET("/:slug/related", cars.ListRelated)

	catalogGroup.GET("/brands", defaultBodyLimit, cars.ListPublicBrands)

	admin := catalogGroup.Group("/admin", middleware.RequireAdmin(identityClient))

	brandRoutes := admin.Group("/brands", defaultBodyLimit)
	brandRoutes.GET("", brands.List)
	brandRoutes.POST("", brands.Create)
	brandRoutes.PATCH("/:id", brands.Update)

	bannerRoutes := admin.Group("/banners", defaultBodyLimit)
	bannerRoutes.GET("", banners.List)
	bannerRoutes.GET("/:position", banners.Detail)
	bannerRoutes.PUT("/:position", banners.Update)
	bannerRoutes.DELETE("/:position", banners.Delete)

	carRoutes := admin.Group("/cars", defaultBodyLimit)
	carRoutes.GET("", cars.List)
	carRoutes.POST("", cars.Create)
	carRoutes.GET("/:id", cars.Detail)
	carRoutes.PATCH("/:id", cars.Update)

	admin.POST("/images", middleware.LimitBody(6*1024*1024), handler.UploadImage(imageStore))

	return engine, nil
}
