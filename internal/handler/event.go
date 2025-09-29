package handler

import (
	"io"
	"net/http"

	"github.com/cshum/vipsgen/vips"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/game3-back/internal/pkg/apperrors"
	"github.com/traPtitech/game3-back/internal/repository"
	"github.com/traPtitech/game3-back/openapi/models"
)

func (h *Handler) PostEvent(c echo.Context) (err error) {
	if _, _, err = h.enforceAdminAccess(c); err != nil {
		return err
	}

	req := &models.PostEventRequest{}
	if err = c.Bind(req); err != nil {
		return apperrors.HandleBindError(err)
	}

	file, err := extractFileFromForm(c, "image")
	if err != nil {
		return apperrors.HandleFileError(err)
	}
	var thumbnail []byte
	if file != nil {
		thumbnail, err = createEventThumbnail(file)
		if err != nil {
			return apperrors.HandleFileError(err)
		}
	}

	submittedGameEvent := repository.EventWithImage{
		Slug:                      req.Slug,
		Title:                     req.Title,
		Date:                      req.Date,
		GameSubmissionPeriodStart: req.GameSubmissionPeriodStart,
		GameSubmissionPeriodEnd:   req.GameSubmissionPeriodEnd,
		Image:                     thumbnail,
	}
	if err = h.repo.CreateEvent(submittedGameEvent); err != nil {
		return apperrors.HandleDbError(err)
	}

	event, err := h.repo.GetEvent(req.Slug)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	if err = h.repo.CreateDefaultTerm(event.Slug); err != nil {
		return apperrors.HandleDbError(err)
	}

	return c.JSON(http.StatusCreated, submittedGameEvent)
}

func (h *Handler) GetEvents(c echo.Context) error {
	events, err := h.repo.GetEvents()
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	return c.JSON(http.StatusOK, events)
}

func (h *Handler) GetCurrentEvent(c echo.Context) error {
	event, err := h.repo.GetCurrentEvent()
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	return c.JSON(http.StatusOK, event)
}

func (h *Handler) PatchEvent(c echo.Context, eventID models.EventSlugInPath) (err error) {
	if _, _, err = h.enforceAdminAccess(c); err != nil {
		return err
	}

	req := &models.PatchEventRequest{}
	if err = c.Bind(req); err != nil {
		return apperrors.HandleBindError(err)
	}

	image, err := extractFileFromForm(c, "image")
	if err != nil {
		return apperrors.HandleFileError(err)
	}
	var thumbnail []byte
	if image != nil {
		thumbnail, err = createEventThumbnail(image)
		if err != nil {
			return apperrors.HandleFileError(err)
		}
	}

	param := repository.PatchEventParam{
		Slug:                      req.Slug,
		Title:                     req.Title,
		Date:                      req.Date,
		GameSubmissionPeriodStart: req.GameSubmissionPeriodStart,
		GameSubmissionPeriodEnd:   req.GameSubmissionPeriodEnd,
	}
	if thumbnail != nil {
		param.Image = &thumbnail
	}
	if err = h.repo.PatchEvent(eventID, param); err != nil {
		return apperrors.HandleDbError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) GetEvent(c echo.Context, eventSlug models.EventSlugInPath) error {
	event, err := h.repo.GetEvent(eventSlug)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	return c.JSON(http.StatusOK, event)
}

func (h *Handler) GetEventImage(c echo.Context, eventID models.EventSlugInPath) error {
	image, err := h.repo.GetEventImage(eventID)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	err = validateCacheAndUpdateHeader(c, image.UpdatedAt.String())
	if err != nil {
		return err
	}

	contentType := http.DetectContentType(image.Image)

	return c.Blob(http.StatusOK, contentType, image.Image)
}

func (h *Handler) GetEventTerms(c echo.Context, eventID models.EventSlugInPath) error {
	events, err := h.repo.GetEventTerms(eventID)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	return c.JSON(http.StatusOK, events)
}

func (h *Handler) GetEventGames(c echo.Context, eventID models.EventSlugInPath) error {
	games, err := h.repo.GetGames(models.GetGamesParams{EventSlug: &eventID})
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	return c.JSON(http.StatusOK, games)
}

func (h *Handler) GetEventCsv(c echo.Context, _ models.EventSlugInPath) error {
	if _, _, err := h.enforceAdminAccess(c); err != nil {
		return err
	}

	return echo.NewHTTPError(http.StatusNotImplemented, "not implemented")
}

func createEventThumbnail(image io.ReadCloser) ([]byte, error) {
	src := vips.NewSource(image)
	defer src.Close()

	original, err := vips.NewImageFromSource(src, vips.DefaultLoadOptions())
	if err != nil {
		return nil, err
	}
	defer original.Close()

	const maxWidth = 600

	newWidth := min(original.Width(), maxWidth)
	newHeight := int(float64(original.Height()) * (float64(newWidth) / float64(original.Width())))

	thumbnail, err := vips.NewThumbnailSource(src, newWidth, &vips.ThumbnailSourceOptions{
		Height: newHeight,
		Intent: vips.IntentRelative,
	})
	if err != nil {
		return nil, err
	}
	defer thumbnail.Close()

	return thumbnail.WebpsaveBuffer(&vips.WebpsaveBufferOptions{
		NearLossless: true,
		Q:            80,
		AlphaQ:       100,
		Effort:       6,
		Keep:         vips.KeepNone,
	})
}
