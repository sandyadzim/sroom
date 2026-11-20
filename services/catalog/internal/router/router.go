package router

import (
	"net/http"

	"catalog/internal/handler"
	"catalog/internal/identity"
	"catalog/internal/middleware"
	"catalog/internal/response"

	"github.com/gin-gonic/gin"
)

func New(identityClient *identity.Client, brands *handler.BrandHandler, banners *handler.BannerHandler, cars *handler.CarHandler, featuredCars *handler.FeaturedCarHandler, units *handler.CarUnitHandler, imageStore handler.ImageStore, bookingToken string) (*gin.Engine, error) {
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

	if bookingToken != "" {
		internal := engine.Group("/internal")
		internalV1 := internal.Group("/v1", defaultBodyLimit, middleware.RequireBookingToken(bookingToken))
		internalV1.GET("/variants/:id/booking-snapshot", cars.BookingSnapshot)
	}

	api := engine.Group("/api")
	v1 := api.Group("/v1")
	catalogGroup := v1.Group("/catalog")

	home := catalogGroup.Group("/home", defaultBodyLimit)
	home.GET("/banners", banners.ListPublic)
	home.GET("/featured-cars", featuredCars.ListPublic)

	publicCars := catalogGroup.Group("/cars", defaultBodyLimit)
	publicCars.GET("", cars.ListPublic)
	publicCars.GET("/:slug", cars.DetailPublic)
	publicCars.GET("/:slug/related", cars.ListRelated)

	catalogGroup.GET("/brands", defaultBodyLimit, cars.ListPublicBrands)

	admin := catalogGroup.Group("/admin", middleware.RequireAdmin(identityClient))
	admin.GET("/fleet-summary", defaultBodyLimit, cars.FleetSummary)

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
	carRoutes.GET("/:id/units", units.List)
	carRoutes.POST("/:id/units", units.Create)

	unitRoutes := admin.Group("/units", defaultBodyLimit)
	unitRoutes.GET("/:id", units.Detail)
	unitRoutes.PATCH("/:id", units.Update)

	featuredRoutes := admin.Group("/featured-cars", defaultBodyLimit)
	featuredRoutes.GET("", featuredCars.List)
	featuredRoutes.PUT("", featuredCars.Replace)

	admin.POST("/images", middleware.LimitBody(6*1024*1024), handler.UploadImage(imageStore))

	return engine, nil
}
