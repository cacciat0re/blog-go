package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gitlab.arorms.cn/arorms/applications/blog/internal/model"
	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/service"
)

type ArticleAPIHandler struct {
	articleService service.IArticleService
}

func NewArticleAPIHandler(articleService service.IArticleService) *ArticleAPIHandler {
	return &ArticleAPIHandler{
		articleService: articleService,
	}
}

func (h *ArticleAPIHandler) Create(w http.ResponseWriter, r *http.Request) {
	var article model.Article
	if err := json.NewDecoder(r.Body).Decode(&article); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.articleService.Create(r.Context(), &article); err != nil {
		log.Printf("failed to create article: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(article); err != nil {
		log.Printf("failed to encode created article %d: %v", article.ID, err)
	}
}

func (h *ArticleAPIHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}

	article, err := h.articleService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			http.Error(w, "article not found", http.StatusNotFound)
			return
		}
		log.Printf("failed to get article %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(article); err != nil {
		log.Printf("failed to encode article %d: %v", id, err)
	}
}

// GetAll lists published articles. It is a read, so its filters come from the
// query string rather than a JSON body.
func (h *ArticleAPIHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	query, err := parseArticleListRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.articleService.GetAllPublished(r.Context(), query)
	if err != nil {
		log.Printf("failed to list published articles: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("failed to encode article list: %v", err)
	}
}

// parseArticleListRequest reads the list filters from the query string. Missing
// numeric parameters stay zero, which the service normalizes.
func parseArticleListRequest(r *http.Request) (payload.ArticleListRequest, error) {
	values := r.URL.Query()
	query := payload.ArticleListRequest{
		Key:      values.Get("key"),
		Category: values.Get("category"),
		Tag:      values.Get("tag"),
	}

	limit, err := queryInt(values.Get("limit"))
	if err != nil {
		return query, fmt.Errorf("invalid limit %q", values.Get("limit"))
	}
	offset, err := queryInt(values.Get("offset"))
	if err != nil {
		return query, fmt.Errorf("invalid offset %q", values.Get("offset"))
	}
	categoryID, err := queryInt(values.Get("category_id"))
	if err != nil {
		return query, fmt.Errorf("invalid category_id %q", values.Get("category_id"))
	}

	query.Limit = int(limit)
	query.Offset = int(offset)
	if categoryID > 0 {
		query.CategoryID = &categoryID
	}
	return query, nil
}

// queryInt parses an optional integer query value; an absent value is zero.
func queryInt(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}
