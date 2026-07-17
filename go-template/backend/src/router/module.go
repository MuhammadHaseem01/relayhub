package router

import (
	"database/sql"
	"strings"

	"github.com/gin-gonic/gin"
)

type Router struct {
	Engine *gin.Engine
	server *Server
}

func NewRouter(db *sql.DB, configs ...ServerConfig) *Router {
	engine := gin.New()
	engine.RemoveExtraSlash = true
	engine.RedirectTrailingSlash = false
	engine.Use(gin.Logger(), gin.Recovery())

	router := &Router{
		Engine: engine,
		server: NewServerCore(db, configs...),
	}
	router.defineRoutes()
	return router
}

func (r *Router) defineRoutes() {
	handler := r.server.withMiddleware(r.server.routes())
	wrapped := gin.WrapH(handler)
	r.Engine.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) > 1 && strings.HasSuffix(c.Request.URL.Path, "/") {
			c.Request.URL.Path = strings.TrimRight(c.Request.URL.Path, "/")
			handler.ServeHTTP(c.Writer, c.Request)
			return
		}
		c.Status(404)
	})

	r.Engine.GET("/api", wrapped)
	r.Engine.OPTIONS("/api", wrapped)
	r.Engine.OPTIONS("/api/*path", wrapped)

	api := r.Engine.Group("/api")
	{
		user := api.Group("/user")
		{
			user.POST("/login", wrapped)
			user.POST("/register", wrapped)
			user.POST("/add", wrapped)
			user.GET("", wrapped)
			user.GET("/:id", wrapped)
			user.PATCH("/:id", wrapped)
			user.DELETE("/:id", wrapped)
		}

		ledger := api.Group("/ledger")
		{
			ledger.GET("", wrapped)
			ledger.GET("/:id", wrapped)
		}

		registerOwnedResourceRoutes(api.Group("/client"), wrapped, true, true)
		registerOwnedResourceRoutes(api.Group("/arrange-vehicle"), wrapped, true, true)
		registerOwnedResourceRoutes(api.Group("/inventory"), wrapped, true, true)
		registerOwnedResourceRoutes(api.Group("/tour-damage"), wrapped, true, true)
		registerOwnedResourceRoutes(api.Group("/tour-deduction"), wrapped, true, true)

		vehicle := api.Group("/vehicle")
		{
			vehicle.POST("/add", wrapped)
			vehicle.GET("", wrapped)
			vehicle.GET("/:id", wrapped)
			vehicle.PATCH("/:id", wrapped)
			vehicle.DELETE("/:id", wrapped)
		}

		driver := api.Group("/driver")
		{
			driver.POST("/add", wrapped)
			driver.GET("", wrapped)
			driver.PATCH("/:id", wrapped)
			driver.DELETE("/:id", wrapped)
		}

		maintenance := api.Group("/vehicle-maintenance")
		{
			maintenance.POST("/add", wrapped)
			maintenance.GET("", wrapped)
			maintenance.GET("/:id", wrapped)
			maintenance.DELETE("/:id", wrapped)
		}
		api.PATCH("/vehicle-maintenance/*path", wrapped)

		tour := api.Group("/tour")
		{
			tour.POST("/add", wrapped)
			tour.GET("", wrapped)
			tour.GET("/:id", wrapped)
			tour.PATCH("/:id", wrapped)
			tour.DELETE("/:id", wrapped)
		}
	}
}

func registerOwnedResourceRoutes(group *gin.RouterGroup, handler gin.HandlerFunc, includeGetByID bool, includeDelete bool) {
	group.POST("/add", handler)
	group.GET("", handler)
	if includeGetByID {
		group.GET("/:id", handler)
	}
	group.PATCH("/:id", handler)
	if includeDelete {
		group.DELETE("/:id", handler)
	}
}
