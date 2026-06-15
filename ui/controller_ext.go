package ui

// ExtendedController extends Controller with additional navigation methods.
// The AppController in controller.go will implement these.
type ExtendedController interface {
	Controller
	// No additional methods needed - tabs are managed by Dashboard itself
}
