package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/traPtitech/game3-back/internal/api"
	"github.com/traPtitech/game3-back/internal/pkg/apperrors"
	"github.com/traPtitech/game3-back/internal/pkg/constants"
	"github.com/traPtitech/game3-back/internal/pkg/enum"
	"github.com/traPtitech/game3-back/internal/pkg/util"
	"github.com/traPtitech/game3-back/openapi/models"
)

func (h *Handler) GetGames(c echo.Context, params models.GetGamesParams) error {
	user, role, err := h.getDiscordUserInfoAndRoleByCookie(c)
	if err != nil {
		return err
	}

	if params.IncludeUnpublished != nil && *params.IncludeUnpublished {
		if (params.UserId == nil || user == nil || *params.UserId != user.ID) && !role.IsAdmin() {
			return echo.NewHTTPError(http.StatusForbidden, "you can't get other user's unpublished game")
		}
	}

	games, err := h.repo.GetGames(params)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	return c.JSON(http.StatusOK, games)
}

func (h *Handler) PostGame(c echo.Context) error {
	user, _, err := h.enforceUserOrAboveAccess(c)
	if err != nil {
		return err
	}

	currentEvent, err := h.repo.GetCurrentEvent()
	if err != nil {
		return apperrors.HandleDbError(err)
	}
	defaultTerm, err := h.repo.GetDefaultTerm(currentEvent.Slug)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	req := &models.PostGameRequest{}
	err = c.Bind(req)
	if err != nil {
		return apperrors.HandleBindError(err)
	}

	icon, err := handleGameIcon(c)
	if err != nil {
		return apperrors.HandleFileError(err)
	}
	req.Icon = *icon
	req.Image, err = handleGameImage(c)
	if err != nil {
		return apperrors.HandleFileError(err)
	}

	newGameID := uuid.New()
	if err = h.repo.PostGame(newGameID, defaultTerm.Id, user.ID, req); err != nil {
		return apperrors.HandleDbError(err)
	}

	event, err := h.repo.GetCurrentEvent()
	if err != nil {
		return apperrors.HandleDbError(err)
	}
	term, err := h.repo.GetDefaultTerm(event.Slug)
	if err != nil {
		return apperrors.HandleDbError(err)
	}
	if err = h.repo.PatchGame(newGameID, &models.PatchGameRequest{TermId: &term.Id}); err != nil {
		return apperrors.HandleDbError(err)
	}

	game, err := h.repo.GetGame(newGameID)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	err = addExhibitorRoleToDiscordUser(user.ID)
	if err != nil {
		return err
	}

	err = createGamePostedMessageToDiscord(game)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, game)
}

func addExhibitorRoleToDiscordUser(discordUserID string) error {
	game3ServerID, err := util.GetEnvOrErr("DISCORD_SERVER_ID")
	if err != nil {
		return err
	}
	exhibitorRole, err := util.GetEnvOrErr("DISCORD_EXHIBITOR_PARTICIPANT_ROLE_ID")
	if err != nil {
		return err
	}

	return api.AddRoleToDiscordUser(game3ServerID, discordUserID, exhibitorRole)
}

func createGamePostedMessageToDiscord(game *models.Game) error {
	game3ChannelID, err := util.GetEnvOrErr("DISCORD_BOT_CHANNEL_ID")
	if err != nil {
		return err
	}

	return api.CreateDiscordMessage(game3ChannelID, "新しいゲームが投稿されました: https://game3.trap.games/entry/"+game.Id.String())
}

func (h *Handler) GetGame(c echo.Context, gameID models.GameIdInPath) error {
	game, err := h.repo.GetGame(gameID)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	return c.JSON(http.StatusOK, game)
}

func (h *Handler) PatchGame(c echo.Context, gameID models.GameIdInPath) error {
	user, role, err := h.enforceUserOrAboveAccess(c)
	if err != nil {
		return err
	}

	req := &models.PatchGameRequest{}
	err = c.Bind(req)
	if err != nil {
		return apperrors.HandleBindError(err)
	}

	req.Icon, err = handleGameIcon(c)
	if err != nil {
		return apperrors.HandleFileError(err)
	}
	req.Image, err = handleGameImage(c)
	if err != nil {
		return apperrors.HandleFileError(err)
	}

	if role == enum.User {
		game, err := h.repo.GetGame(gameID)
		if err != nil {
			return apperrors.HandleDbError(err)
		}
		if game.DiscordUserId != user.ID {
			return echo.NewHTTPError(http.StatusForbidden, "you can't update other user's game")
		}
		if req.DiscordUserId != nil || req.TermId != nil || req.IsPublished != nil || req.Place != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "you can't update discordUserId, termId, place")
		}
	}

	if err := h.repo.PatchGame(gameID, req); err != nil {
		return apperrors.HandleDbError(err)
	}

	game, err := h.repo.GetGame(gameID)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	return c.JSON(http.StatusOK, game)
}

func (h *Handler) GetGameIcon(c echo.Context, gameID models.GameIdInPath) error {
	icon, err := h.repo.GetGameIcon(gameID)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	err = validateCacheAndUpdateHeader(c, icon.UpdatedAt.String())
	if err != nil {
		return err
	}

	return c.Blob(http.StatusOK, "image/png", icon.Image)
}

func (h *Handler) GetGameImage(c echo.Context, gameID models.GameIdInPath) error {
	image, err := h.repo.GetGameImage(gameID)
	if err != nil {
		return apperrors.HandleDbError(err)
	}

	err = validateCacheAndUpdateHeader(c, image.UpdatedAt.String())
	if err != nil {
		return err
	}

	return c.Blob(http.StatusOK, "image/png", image.Image)
}

func handleGameImage(c echo.Context) (*types.File, error) {
	return handleImageFileAndConvertImageToPNGAndResizeImage(c, "image", constants.GameImageWidth, constants.GameImageHeight)
}

func handleGameIcon(c echo.Context) (*types.File, error) {
	return handleImageFileAndConvertImageToPNGAndResizeImage(c, "icon", constants.GameIconSize, constants.GameIconSize)
}
