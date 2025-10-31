package bss

const (
	StatusUnapproved int = 1
	StatusApproved   int = 2
	StatusCancelled  int = 3
)

type Operation struct {
	OperationNr int    `json:"operationNr"`
	MarkingNr   int    `json:"markingNr"`
	MarkingCode string `json:"markingCode"`
	MarkingName string `json:"markingName"`
	ValidFrom   string `json:"validFrom"`
	ValidTo     string `json:"validTo"`
	StatusCfgNr int    `json:"statusCfgNr"`
}

type Response struct {
	TotalCount int         `json:"totalCount"`
	PageSize   int         `json:"pageSize"`
	PageNumber int         `json:"pageNumber"`
	PageCount  int         `json:"pageCount"`
	Data       []Operation `json:"data"`
}
