package response

type Standard struct {
	Error    *ErrorPayload    `json:"error,omitempty"`
	Response *ResponsePayload `json:"response,omitempty"`
	Data     *DataPayload     `json:"data,omitempty"`
}

type ErrorPayload struct {
	Code int    `json:"code"`
	Text string `json:"text"`
}

type ResponsePayload map[string]interface{}

type DataPayload map[string]interface{}
