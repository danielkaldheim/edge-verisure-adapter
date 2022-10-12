package verisure

import "github.com/thingsplex/verisure/model"

type GraphQLQuery struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

type GraphQLResponse struct {
	Errors []*model.Errors `json:"errors"`
	Data   *model.Data     `json:"data"`
}
