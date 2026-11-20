package handler

import (
	"fmt"
	"net/http"
	"time"

	"gitlab.arorms.cn/arorms/applications/blog/internal/service"
	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
	"gitlab.arorms.cn/arorms/applications/blog/internal/viewmodel"
)

// GalleryHandler serves the picture gallery.
type GalleryHandler struct {
	pictures service.IPictureService
}

func NewGalleryHandler(pictures service.IPictureService) *GalleryHandler {
	return &GalleryHandler{pictures: pictures}
}

// Gallery renders one page of pictures.
//
// The gallery has no filters, so it reads only the page state it can honour;
// reusing the article list's parameter set would imply a search and category
// filter this page does not have.
func (h *GalleryHandler) Gallery(w http.ResponseWriter, r *http.Request) {
	serve(w, r, func() (view.Page, error) {
		page := parsePage(r.URL.Query().Get("page"))
		limit := parseLimit(r.URL.Query().Get("limit"))

		result, err := h.pictures.GetGallery(r.Context(), limit, (page-1)*limit)
		if err != nil {
			return view.Page{}, err
		}
		if page > 1 && (result.Total == 0 || page > result.TotalPages) {
			return view.Page{}, view.NotFound(nil)
		}

		base := viewmodel.SectionPath("gallery")
		query := viewmodel.ListQuery{Page: page, Limit: limit}

		pictures := make([]viewmodel.PictureViewModel, 0, len(result.Content))
		for _, item := range result.Content {
			createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
			if err != nil {
				return view.Page{}, fmt.Errorf("parse picture %d timestamp: %w", item.ID, err)
			}
			iso, display := viewmodel.FormatDate(createdAt, zone)
			frame := viewmodel.PictureViewModel{
				ID:        item.ID,
				Alt:       item.Alt,
				URL:       item.URL,
				Thumbnail: item.ThumbnailURL,
				DateISO:   iso,
				DateShow:  display,
			}
			for _, tag := range item.Tags {
				frame.Tags = append(frame.Tags, viewmodel.Link{
					Label: tag.Name,
					Href:  viewmodel.SectionPath("articles") + "?tag=" + tag.Slug,
				})
			}
			pictures = append(pictures, frame)
		}

		return view.Page{
			View: view.GalleryView,
			Data: viewmodel.GalleryViewModel{
				Meta:       meta(r.URL.Path, "画廊", "写作之外的影像记录。"),
				Pictures:   pictures,
				Total:      result.Total,
				Pagination: viewmodel.NewPagination(base, query, page, result.TotalPages, result.Total),
				Empty:      len(pictures) == 0,
			},
		}, nil
	})
}
