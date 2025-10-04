package handler

import (
	"mime/multipart"
	"net/http"

	"github.com/labstack/echo/v4"
)

// extractFileFromForm extracts a file from a multipart form.
// If the file does not exist, it returns (nil, nil).
func extractFileFromForm(c echo.Context, formFileName string) (multipart.File, error) {
	fileHeader, err := c.FormFile(formFileName)
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, nil
		}

		return nil, err
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}

	return file, nil
}
