package util

import (
	customErr "Jobly/internal/errors"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"net/http"
)

func HandleValidationError(ctx *gin.Context, err error) bool {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		validationDetails := make([]map[string]string, len(ve))
		for i, fe := range ve {
			validationDetails[i] = map[string]string{
				"field":   fe.Field(),
				"message": fmt.Sprintf("%s is %s", fe.Field(), fe.Tag()),
			}
		}

		customErr.WriteError(ctx, customErr.NewErrorWithMeta(
			customErr.VALIDATION_ERROR,
			"Invalid request body",
			http.StatusBadRequest,
			err,
			map[string]any{"details": validationDetails},
		))
		return true
	}
	return false
}
