package chatapi

type Comments struct {
	requestsWithComments map[string]BurpRequestResponse
}

func NewComments() Comments {
	return Comments{
		make(map[string]BurpRequestResponse),
	}
}

func (c *Comments) getRequestWithComments(key string) BurpRequestResponse {
	return c.requestsWithComments[key]
}

func (c *Comments) setRequestWithComments(key string, burpReqRespWithComments BurpRequestResponse) {
	c.requestsWithComments[key] = burpReqRespWithComments
}
