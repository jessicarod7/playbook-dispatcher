package controllers

import (
	"fmt"
	"net/http"
	"playbook-dispatcher/internal/api/middleware"
	dbModel "playbook-dispatcher/internal/common/model/db"
	"strings"

	"github.com/labstack/echo/v4"
	identityMiddleware "github.com/redhatinsights/platform-go-middlewares/identity"
)

// these functions should not be needed - the generated code should fill in default values from the schema
func getLimit(params ApiRunsListParams) int {
	if params.Limit != nil {
		return (int(*params.Limit))
	}

	return 50
}

func getOffset(params ApiRunsListParams) int {
	if params.Offset != nil {
		return int(*params.Offset)
	}

	return 0
}

func getOrderBy(params ApiRunsListParams) string {
	if params.SortBy == nil || len(*params.SortBy) == 0 {
		return "created_at desc"
	}

	if parts := strings.Split(string(*params.SortBy), ":"); len(parts) == 1 {
		return fmt.Sprintf("%s %s", parts[0], "desc")
	} else {
		return fmt.Sprintf("%s %s", parts[0], parts[1])
	}
}

func (this *controllers) ApiRunsList(ctx echo.Context, params ApiRunsListParams) error {
	var dbRuns []dbModel.Run

	identity := identityMiddleware.Get(ctx.Request().Context())

	queryBuilder := this.database.Where(&dbModel.Run{
		Account: identity.Identity.AccountNumber,
	})

	if params.Filter != nil {
		if params.Filter.Status != nil {
			status := *params.Filter.Status
			switch status {
			case "timeout": // TODO
				queryBuilder.Where("runs.created_at + runs.timeout * interval '1 second' <= NOW()")
				status = "running" // TODO
			case "running": // TODO
				queryBuilder.Where("runs.created_at + runs.timeout * interval '1 second' > NOW()")
			}

			queryBuilder.Where("runs.status = ?", status)
		}

		if params.Filter.Recipient != nil {
			queryBuilder.Where("runs.recipient = ?", *params.Filter.Recipient)
		}
	}

	if labelFilters := middleware.GetLabelFilters(ctx); len(labelFilters) > 0 {
		for key, value := range labelFilters {
			queryBuilder.Where("runs.labels ->> ? = ?", key, value)
		}
	}

	queryBuilder.Order(getOrderBy(params))
	queryBuilder.Order("id") // secondary criteria to guarantee stable sorting

	queryBuilder.Limit(getLimit(params))
	queryBuilder.Offset(getOffset(params))

	dbResult := queryBuilder.Find(&dbRuns)

	if dbResult.Error != nil {
		this.log.Error(dbResult.Error)
		return ctx.NoContent(http.StatusInternalServerError)
	}

	response := make([]Run, len(dbRuns))

	for i, v := range dbRuns {
		response[i] = *dbRuntoApiRun(&v)
	}

	return ctx.JSON(http.StatusOK, &Runs{
		Data: response,
		Meta: Meta{
			Count: len(response),
		},
	})
}
