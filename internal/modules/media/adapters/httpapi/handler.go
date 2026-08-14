package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/fatkulnurk/go-project-starter/internal/application/apierr"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/application/queries"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/domain"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

type handler struct {
	deps Deps
}

func (h *handler) upload(w http.ResponseWriter, r *http.Request) {
	if h.deps.MaxUploadSize > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, h.deps.MaxUploadSize)
	}
	if err := r.ParseMultipartForm(maxUploadMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			platformhttp.WriteMappedError(w, apierr.ErrPayloadTooLarge)
			return
		}
		platformhttp.WriteMappedError(w, domain.ErrInvalid)
		return
	}
	file, header, err := r.FormFile(formFieldFile)
	if err != nil {
		platformhttp.WriteMappedError(w, domain.ErrInvalid)
		return
	}
	defer file.Close()

	res, err := h.deps.AddMedia.Execute(r.Context(), commands.AddMediaCommand{
		ModelType:  r.FormValue(formFieldModelType),
		ModelID:    r.FormValue(formFieldModelID),
		Collection: r.FormValue(formFieldCollection),
		Name:       header.Filename,
		MimeType:   header.Header.Get(headerContentType),
		Size:       header.Size,
		Reader:     file,
	})
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccess(w, http.StatusCreated, toMediaResponse(res))
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.deps.ListByModel.Execute(r.Context(), queries.ListByModelQuery{
		ModelType:  r.URL.Query().Get(queryModelType),
		ModelID:    r.URL.Query().Get(queryModelID),
		Collection: r.URL.Query().Get(queryCollection),
	})
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccess(w, http.StatusOK, toMediaResponses(items))
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.deps.GetMedia.Execute(r.Context(), id)
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	item.Reader.Close()
	platformhttp.WriteSuccess(w, http.StatusOK, toMediaResponse(item.Media))
}

func (h *handler) download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.deps.GetMedia.Execute(r.Context(), id)
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	defer item.Reader.Close()

	w.Header().Set(headerContentType, item.Media.MimeType)
	w.Header().Set(headerContentLength, strconv.FormatInt(item.Media.Size, 10))
	w.Header().Set(headerContentDisposition, "attachment; filename=\""+item.Media.Name+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, item.Reader)
}

func (h *handler) remove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.deps.RemoveMedia.Execute(r.Context(), commands.RemoveMediaCommand{ID: id}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusOK, "media removed")
}
