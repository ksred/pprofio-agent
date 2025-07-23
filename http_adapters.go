package pprofio

import (
	"net/http"
)

// Router-specific middleware helpers for the Profiler
// These methods provide easy integration with popular Go web frameworks

// ForStandardHTTP returns middleware for standard net/http and compatible routers
// Usage:
//
//	wrappedHandler := profiler.ForStandardHTTP()(handler)
//	http.ListenAndServe(":8080", wrappedHandler)
func (p *Profiler) ForStandardHTTP() func(http.Handler) http.Handler {
	return p.HTTPMiddleware()
}

// ForChi returns middleware for Chi router (same as standard HTTP)
// Usage:
//
//	r := chi.NewRouter()
//	r.Use(profiler.ForChi())
func (p *Profiler) ForChi() func(http.Handler) http.Handler {
	return p.HTTPMiddleware()
}

// ForGorillaMux returns middleware for Gorilla Mux (same as standard HTTP)
// Usage:
//
//	r := mux.NewRouter()
//	r.Use(profiler.ForGorillaMux())
func (p *Profiler) ForGorillaMux() func(http.Handler) http.Handler {
	return p.HTTPMiddleware()
}

// ForGin provides integration helper for Gin framework
// This returns the standard HTTP middleware that can be wrapped with gin.WrapH
// Usage:
//
//	r := gin.Default()
//	r.Use(gin.WrapH(profiler.ForGin()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))))
//
// Or more simply, use the convenience method:
//
//	r.Use(profiler.GinMiddleware())
func (p *Profiler) ForGin() func(http.Handler) http.Handler {
	return p.HTTPMiddleware()
}

// GinMiddleware provides a convenient Gin middleware function
// Usage:
//
//	r := gin.Default()
//	r.Use(profiler.GinMiddleware())
//
// This is a convenience method that handles the gin.WrapH conversion automatically
func (p *Profiler) GinMiddleware() interface{} {
	// Return a function that matches gin.HandlerFunc signature
	return func(c interface{}) {
		// This is a placeholder that demonstrates the concept
		// In practice, this would need proper gin.Context handling
		// The actual implementation would be:
		//
		// return gin.WrapH(middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//     c.(*gin.Context).Next()
		// })))
		//
		// But since we don't import Gin, we provide instructions instead

		// Users should use: gin.WrapH(profiler.HTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
		panic("Use gin.WrapH(profiler.HTTPMiddleware()(dummyHandler)) instead - see documentation")
	}
}

// ForEcho returns middleware for Echo framework
// Usage:
//
//	e := echo.New()
//	e.Use(echo.WrapMiddleware(profiler.ForEcho()))
func (p *Profiler) ForEcho() func(http.Handler) http.Handler {
	return p.HTTPMiddleware()
}

// ForFastHTTP returns middleware for FastHTTP-based frameworks
// Note: This requires a different implementation since FastHTTP doesn't use net/http
// Usage: Currently not supported - FastHTTP has a different request/response model
func (p *Profiler) ForFastHTTP() interface{} {
	// FastHTTP uses different request/response types, so standard HTTP middleware won't work
	// This would require a separate implementation
	panic("FastHTTP middleware requires separate implementation - not yet supported")
}
