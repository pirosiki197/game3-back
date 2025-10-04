package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/traPtitech/game3-back/internal/pkg/apperrors"
	"github.com/traPtitech/game3-back/internal/pkg/enum"
	"github.com/traPtitech/game3-back/internal/pkg/util"
	"github.com/traPtitech/game3-back/openapi/models"
)

type GetDiscordUserInfoResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

type GetDiscordUserGuildsResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Icon           *string  `json:"icon"`
	Owner          bool     `json:"owner"`
	Permissions    int      `json:"permissions"`
	PermissionsNew string   `json:"permissions_new"`
	Features       []string `json:"features"`
}

type TokenResponse struct {
	TokenType    *string `json:"token_type"`
	AccessToken  *string `json:"access_token"`
	ExpiresIn    *int    `json:"expires_in"`
	RefreshToken *string `json:"refresh_token"`
	Scope        *string `json:"scope"`
}

func GetDiscordUserInfo(accessToken *string) (*GetDiscordUserInfoResponse, error) {
	req, err := http.NewRequest("GET", "https://discordapp.com/api/users/@me", nil)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+*accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get Discord user: status "+resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var discordUser GetDiscordUserInfoResponse
	if err = json.Unmarshal(body, &discordUser); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return &discordUser, nil
}

func GetDiscordUserServers(accessToken *string) ([]GetDiscordUserGuildsResponse, error) {
	req, err := http.NewRequest("GET", "https://discordapp.com/api/users/@me/guilds", nil)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+*accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get Discord user guilds: status "+resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var discordUserGuilds []GetDiscordUserGuildsResponse
	if err = json.Unmarshal(body, &discordUserGuilds); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return discordUserGuilds, nil
}

func AddUserToGuild(accessToken *string, guildID string, userID string, userRoles []string) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(map[string]interface{}{
		"access_token": *accessToken,
		"roles":        userRoles,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", "https://discord.com/api/guilds/"+guildID+"/members/"+userID, &buf)
	if err != nil {
		return err
	}

	botToken, err := util.GetEnvOrErr("DISCORD_BOT_TOKEN")
	if err != nil {
		return err

	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return apperrors.NewAlreadyInGuildError()
	}
	if resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return errors.New("Failed to add user to guild: status " + resp.Status + " body: " + string(body))
	}

	return nil
}

// AddRoleToDiscordUser PUT guilds/{guild.id}/members/{user.id}/roles/{role.id}
// Adds a role to a guild member. Requires the MANAGE_ROLES permission. Returns a 204 empty response on success. Fires a Guild Member Update Gateway event.
func AddRoleToDiscordUser(guildID string, userID string, roleID string) error {
	req, err := http.NewRequest("PUT", "https://discord.com/api/guilds/"+guildID+"/members/"+userID+"/roles/"+roleID, nil)
	if err != nil {
		return err
	}

	botToken, err := util.GetEnvOrErr("DISCORD_BOT_TOKEN")
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return errors.New("Failed to add user role: status " + resp.Status + " body: " + string(body))
	}

	return nil
}

// CreateDiscordMessage POST /channels/{channel.id}/messages
func CreateDiscordMessage(channelID string, content string) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(map[string]string{
		"content": content,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://discord.com/api/channels/"+channelID+"/messages", &buf)
	if err != nil {
		return err
	}

	botToken, err := util.GetEnvOrErr("DISCORD_BOT_TOKEN")
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return errors.New("Failed to create message: status " + resp.Status + " body: " + string(body))
	}

	return nil
}

func GetDiscordUserToken(params models.OauthCallbackParams) (*TokenResponse, error) {
	clientID, err := util.GetEnvOrErr("DISCORD_CLIENT_ID")
	if err != nil {
		return nil, err
	}
	clientSecret, err := util.GetEnvOrErr("DISCORD_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}
	clientRedirectURI, err := util.GetEnvOrErr("DISCORD_CLIENT_REDIRECT_URI")
	if err != nil {
		return nil, err
	}

	formData := url.Values{}
	formData.Set("client_id", clientID)
	formData.Set("client_secret", clientSecret)
	formData.Set("grant_type", "authorization_code")
	formData.Set("code", params.Code)
	formData.Set("redirect_uri", clientRedirectURI)
	formData.Set("scope", "identify")

	req, err := http.NewRequest("POST", "https://discordapp.com/api/oauth2/token", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Discord OAuth failed: status: " + resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tokenResponse TokenResponse
	if err = json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

func GetDiscordUserRole(discordUserID, guildID, adminRoleID string) (enum.UserRole, error) {
	botToken, err := util.GetEnvOrErr("DISCORD_BOT_TOKEN")
	if err != nil {
		return enum.User, err
	}

	u := "https://discord.com/api/guilds/" + guildID + "/members/" + discordUserID
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return enum.User, err
	}
	req.Header.Set("Authorization", "Bot "+botToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return enum.User, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return enum.User, errors.New("failed to get user roles: status " + resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return enum.User, err
	}

	var member struct {
		Roles []string `json:"roles"`
	}
	if err := json.Unmarshal(body, &member); err != nil {
		return enum.User, err
	}

	for _, roleID := range member.Roles {
		if roleID == adminRoleID {
			return enum.Admin, nil
		}
	}

	return enum.User, nil
}

func GetDiscordOAuthRedirectURI() (string, error) {
	clientID, err := util.GetEnvOrErr("DISCORD_CLIENT_ID")
	if err != nil {
		return "", err
	}
	clientRedirectURI, err := util.GetEnvOrErr("DISCORD_CLIENT_REDIRECT_URI")
	if err != nil {
		return "", err
	}

	discordOAuthBaseURL := "https://discord.com/oauth2/authorize"
	u, err := url.Parse(discordOAuthBaseURL)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Add("client_id", clientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", clientRedirectURI)
	scopes := "identify guilds guilds.join"
	params.Add("scope", scopes)
	u.RawQuery = params.Encode()

	return u.String(), nil
}
