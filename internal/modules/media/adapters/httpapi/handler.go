package httpapi

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/apierr"
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/application/queries"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/domain"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

type handler struct {
	deps Deps
}

// canManage reports whether the caller holds the media.manage permission.
func (h *handler) canManage(ctx context.Context) bool {
	id := appauth.IdentityFrom(ctx)
	if id == nil {
		return false
	}
	return h.deps.Authorizer.Can(ctx, authorization.Identity{UserID: id.UserID, Roles: id.Roles}, authorization.PermissionManageMedia, nil) == nil
}

// owns reports whether the caller is the owner of the media (its model is the
// caller's user account) or holds media.manage. Anything else is treated as
// not-owned and hidden from reads.
func (h *handler) owns(ctx context.Context, m *domain.Media) bool {
	if h.canManage(ctx) {
		return true
	}
	id := appauth.IdentityFrom(ctx)
	if id == nil {
		return false
	}
	return m.ModelType == "user" && m.ModelID == id.UserID
}

// ownedScope returns the model scope the caller may read. Non-managers are
// restricted to their own user model, so listing never exposes other users'
// media.
func (h *handler) ownedScope(ctx context.Context) (modelType, modelID string, all bool) {
	if h.canManage(ctx) {
		return "", "", true
	}
	id := appauth.IdentityFrom(ctx)
	if id == nil {
		return "", "", false
	}
	return "user", id.UserID, false
}

// mediaURL returns a public URL for a media object. It prefers the storage
// driver's generated URL (e.g. S3 presigned); when unavailable (local disk,
// missing generator, or generator error) it falls back to the API download
// endpoint built from BaseURL.
func (h *handler) mediaURL(ctx context.Context, m *domain.Media) string {
	if h.deps.URLGenerator != nil {
		if u, err := h.deps.URLGenerator.URL(ctx, m.FileName); err == nil && u != "" {
			return u
		}
	}
	return strings.TrimSuffix(h.deps.BaseURL, "/") + mediaBasePath + "/" + m.ID + pathDownload
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
	platformhttp.WriteSuccess(w, http.StatusCreated, toMediaResponse(res, h.mediaURL(r.Context(), res)))
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	modelType, modelID, all := h.ownedScope(r.Context())
	q := queries.ListByModelQuery{
		ModelType:  r.URL.Query().Get(queryModelType),
		ModelID:    r.URL.Query().Get(queryModelID),
		Collection: r.URL.Query().Get(queryCollection),
	}
	if !all {
		q.ModelType = modelType
		q.ModelID = modelID
	}
	items, err := h.deps.ListByModel.Execute(r.Context(), q)
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	ctx := r.Context()
	platformhttp.WriteSuccess(w, http.StatusOK, toMediaResponses(items, func(m *domain.Media) string {
		return h.mediaURL(ctx, m)
	}))
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.deps.GetMedia.Execute(r.Context(), id)
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	defer item.Reader.Close()
	if !h.owns(r.Context(), item.Media) {
		platformhttp.WriteMappedError(w, domain.ErrNotFound)
		return
	}
	platformhttp.WriteSuccess(w, http.StatusOK, toMediaResponse(item.Media, h.mediaURL(r.Context(), item.Media)))
}

func (h *handler) download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.deps.GetMedia.Execute(r.Context(), id)
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	defer item.Reader.Close()
	if !h.owns(r.Context(), item.Media) {
		platformhttp.WriteMappedError(w, domain.ErrNotFound)
		return
	}

	w.Header().Set(headerContentType, item.Media.MimeType)
	w.Header().Set(headerContentLength, strconv.FormatInt(item.Media.Size, 10))
	w.Header().Set(headerContentDisposition, mime.FormatMediaType("attachment", map[string]string{"filename": mimeHeaderSanitize(item.Media.Name)}))
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

// mimeHeaderSanitize strips control characters (notably CR/LF) from a value
// that will be embedded in a MIME header, preventing header injection.
func mimeHeaderSanitize(v string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '\x00' {
			return -1
		}
		return r
	}, v)
}
