package chatapi

import "github.com/valyala/fasthttp"

func AuthPackage(ctx *fasthttp.RequestCtx, hub *Hub) {
	if !hub.authPackageDownloaded {

	} else {
		ctx.Error("Auth package already downloaded", fasthttp.StatusBadRequest)
	}
}
