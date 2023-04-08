package middleware

import (
	"net/http"

	"github.com/alextanhongpin/go-core-microservice/http/types"
	"github.com/rs/xid"
)

var RequestIDContext types.ContextKey[string] = "request_id_ctx"

func RequestID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		requestID := xid.New().String()
		ctx := r.Context()
		ctx = RequestIDContext.WithValue(ctx, requestID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
