//go:build !windows && !darwin

package pathdialog

func show(request Request) (Result, error) {
	return unavailable(request)
}
