package repository

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/oapi-codegen/runtime/types"
	"github.com/traPtitech/game3-back/internal/pkg/util"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Patch(tableName string, idName string, idValue interface{}, patchStruct interface{}) error {
	query := fmt.Sprintf("UPDATE %s SET ", tableName)
	params := []interface{}{}

	v := reflect.ValueOf(patchStruct).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() == reflect.Pointer && !field.IsNil() {
			dbTag := util.ToSnakeCase(t.Field(i).Name)
			var paramValue interface{}

			// *types.File 型のフィールドを特別に処理
			if _, ok := field.Interface().(*types.File); ok {
				fileField := field.Interface().(*types.File)
				imageData, err := fileField.Bytes()
				if err != nil {
					return err
				}
				paramValue = imageData
			} else {
				// *types.File 以外のフィールドはそのまま
				paramValue = field.Elem().Interface()
			}

			query += fmt.Sprintf("%s = ?, ", dbTag)
			params = append(params, paramValue)
		}
	}

	query = strings.TrimSuffix(query, ", ") + fmt.Sprintf(" WHERE %s = ?", idName)
	params = append(params, idValue)

	if _, err := r.db.Exec(query, params...); err != nil {
		return err
	}

	return nil
}
