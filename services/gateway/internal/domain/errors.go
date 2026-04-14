package domain

import "github.com/base-go/base/pkg/apperror"

// Domain-specific errors cho Gateway Service.
var (
	ErrRouteNotFound     = apperror.NotFound("no matching route found for this request")
	ErrMethodNotAllowed  = apperror.BadRequest("method not allowed for this route")
	ErrUpstreamUnavail   = apperror.ServiceUnavailable("upstream service is unavailable")
	ErrUpstreamTimeout   = apperror.ServiceUnavailable("upstream service timed out")
)
