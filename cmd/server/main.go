package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"gitlab.arorms.cn/arorms/applications/blog/internal/config"
	"gitlab.arorms.cn/arorms/applications/blog/internal/handler"
	"gitlab.arorms.cn/arorms/applications/blog/internal/handler/api"
	"gitlab.arorms.cn/arorms/applications/blog/internal/repository"
	"gitlab.arorms.cn/arorms/applications/blog/internal/route"
	"gitlab.arorms.cn/arorms/applications/blog/internal/service"
	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
)

func main() {
	log.SetOutput(os.Stdout)

	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Name,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	renderer, err := view.New()
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	articleRepository := repository.NewArticleRepository(db)
	categoryRepository := repository.NewCategoryRepository(db)
	tagRepository := repository.NewTagRepository(db)
	pictureRepository := repository.NewPictureRepository(db)

	articleService := service.NewArticleService(articleRepository)
	categoryService := service.NewCategoryService(categoryRepository)
	tagService := service.NewTagService(tagRepository)
	pictureService := service.NewPictureService(pictureRepository)

	handler.Init(renderer, timezone(cfg.Server.Timezone))

	r := route.New(route.Handlers{
		MVC: route.MVC{
			Home:     handler.NewHomeHandler(articleService),
			Articles: handler.NewArticlesHandler(articleService, categoryService, tagService),
			Gallery:  handler.NewGalleryHandler(pictureService),
			About:    handler.NewAboutHandler(),
		},
		API: route.API{Article: api.NewArticleAPIHandler(articleService)},
	}, cfg.CORS)

	log.Printf("Server is running on: http://localhost:%d", cfg.Server.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// timezone resolves the display timezone. Dates are rendered in it regardless of
// the host's clock, so a deploy cannot silently shift every timestamp.
func timezone(name string) *time.Location {
	if name == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		log.Printf("Unknown timezone %q, falling back to UTC: %v", name, err)
		return time.UTC
	}
	return loc
}
